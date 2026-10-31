package models

import (
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Role defines the permission level of a user.
type Role string

const (
	RoleAdmin   Role = "admin"
	RoleManager Role = "manager"
	RoleViewer  Role = "viewer"
)

// User represents an application user.
// It is stored in the `users` table and managed via GORM.
type User struct {
	ID           uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Email        string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Name         string         `gorm:"type:varchar(150);not null" json:"name"`
	PasswordHash string         `gorm:"type:char(60);not null" json:"-"`
	Role         Role           `gorm:"type:varchar(20);default:'viewer';not null" json:"role"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName overrides the default table name.
func (User) TableName() string {
	return "users"
}

// NewUser creates a new User instance with a hashed password.
// It validates the supplied fields and returns an error if validation fails.
func NewUser(email, name, plainPassword string, role Role) (*User, error) {
	if err := validateEmail(email); err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("name cannot be empty")
	}
	if err := validatePassword(plainPassword); err != nil {
		return nil, err
	}
	if err := validateRole(role); err != nil {
		return nil, err
	}

	hash, err := hashPassword(plainPassword)
	if err != nil {
		return nil, err
	}

	return &User{
		Email:        strings.ToLower(strings.TrimSpace(email)),
		Name:         strings.TrimSpace(name),
		PasswordHash: hash,
		Role:         role,
	}, nil
}

// SetPassword hashes the provided plain‑text password and stores it.
// It can be used to change a user's password.
func (u *User) SetPassword(plainPassword string) error {
	if err := validatePassword(plainPassword); err != nil {
		return err
	}
	hash, err := hashPassword(plainPassword)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	return nil
}

// VerifyPassword checks whether the supplied plain‑text password matches the stored hash.
func (u *User) VerifyPassword(plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(plainPassword))
	return err == nil
}

// BeforeCreate GORM hook ensures email normalization and password hashing
// in case a User is persisted without using NewUser.
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.Email = strings.ToLower(strings.TrimSpace(u.Email))
	if u.PasswordHash == "" {
		return errors.New("password hash must be set before creating a user")
	}
	return nil
}

// ---------- Helper functions ----------

func hashPassword(pw string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

func validateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return errors.New("email cannot be empty")
	}
	// Very lightweight validation; more thorough checks can be added later.
	if !strings.Contains(email, "@") {
		return errors.New("invalid email format")
	}
	return nil
}

func validatePassword(pw string) error {
	if len(pw) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
}

func validateRole(r Role) error {
	switch r {
	case RoleAdmin, RoleManager, RoleViewer:
		return nil
	default:
		return errors.New("invalid role")
	}
}