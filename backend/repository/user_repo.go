package repository

import (
	"context"
	"errors"

	"github.com/pulseforge/backend/models"
	"gorm.io/gorm"
)

// UserRepository defines CRUD operations for the User model.
type UserRepository interface {
	// Create inserts a new user record.
	Create(ctx context.Context, user *models.User) error
	// GetByID retrieves a user by its primary key.
	GetByID(ctx context.Context, id uint) (*models.User, error)
	// GetByEmail retrieves a user by its email address.
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	// Update persists changes to an existing user.
	Update(ctx context.Context, user *models.User) error
	// Delete removes a user record.
	Delete(ctx context.Context, id uint) error
	// List returns a slice of users with optional pagination.
	List(ctx context.Context, limit, offset int) ([]*models.User, error)
}

// userRepository is the concrete implementation of UserRepository.
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new instance of UserRepository.
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create inserts a new user record into the database.
func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	if user == nil {
		return errors.New("user is nil")
	}
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return err
	}
	return nil
}

// GetByID fetches a user by its ID.
func (r *userRepository) GetByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail fetches a user by email address.
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// Update saves the modified fields of a user.
// It expects the user struct to contain a valid primary key.
func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	if user == nil {
		return errors.New("user is nil")
	}
	if user.ID == 0 {
		return errors.New("user ID is required for update")
	}
	// Use Select to only update non-zero fields, but allow zero values via map if needed.
	if err := r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", user.ID).Updates(user).Error; err != nil {
		return err
	}
	return nil
}

// Delete removes a user by ID.
func (r *userRepository) Delete(ctx context.Context, id uint) error {
	if id == 0 {
		return errors.New("invalid user ID")
	}
	if err := r.db.WithContext(ctx).Delete(&models.User{}, id).Error; err != nil {
		return err
	}
	return nil
}

// List returns users with optional pagination.
// If limit <= 0 a default of 100 is used. Offset defaults to 0.
func (r *userRepository) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	var users []*models.User
	if err := r.db.WithContext(ctx).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}