package controller

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pulseforge/backend/logger"
	"github.com/pulseforge/backend/models"
	"github.com/pulseforge/backend/repository"
	"github.com/pulseforge/backend/service"
)

// WebhookController handles incoming CI/CD webhook events.
type WebhookController struct {
	pipelineRepo   repository.PipelineRepository
	jobRepo        repository.JobRepository
	queuePublisher service.QueuePublisher
	log            *logger.Logger
}

// NewWebhookController creates a new instance of WebhookController.
func NewWebhookController(
	pipelineRepo repository.PipelineRepository,
	jobRepo repository.JobRepository,
	queuePublisher service.QueuePublisher,
	log *logger.Logger,
) *WebhookController {
	return &WebhookController{
		pipelineRepo:   pipelineRepo,
		jobRepo:        jobRepo,
		queuePublisher: queuePublisher,
		log:            log,
	}
}

// RegisterRoutes registers the webhook endpoint on the provided router.
func (wc *WebhookController) RegisterRoutes(r *gin.Engine) {
	// The HMAC middleware is expected to be attached upstream.
	r.POST("/webhook/:provider", wc.HandleWebhook)
}

// HandleWebhook processes a webhook POST request, validates the payload,
// stores the job record and enqueues it for asynchronous AI analysis.
func (wc *WebhookController) HandleWebhook(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "provider missing in URL"})
		return
	}

	// Read raw body – needed for logging / future processing.
	body, err := c.GetRawData()
	if err != nil {
		wc.log.Error("failed to read request body", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	var (
		job      *models.Job
		pipeline *models.Pipeline
	)

	switch provider {
	case "github":
		job, pipeline, err = wc.handleGitHub(body)
	case "gitlab":
		job, pipeline, err = wc.handleGitLab(body)
	case "circleci":
		job, pipeline, err = wc.handleCircleCI(body)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported provider"})
		return
	}
	if err != nil {
		wc.log.Error("payload processing error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if pipeline == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "pipeline not registered"})
		return
	}

	// Persist the job.
	if err := wc.jobRepo.Create(job); err != nil {
		wc.log.Error("failed to store job", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not store job"})
		return
	}

	// Enqueue for AI analysis – non‑blocking, log on failure but do not fail the request.
	if err := wc.queuePublisher.EnqueueJob(job.ID); err != nil {
		wc.log.Error("failed to enqueue job for AI analysis", err)
	}

	c.JSON(http.StatusAccepted, gin.H{
		"status": "queued",
		"job_id": job.ID,
	})
}

// ---------------------------------------------------------------------------
// Provider‑specific payload handling
// ---------------------------------------------------------------------------

func (wc *WebhookController) handleGitHub(payload []byte) (*models.Job, *models.Pipeline, error) {
	var gh struct {
		Repository struct {
			FullName string `json:"full_name"` // e.g. "org/repo"
		} `json:"repository"`
		WorkflowJob struct {
			ID          int64  `json:"id"`
			Name        string `json:"name"`
			Status      string `json:"status"`      // queued, in_progress, completed
			Conclusion  string `json:"conclusion"`  // success, failure, cancelled, neutral, timed_out, action_required
			StartedAt   string `json:"started_at"`  // RFC3339
			CompletedAt string `json:"completed_at"` // RFC3339
			// The raw log is not part of the webhook; it will be fetched later if needed.
		} `json:"workflow_job"`
	}
	if err := json.Unmarshal(payload, &gh); err != nil {
		return nil, nil, err
	}

	// Locate the pipeline configuration.
	pipeline, err := wc.pipelineRepo.GetByProviderAndRepo("github", gh.Repository.FullName)
	if err != nil {
		return nil, nil, err
	}

	status := mapGitHubStatus(gh.WorkflowJob.Status, gh.WorkflowJob.Conclusion)
	duration := computeDuration(gh.WorkflowJob.StartedAt, gh.WorkflowJob.CompletedAt)

	job := &models.Job{
		PipelineID:   pipeline.ID,
		ExternalJobID: strconv.FormatInt(gh.WorkflowJob.ID, 10),
		Name:         gh.WorkflowJob.Name,
		Status:       status,
		Duration:     duration,
		Log:          "", // placeholder – actual log can be fetched asynchronously.
		CreatedAt:    time.Now(),
	}
	return job, pipeline, nil
}

func (wc *WebhookController) handleGitLab(payload []byte) (*models.Job, *models.Pipeline, error) {
	var gl struct {
		Project struct {
			PathWithNamespace string `json:"path_with_namespace"` // e.g. "org/repo"
		} `json:"project"`
		ObjectAttributes struct {
			ID          int64  `json:"id"`
			Name        string `json:"name"`
			Status      string `json:"status"`      // created, pending, running, success, failed, canceled
			CreatedAt   string `json:"created_at"` // ISO8601
			StartedAt   string `json:"started_at"` // ISO8601
			FinishedAt  string `json:"finished_at"` // ISO8601
			Duration    float64 `json:"duration"`   // seconds (optional)
		} `json:"object_attributes"`
	}
	if err := json.Unmarshal(payload, &gl); err != nil {
		return nil, nil, err
	}

	pipeline, err := wc.pipelineRepo.GetByProviderAndRepo("gitlab", gl.Project.PathWithNamespace)
	if err != nil {
		return nil, nil, err
	}

	status := mapGitLabStatus(gl.ObjectAttributes.Status)
	duration := int64(gl.ObjectAttributes.Duration)
	if duration == 0 && gl.ObjectAttributes.StartedAt != "" && gl.ObjectAttributes.FinishedAt != "" {
		duration = computeDuration(gl.ObjectAttributes.StartedAt, gl.ObjectAttributes.FinishedAt)
	}

	job := &models.Job{
		PipelineID:   pipeline.ID,
		ExternalJobID: strconv.FormatInt(gl.ObjectAttributes.ID, 10),
		Name:         gl.ObjectAttributes.Name,
		Status:       status,
		Duration:    duration,
		Log:         "", // to be fetched later
		CreatedAt:   time.Now(),
	}
	return job, pipeline, nil
}

func (wc *WebhookController) handleCircleCI(payload []byte) (*models.Job, *models.Pipeline, error) {
	var cc struct {
		Pipeline struct {
			Number int64  `json:"number"`
			VCS struct {
				Provider string `json:"provider"` // "github" or "bitbucket"
				Slug     string `json:"slug"`     // "org/repo"
			} `json:"vcs"`
		} `json:"pipeline"`
		Job struct {
			ID          string `json:"id"` // UUID
			Name        string `json:"name"`
			Status      string `json:"status"` // "running", "success", "failed", "canceled", "not_run"
			StartedAt   string `json:"started_at"` // RFC3339
			StoppedAt   string `json:"stopped_at"` // RFC3339
			Duration    int64  `json:"duration"`   // seconds (optional)
		} `json:"job"`
	}
	if err := json.Unmarshal(payload, &cc); err != nil {
		return nil, nil, err
	}

	// CircleCI does not embed a secret; we match on the VCS slug.
	pipeline, err := wc.pipelineRepo.GetByProviderAndRepo("circleci", cc.Pipeline.VCS.Slug)
	if err != nil {
		return nil, nil, err
	}

	status := mapCircleCIStatus(cc.Job.Status)
	duration := cc.Job.Duration
	if duration == 0 && cc.Job.StartedAt != "" && cc.Job.StoppedAt != "" {
		duration = computeDuration(cc.Job.StartedAt, cc.Job.StoppedAt)
	}

	job := &models.Job{
		PipelineID:   pipeline.ID,
		ExternalJobID: cc.Job.ID,
		Name:         cc.Job.Name,
		Status:       status,
		Duration:    duration,
		Log:         "", // to be fetched later
		CreatedAt:   time.Now(),
	}
	return job, pipeline, nil
}

// ---------------------------------------------------------------------------
// Helper utilities
// ---------------------------------------------------------------------------

func computeDuration(startStr, endStr string) int64 {
	if startStr == "" || endStr == "" {
		return 0
	}
	start, err1 := time.Parse(time.RFC3339, startStr)
	end, err2 := time.Parse(time.RFC3339, endStr)
	if err1 != nil || err2 != nil {
		return 0
	}
	return int64(end.Sub(start).Seconds())
}

// GitHub status mapping.
func mapGitHubStatus(state, conclusion string) string {
	if state == "completed" {
		switch conclusion {
		case "success":
			return "success"
		case "failure", "timed_out", "action_required":
			return "failed"
		case "cancelled":
			return "canceled"
		default:
			return "unknown"
		}
	}
	// in_progress, queued, etc.
	switch state {
	case "in_progress":
		return "running"
	case "queued":
		return "queued"
	default:
		return "unknown"
	}
}

// GitLab status mapping.
func mapGitLabStatus(state string) string {
	switch state {
	case "created", "pending":
		return "queued"
	case "running":
		return "running"
	case "success":
		return "success"
	case "failed":
		return "failed"
	case "canceled":
		return "canceled"
	default:
		return "unknown"
	}
}

// CircleCI status mapping.
func mapCircleCIStatus(state string) string {
	switch state {
	case "running":
		return "running"
	case "success":
		return "success"
	case "failed":
		return "failed"
	case "canceled", "not_run":
		return "canceled"
	default:
		return "unknown"
	}
}