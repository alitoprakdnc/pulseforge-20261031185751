package repository

import (
	"context"
	"errors"

	"github.com/yourorg/pulseforge/backend/models"
	"gorm.io/gorm"
)

// ErrAISuggestionNotFound is returned when a suggestion cannot be located.
var ErrAISuggestionNotFound = errors.New("ai suggestion not found")

// AISuggestionRepository defines the contract for persisting and retrieving AI suggestions.
type AISuggestionRepository interface {
	// Create stores a new AI suggestion.
	Create(ctx context.Context, suggestion *models.AISuggestion) error
	// GetByID fetches a suggestion by its primary key.
	GetByID(ctx context.Context, id uint) (*models.AISuggestion, error)
	// ListByJobID returns suggestions linked to a specific job, ordered by newest first.
	// limit and offset are used for pagination; passing 0 for limit returns all records.
	ListByJobID(ctx context.Context, jobID uint, limit, offset int) ([]*models.AISuggestion, error)
	// DeleteByID removes a suggestion from the datastore.
	DeleteByID(ctx context.Context, id uint) error
}

// aiSuggestionRepo is the concrete GORM implementation of AISuggestionRepository.
type aiSuggestionRepo struct {
	db *gorm.DB
}

// NewAISuggestionRepository creates a new repository instance bound to the supplied GORM DB.
func NewAISuggestionRepository(db *gorm.DB) AISuggestionRepository {
	return &aiSuggestionRepo{db: db}
}

// Create inserts a new AI suggestion record. The caller must ensure that
// the JobID field is populated. GORM's built‑in timestamps (CreatedAt, UpdatedAt)
// are automatically managed.
func (r *aiSuggestionRepo) Create(ctx context.Context, suggestion *models.AISuggestion) error {
	if suggestion == nil {
		return errors.New("nil suggestion")
	}
	if suggestion.JobID == 0 {
		return errors.New("suggestion must reference a job")
	}
	if err := r.db.WithContext(ctx).Create(suggestion).Error; err != nil {
		return err
	}
	return nil
}

// GetByID retrieves a suggestion by its primary key. If the record does not exist,
// ErrAISuggestionNotFound is returned.
func (r *aiSuggestionRepo) GetByID(ctx context.Context, id uint) (*models.AISuggestion, error) {
	var s models.AISuggestion
	if err := r.db.WithContext(ctx).First(&s, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAISuggestionNotFound
		}
		return nil, err
	}
	return &s, nil
}

// ListByJobID fetches suggestions for a given job. Results are ordered by
// CreatedAt descending (newest first). Pagination is optional – passing limit=0
// disables the LIMIT clause.
func (r *aiSuggestionRepo) ListByJobID(ctx context.Context, jobID uint, limit, offset int) ([]*models.AISuggestion, error) {
	var suggestions []*models.AISuggestion
	query := r.db.WithContext(ctx).
		Where("job_id = ?", jobID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&suggestions).Error; err != nil {
		return nil, err
	}
	return suggestions, nil
}

// DeleteByID removes a suggestion. If the record does not exist, ErrAISuggestionNotFound
// is returned.
func (r *aiSuggestionRepo) DeleteByID(ctx context.Context, id uint) error {
	result := r.db.WithContext(ctx).Delete(&models.AISuggestion{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAISuggestionNotFound
	}
	return nil
}