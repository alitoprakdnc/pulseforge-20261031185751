package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/PulseForge/backend/models"
	"github.com/PulseForge/backend/repository"
	"github.com/PulseForge/backend/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// QueueMonitor periodically polls CI providers for pending job counts
// and updates the corresponding Pipeline records.
//
// It is intended to be run as a background goroutine:
//
//   ctx, cancel := context.WithCancel(context.Background())
//   defer cancel()
//   monitor := NewQueueMonitor(db, logger, 2*time.Minute)
//   go monitor.Start(ctx)
type QueueMonitor struct {
	db         *gorm.DB
	logger     *zap.Logger
	interval   time.Duration
	httpClient *http.Client
	pipelineRepo repository.PipelineRepository
}

// NewQueueMonitor creates a new QueueMonitor.
// interval defines how often the provider APIs are queried.
func NewQueueMonitor(db *gorm.DB, logger *zap.Logger, interval time.Duration) *QueueMonitor {
	return &QueueMonitor{
		db:           db,
		logger:       logger,
		interval:     interval,
		httpClient:   &http.Client{Timeout: 15 * time.Second},
		pipelineRepo: repository.NewPipelineRepository(db),
	}
}

// Start runs the monitor until the supplied context is cancelled.
func (qm *QueueMonitor) Start(ctx context.Context) {
	qm.logger.Info("queue monitor started", zap.Duration("interval", qm.interval))
	ticker := time.NewTicker(qm.interval)
	defer ticker.Stop()

	// Run once immediately
	qm.processAllPipelines(ctx)

	for {
		select {
		case <-ctx.Done():
			qm.logger.Info("queue monitor stopped")
			return
		case <-ticker.C:
			qm.processAllPipelines(ctx)
		}
	}
}

// processAllPipelines fetches all pipelines and updates their queue metrics.
func (qm *QueueMonitor) processAllPipelines(ctx context.Context) {
	pipelines, err := qm.pipelineRepo.ListAll(ctx)
	if err != nil {
		qm.logger.Error("failed to list pipelines", zap.Error(err))
		return
	}

	for _, p := range pipelines {
		qLen, err := qm.fetchQueueLength(ctx, &p)
		if err != nil {
			qm.logger.Warn("failed to fetch queue length",
				zap.Uint("pipeline_id", p.ID),
				zap.String("provider", p.Provider),
				zap.Error(err))
			continue
		}

		// Update the pipeline record with the new metric.
		update := map[string]interface{}{
			"queue_length":        qLen,
			"queue_last_checked": time.Now(),
		}
		if err := qm.db.Model(&models.Pipeline{}).Where("id = ?", p.ID).Updates(update).Error; err != nil {
			qm.logger.Error("failed to update pipeline queue metric",
				zap.Uint("pipeline_id", p.ID),
				zap.Error(err))
		} else {
			qm.logger.Debug("updated queue metric",
				zap.Uint("pipeline_id", p.ID),
				zap.Int("queue_length", qLen))
		}
	}
}

// fetchQueueLength dispatches to the provider‑specific implementation.
func (qm *QueueMonitor) fetchQueueLength(ctx context.Context, p *models.Pipeline) (int, error) {
	switch p.Provider {
	case "github":
		return qm.fetchGitHubQueue(ctx, p)
	case "gitlab":
		return qm.fetchGitLabQueue(ctx, p)
	case "circleci":
		return qm.fetchCircleCIQueue(ctx, p)
	default:
		return 0, fmt.Errorf("unsupported provider %s", p.Provider)
	}
}

// fetchGitHubQueue uses the GitHub GraphQL API to count queued workflow runs.
// It expects the pipeline to have RepoOwner and RepoName populated.
// The CI token is retrieved from Secret Manager using the secret name stored in p.CITokenSecret.
func (qm *QueueMonitor) fetchGitHubQueue(ctx context.Context, p *models.Pipeline) (int, error) {
	token, err := utils.GetSecret(ctx, p.CITokenSecret)
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve GitHub token: %w", err)
	}

	query := `
	query($owner:String!, $name:String!) {
	  repository(owner: $owner, name: $name) {
	    workflowRuns(first: 1, statuses: QUEUED) {
	      totalCount
	    }
	  }
	}`
	variables := map[string]interface{}{
		"owner": p.RepoOwner,
		"name":  p.RepoName,
	}
	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.github.com/graphql", utils.NewJSONReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := qm.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("github api returned %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			Repository struct {
				WorkflowRuns struct {
					TotalCount int `json:"totalCount"`
				} `json:"workflowRuns"`
			} `json:"repository"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	if len(result.Errors) > 0 {
		return 0, fmt.Errorf("github graphql errors: %v", result.Errors[0].Message)
	}
	return result.Data.Repository.WorkflowRuns.TotalCount, nil
}

// fetchGitLabQueue uses the GitLab REST API to count pending jobs.
// It expects p.CIProjectID to contain the URL‑encoded project ID (e.g. "group%2Fmy-repo").
func (qm *QueueMonitor) fetchGitLabQueue(ctx context.Context, p *models.Pipeline) (int, error) {
	token, err := utils.GetSecret(ctx, p.CITokenSecret)
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve GitLab token: %w", err)
	}

	// GitLab provides a jobs endpoint; we filter by scope=waiting.
	apiURL := fmt.Sprintf("https://gitlab.com/api/v4/projects/%s/jobs?scope=waiting", url.PathEscape(p.CIProjectID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("PRIVATE-TOKEN", token)

	resp, err := qm.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("gitlab api returned %d", resp.StatusCode)
	}

	var jobs []struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&jobs); err != nil {
		return 0, err
	}
	return len(jobs), nil
}

// fetchCircleCIQueue uses the CircleCI API to count pending workflows.
// It expects p.CIProjectSlug to be in the form "vcs-type/owner/repo" (e.g. "gh/pulseforge/pulseforge").
func (qm *QueueMonitor) fetchCircleCIQueue(ctx context.Context, p *models.Pipeline) (int, error) {
	token, err := utils.GetSecret(ctx, p.CITokenSecret)
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve CircleCI token: %w", err)
	}

	// CircleCI v2 API: GET /pipeline?org-slug=...&page-token=...
	// The endpoint does not expose a direct queue length, so we approximate by counting
	// workflows with status "queued" for the recent pipelines.
	apiURL := fmt.Sprintf("https://circleci.com/api/v2/pipeline/%s?branch=main", p.CIProjectSlug)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Circle-Token", token)

	resp, err := qm.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("circleci api returned %d", resp.StatusCode)
	}

	var result struct {
		Items []struct {
			ID     string `json:"id"`
			Number int    `json:"number"`
			State  string `json:"state"` // "created", "queued", "running", "success", ...
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	queueCount := 0
	for _, item := range result.Items {
		if item.State == "queued" || item.State == "created" {
			queueCount++
		}
	}
	return queueCount, nil
}