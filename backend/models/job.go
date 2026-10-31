package models

import (
	"time"

	"gorm.io/gorm"
)

// JobStatus represents the execution state of a CI/CD job.
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusSuccess   JobStatus = "success"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// Job stores a single CI/CD job execution record.
// It contains a short excerpt of the job log, execution status, duration,
// and an optional link to an AI‑generated suggestion.
type Job struct {
	// Primary key.
	ID uint `gorm:"primaryKey" json:"id"`

	// Foreign key to the owning pipeline.
	PipelineID uint `gorm:"not null;index" json:"pipeline_id"`

	// The CI provider that produced the job (e.g., github, gitlab, circleci).
	Provider string `gorm:"type:varchar(50);not null" json:"provider"`

	// The identifier of the job as reported by the provider.
	// Unique per provider to avoid duplicate ingestion.
	ProviderJobID string `gorm:"type:varchar(255);not null;uniqueIndex:idx_provider_job" json:"provider_job_id"`

	// Current execution status.
	Status JobStatus `gorm:"type:varchar(20);not null;index" json:"status"`

	// Duration of the job execution.
	// Stored as nanoseconds to preserve precision.
	Duration time.Duration `gorm:"type:bigint;not null" json:"duration"`

	// A short excerpt (first few hundred bytes) of the raw job log.
	// Useful for quick UI previews and for feeding the AI engine.
	LogExcerpt string `gorm:"type:text" json:"log_excerpt,omitempty"`

	// Optional foreign key to an AI suggestion generated for this job.
	AISuggestionID *uint `gorm:"index" json:"ai_suggestion_id,omitempty"`

	// The AI suggestion entity. Populated with Preload when needed.
	AISuggestion *AISuggestion `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"ai_suggestion,omitempty"`

	// Timestamps.
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // soft delete
}

// TableName overrides the default table name used by GORM.
func (Job) TableName() string {
	return "jobs"
}

// DurationSeconds returns the job duration expressed in seconds with millisecond precision.
func (j *Job) DurationSeconds() float64 {
	return float64(j.Duration) / float64(time.Second)
}