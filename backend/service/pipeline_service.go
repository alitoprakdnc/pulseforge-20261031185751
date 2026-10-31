package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/streadway/amqp"
	"go.uber.org/zap"

	"github.com/yourorg/pulseforge/backend/models"
	"github.com/yourorg/pulseforge/backend/repository"
)

// PipelineService encapsulates the business logic for handling CI/CD webhook
// ingestion, normalising the payloads and triggering asynchronous AI analysis.
type PipelineService struct {
	pipelineRepo repository.PipelineRepository
	jobRepo      repository.JobRepository
	amqpChannel  *amqp.Channel
	logger       *zap.Logger
}

// NewPipelineService creates a new PipelineService instance.
func NewPipelineService(
	pipelineRepo repository.PipelineRepository,
	jobRepo repository.JobRepository,
	amqpChannel *amqp.Channel,
	logger *zap.Logger,
) *PipelineService {
	return &PipelineService{
		pipelineRepo: pipelineRepo,
		jobRepo:      jobRepo,
		amqpChannel:  amqpChannel,
		logger:       logger,
	}
}

// IngestWebhook processes an incoming webhook payload.
//
//   provider   – one of "github", "gitlab", "circleci"
//   pipelineID – internal DB identifier of the pipeline that owns the webhook
//   payload    – raw request body (already validated for HMAC by middleware)
func (s *PipelineService) IngestWebhook(ctx context.Context, provider string, pipelineID int64, payload []byte) error {
	// Ensure the pipeline exists and fetch its metadata.
	p, err := s.pipelineRepo.GetByID(ctx, pipelineID)
	if err != nil {
		return fmt.Errorf("pipeline lookup failed: %w", err)
	}

	// Normalise the provider‑specific payload into a slice of Job models.
	jobs, err := s.parsePayload(provider, payload, p.ID)
	if err != nil {
		return fmt.Errorf("payload parsing failed: %w", err)
	}
	if len(jobs) == 0 {
		s.logger.Info("no jobs extracted from webhook", zap.String("provider", provider), zap.Int64("pipeline_id", p.ID))
		return nil
	}

	// Persist each job and enqueue it for AI analysis.
	for _, job := range jobs {
		if err := s.jobRepo.Create(ctx, job); err != nil {
			s.logger.Error("failed to store job", zap.Error(err), zap.Int64("pipeline_id", p.ID), zap.String("external_id", job.ExternalID))
			continue
		}
		if err := s.enqueueJob(job.ID); err != nil {
			s.logger.Error("failed to enqueue AI analysis job", zap.Error(err), zap.Int64("job_id", job.ID))
			// Not a fatal error – the job is stored; analysis can be retried later.
		}
	}
	return nil
}

