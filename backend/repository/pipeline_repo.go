package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"backend/models"
	"backend/secretmanager"
)

// PipelineRepository provides CRUD operations for Pipeline entities.
// Secrets (webhook signing keys) are stored in an external secret manager
// and referenced by the SecretRef field on the Pipeline model.
type PipelineRepository struct {
	db           *gorm.DB
	secretClient secretmanager.Client
}

// NewPipelineRepository creates a new instance of PipelineRepository.
func NewPipelineRepository(db *gorm.DB, secretClient secretmanager.Client) *PipelineRepository {
	return &PipelineRepository{
		db:           db,
		secretClient: secretClient,
	}
}

// ErrPipelineNotFound is returned when a pipeline cannot be located.
var ErrPipelineNotFound = errors.New("pipeline not found")

// Create stores a new pipeline record and persists its webhook secret
// in the configured secret manager. The secret is never written to the
// database; only a reference (SecretRef) is saved.
func (r *PipelineRepository) Create(ctx context.Context, p *models.Pipeline, webhookSecret string) error {
	if p == nil {
		return errors.New("pipeline is nil")
	}
	if webhookSecret == "" {
		return errors.New("webhook secret cannot be empty")
	}

	// Generate a unique secret reference name.
	secretRef := fmt.Sprintf("pipeline-%s-secret", uuid.NewString())

	// Store the secret in the secret manager first.
	if err := r.secretClient.SetSecret(ctx, secretRef, webhookSecret); err != nil {
		return fmt.Errorf("failed to store webhook secret: %w", err)
	}
	p.SecretRef = secretRef

	// Persist the pipeline record.
	if err := r.db.WithContext(ctx).Create(p).Error; err != nil {
		// Attempt to clean up the secret if DB write fails.
		_ = r.secretClient.DeleteSecret(ctx, secretRef)
		return fmt.Errorf("failed to create pipeline record: %w", err)
	}
	return nil
}

// GetByID retrieves a pipeline by its primary key.
// The returned Pipeline contains the SecretRef but not the secret value itself.
func (r *PipelineRepository) GetByID(ctx context.Context, id uint) (*models.Pipeline, error) {
	var p models.Pipeline
	if err := r.db.WithContext(ctx).First(&p, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPipelineNotFound
		}
		return nil, fmt.Errorf("failed to fetch pipeline: %w", err)
	}
	return &p, nil
}

// ListByUser returns all pipelines owned by the specified user.
func (r *PipelineRepository) ListByUser(ctx context.Context, userID uint) ([]*models.Pipeline, error) {
	var pipelines []*models.Pipeline
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&pipelines).Error; err != nil {
		return nil, fmt.Errorf("failed to list pipelines: %w", err)
	}
	return pipelines, nil
}

// Update modifies mutable fields of a pipeline.
// If a non‑empty webhookSecret is supplied, the secret manager entry is
// updated and the SecretRef remains unchanged.
func (r *PipelineRepository) Update(ctx context.Context, p *models.Pipeline, webhookSecret string) error {
	if p == nil {
		return errors.New("pipeline is nil")
	}

	// Begin a transaction to keep DB and secret manager in sync.
	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %w", tx.Error)
	}
	defer func() {
		// Ensure the transaction is rolled back if not committed.
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// Update secret if a new value is provided.
	if webhookSecret != "" {
		if p.SecretRef == "" {
			// Should never happen, but guard against it.
			newRef := fmt.Sprintf("pipeline-%s-secret", uuid.NewString())
			if err := r.secretClient.SetSecret(ctx, newRef, webhookSecret); err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to store new webhook secret: %w", err)
			}
			p.SecretRef = newRef
		} else {
			if err := r.secretClient.SetSecret(ctx, p.SecretRef, webhookSecret); err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to update webhook secret: %w", err)
			}
		}
	}

	// Persist pipeline changes (excluding SecretRef which is already set).
	if err := tx.Save(p).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to update pipeline: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit pipeline update: %w", err)
	}
	return nil
}

// Delete removes a pipeline record and its associated secret from the secret manager.
func (r *PipelineRepository) Delete(ctx context.Context, id uint) error {
	// Load pipeline to obtain SecretRef.
	p, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Transactionally delete DB record.
	if err := r.db.WithContext(ctx).Delete(&models.Pipeline{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete pipeline record: %w", err)
	}

	// Clean up secret if reference exists.
	if p.SecretRef != "" {
		if err := r.secretClient.DeleteSecret(ctx, p.SecretRef); err != nil {
			// Log but do not fail the whole operation – secret leakage is less critical than DB consistency.
			// Assuming a logger is available via the package's logger; fallback to fmt if not.
			fmt.Printf("warning: failed to delete secret %s: %v\n", p.SecretRef, err)
		}
	}
	return nil
}

// GetWebhookSecret fetches the raw webhook secret for a given pipeline.
// Caller must have appropriate authorization; this method only retrieves the secret.
func (r *PipelineRepository) GetWebhookSecret(ctx context.Context, pipelineID uint) (string, error) {
	p, err := r.GetByID(ctx, pipelineID)
	if err != nil {
		return "", err
	}
	if p.SecretRef == "" {
		return "", errors.New("pipeline does not have an associated secret")
	}
	secret, err := r.secretClient.GetSecret(ctx, p.SecretRef)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve webhook secret: %w", err)
	}
	return secret, nil
}