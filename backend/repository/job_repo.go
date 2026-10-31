package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/pulseforge/pulseforge/backend/models"
)

// JobRepository provides persistence operations for Job entities.
type JobRepository struct {
	db *gorm.DB
}

// NewJobRepository creates a new instance of JobRepository.
func NewJobRepository(db *gorm.DB) *JobRepository {
	return &JobRepository{db: db}
}

// Create persists a single Job record.
func (r *JobRepository) Create(ctx context.Context, job *models.Job) error {
	if job == nil {
		return fmt.Errorf("job is nil")
	}
	if err := r.db.WithContext(ctx).Create(job).Error; err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}
	return nil
}

// BulkCreate inserts many Job records in batches.
// The batchSize parameter controls how many rows are inserted per DB round‑trip.
// If batchSize <= 0 a sensible default of 1000 is used.
func (r *JobRepository) BulkCreate(ctx context.Context, jobs []*models.Job, batchSize int) error {
	if len(jobs) == 0 {
		return nil
	}
	if batchSize <= 0 {
		batchSize = 1000
	}
	if err := r.db.WithContext(ctx).CreateInBatches(jobs, batchSize).Error; err != nil {
		return fmt.Errorf("failed to bulk insert jobs: %w", err)
	}
	return nil
}

// GetByID retrieves a Job by its primary key.
func (r *JobRepository) GetByID(ctx context.Context, id uint) (*models.Job, error) {
	var job models.Job
	if err := r.db.WithContext(ctx).First(&job, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to fetch job id %d: %w", id, err)
	}
	return &job, nil
}

// ListByPipeline returns a slice of Jobs belonging to a specific pipeline.
// Results are ordered by creation time descending.
// limit and offset control pagination; if limit <= 0 all matching rows are returned.
func (r *JobRepository) ListByPipeline(ctx context.Context, pipelineID uint, limit, offset int) ([]*models.Job, error) {
	query := r.db.WithContext(ctx).
		Where("pipeline_id = ?", pipelineID).
		Order("created_at DESC").
		Offset(offset)

	if limit > 0 {
		query = query.Limit(limit)
	}

	var jobs []*models.Job
	if err := query.Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("failed to list jobs for pipeline %d: %w", pipelineID, err)
	}
	return jobs, nil
}

// ListRecentFailed returns the most recent jobs that ended with a failure status.
// limit caps the number of rows returned; if limit <= 0 a default of 50 is used.
func (r *JobRepository) ListRecentFailed(ctx context.Context, limit int) ([]*models.Job, error) {
	if limit <= 0 {
		limit = 50
	}
	var jobs []*models.Job
	if err := r.db.WithContext(ctx).
		Where("status = ?", models.JobStatusFailed).
		Order("created_at DESC").
		Limit(limit).
		Find(&jobs).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch recent failed jobs: %w", err)
	}
	return jobs, nil
}

// DeleteOlderThan removes jobs older than the supplied timestamp.
// It returns the number of rows deleted.
func (r *JobRepository) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("created_at < ?", before).
		Delete(&models.Job{})
	if result.Error != nil {
		return 0, fmt.Errorf("failed to delete old jobs: %w", result.Error)
	}
	return result.RowsAffected, nil
}