// parsePayload converts a raw webhook payload into a slice of Job models.
// It returns an error if the provider is unknown or the payload cannot be unmarshalled.
func (s *PipelineService) parsePayload(provider string, payload []byte, pipelineID uint) ([]*models.Job, error) {
	switch provider {
	case "github":
		return s.parseGitHubPayload(payload, pipelineID)
	case "gitlab":
		return s.parseGitLabPayload(payload, pipelineID)
	case "circleci":
		return s.parseCircleCIPayload(payload, pipelineID)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// parseGitHubPayload extracts job information from a GitHub Actions webhook.
// The implementation supports the `workflow_job` event which contains a single job.
// For other event types we simply ignore them (return empty slice).
func (s *PipelineService) parseGitHubPayload(payload []byte, pipelineID uint) ([]*models.Job, error) {
	type githubJob struct {
		ID          int64  `json:"id"`
		Name        string `json:"name"`
		Status      string `json:"status"`      // queued, in_progress, completed
		Conclusion  string `json:"conclusion"`  // success, failure, cancelled, etc.
		RunStartedAt string `json:"run_started_at"` // ISO8601 timestamp
		CompletedAt string `json:"completed_at"`   // ISO8601 timestamp (optional)
		Steps       []struct {
			Name    string `json:"name"`
			Conclusion string `json:"conclusion"`
			Output   struct {
				Summary string `json:"summary"`
			} `json:"output"`
		} `json:"steps"`
	}
	var event struct {
		Action string    `json:"action"` // e.g., "queued", "completed"
		WorkflowJob githubJob `json:"workflow_job"`
	}
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("github payload unmarshal: %w", err)
	}
	// If the webhook does not contain a workflow_job, ignore.
	if event.WorkflowJob.ID == 0 {
		return []*models.Job{}, nil
	}
	job := &models.Job{
		PipelineID: pipelineID,
		Provider:   "github",
		ExternalID: fmt.Sprintf("%d", event.WorkflowJob.ID),
		Name:       event.WorkflowJob.Name,
		Status:     mapGitHubStatus(event.WorkflowJob.Status, event.WorkflowJob.Conclusion),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	// Compute duration if both timestamps are present.
	if event.WorkflowJob.RunStartedAt != "" && event.WorkflowJob.CompletedAt != "" {
		start, err1 := time.Parse(time.RFC3339, event.WorkflowJob.RunStartedAt)
		end, err2 := time.Parse(time.RFC3339, event.WorkflowJob.CompletedAt)
		if err1 == nil && err2 == nil && end.After(start) {
			job.DurationSeconds = int64(end.Sub(start).Seconds())
		}
	}
	// Build a short log excerpt from step conclusions.
	if len(event.WorkflowJob.Steps) > 0 {
		excerpt := ""
		for i, step := range event.WorkflowJob.Steps {
			if i >= 5 {
				break
			}
			excerpt += fmt.Sprintf("%s: %s\n", step.Name, step.Conclusion)
		}
		job.LogExcerpt = excerpt
	}
	return []*models.Job{job}, nil
}

// parseGitLabPayload extracts job information from a GitLab CI webhook.
// GitLab sends a `pipeline` event that may contain multiple builds; we map each
// build as a separate Job record.
func (s *PipelineService) parseGitLabPayload(payload []byte, pipelineID uint) ([]*models.Job, error) {
	type gitlabBuild struct {
		ID          int64   `json:"id"`
		Name        string  `json:"name"`
		Status      string  `json:"status"` // created, pending, running, success, failed, canceled, skipped
		Duration    float64 `json:"duration"` // seconds, may be null
		CreatedAt   string  `json:"created_at"` // ISO8601
		StartedAt   string  `json:"started_at"` // ISO8601
		FinishedAt  string  `json:"finished_at"` // ISO8601
	}
	type gitlabPipeline struct {
		ObjectKind string `json:"object_kind"` // should be "pipeline"
		ObjectAttributes struct {
			ID int64 `json:"id"`
		} `json:"object_attributes"`
		Builds []gitlabBuild `json:"builds"`
	}
	var event gitlabPipeline
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("gitlab payload unmarshal: %w", err)
	}
	if event.ObjectKind != "pipeline" {
		return []*models.Job{}, nil
	}
	var jobs []*models.Job
	for _, b := range event.Builds {
		job := &models.Job{
			PipelineID: pipelineID,
			Provider:   "gitlab",
			ExternalID: fmt.Sprintf("%d", b.ID),
			Name:       b.Name,
			Status:     mapGitLabStatus(b.Status),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if b.Duration > 0 {
			job.DurationSeconds = int64(b.Duration)
		}
		// Build a tiny log excerpt from timestamps if available.
		if b.StartedAt != "" && b.FinishedAt != "" {
			job.LogExcerpt = fmt.Sprintf("started: %s, finished: %s", b.StartedAt, b.FinishedAt)
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// parseCircleCIPayload extracts job information from a CircleCI webhook.
// CircleCI sends a `pipeline` event that contains a list of workflow jobs.
func (s *PipelineService) parseCircleCIPayload(payload []byte, pipelineID uint) ([]*models.Job, error) {
	type circleJob struct {
		ID          string `json:"id"` // UUID
		Name        string `json:"name"`
		Status      string `json:"status"` // success, failed, canceled, etc.
		StartedAt   string `json:"started_at"` // ISO8601
		StoppedAt   string `json:"stopped_at"` // ISO8601
	}
	type circleWorkflow struct {
		Jobs []circleJob `json:"jobs"`
	}
	type circlePayload struct {
		Type      string          `json:"type"` // should be "pipeline"
		PipelineID string         `json:"pipeline_id"`
		Workflows []circleWorkflow `json:"workflows"`
	}
	var event circlePayload
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("circleci payload unmarshal: %w", err)
	}
	if event.Type != "pipeline" {
		return []*models.Job{}, nil
	}
	var jobs []*models.Job
	for _, wf := range event.Workflows {
		for _, j := range wf.Jobs {
			job := &models.Job{
				PipelineID: pipelineID,
				Provider:   "circleci",
				ExternalID: j.ID,
				Name:       j.Name,
				Status:     mapCircleCIStatus(j.Status),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if j.StartedAt != "" && j.StoppedAt != "" {
				start, err1 := time.Parse(time.RFC3339, j.StartedAt)
				end, err2 := time.Parse(time.RFC3339, j.StoppedAt)
				if err1 == nil && err2 == nil && end.After(start) {
					job.DurationSeconds = int64(end.Sub(start).Seconds())
				}
				job.LogExcerpt = fmt.Sprintf("started: %s, stopped: %s", j.StartedAt, j.StoppedAt)
			}
			jobs = append(jobs, job)
		}
	}
	return jobs, nil
}

// enqueueJob publishes a message to the "ai_analysis" RabbitMQ queue so that
// the background worker can fetch the job logs and generate AI suggestions.
func (s *PipelineService) enqueueJob(jobID int64) error {
	msg := struct {
		JobID int64 `json:"job_id"`
	}{
		JobID: jobID,
	}
	body, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal ai job message: %w", err)
	}
	err = s.amqpChannel.Publish(
		"",               // default exchange
		"ai_analysis",    // routing key / queue name
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("publish to ai_analysis queue: %w", err)
	}
	s.logger.Debug("enqueued job for AI analysis", zap.Int64("job_id", jobID))
	return nil
}

// Helper functions to map provider‑specific status strings to our canonical set.
func mapGitHubStatus(apiStatus, conclusion string) string {
	switch apiStatus {
	case "queued", "in_progress":
		return "running"
	case "completed":
		switch conclusion {
		case "success":
			return "passed"
		case "failure", "timed_out", "action_required":
			return "failed"
		case "cancelled":
			return "cancelled"
		default:
			return "unknown"
		}
	default:
		return "unknown"
	}
}

func mapGitLabStatus(apiStatus string) string {
	switch apiStatus {
	case "running", "pending":
		return "running"
	case "success":
		return "passed"
	case "failed":
		return "failed"
	case "canceled":
		return "cancelled"
	case "skipped":
		return "skipped"
	default:
		return "unknown"
	}
}

func mapCircleCIStatus(apiStatus string) string {
	switch apiStatus {
	case "running", "queued":
		return "running"
	case "success":
		return "passed"
	case "failed":
		return "failed"
	case "canceled":
		return "cancelled"
	default:
		return "unknown"
	}
}