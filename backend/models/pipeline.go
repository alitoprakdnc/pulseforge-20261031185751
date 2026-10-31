package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// CIProvider represents a supported CI/CD provider.
type CIProvider string

const (
	ProviderGitHub   CIProvider = "github"
	ProviderGitLab   CIProvider = "gitlab"
	ProviderCircleCI CIProvider = "circleci"
)

// IsValid checks whether the provider value is one of the supported constants.
func (p CIProvider) IsValid() bool {
	switch p {
	case ProviderGitHub, ProviderGitLab, ProviderCircleCI:
		return true
	}
	return false
}

// Metadata stores arbitrary key‑value pairs for a pipeline.
// It is persisted as JSONB in PostgreSQL.
type Metadata map[string]interface{}

// Value implements the driver.Valuer interface for storing Metadata.
func (m Metadata) Value() (driver.Value, error) {
	if len(m) == 0 {
		return "{}", nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Scan implements the sql.Scanner interface for loading Metadata.
func (m *Metadata) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("cannot scan type %T into Metadata", value)
	}
	return json.Unmarshal(data, m)
}

// Pipeline represents a CI/CD pipeline that sends webhook events to PulseForge.
type Pipeline struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	Provider      CIProvider     `gorm:"type:varchar(20);not null;index" json:"provider"`
	Repository    string         `gorm:"type:varchar(255);not null;index" json:"repository"` // e.g. "owner/repo"
	WebhookSecret string         `gorm:"type:varchar(255);not null" json:"-"`                 // never exposed via API
	Metadata      Metadata       `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
	Jobs          []Job          `gorm:"foreignKey:PipelineID" json:"-"`
}

// TableName overrides the default table name used by GORM.
func (Pipeline) TableName() string {
	return "pipelines"
}

// BeforeCreate runs validation before a Pipeline record is persisted.
func (p *Pipeline) BeforeCreate(tx *gorm.DB) (err error) {
	if !p.Provider.IsValid() {
		return fmt.Errorf("invalid CI provider: %s", p.Provider)
	}
	if p.Repository == "" {
		return fmt.Errorf("repository cannot be empty")
	}
	if p.WebhookSecret == "" {
		return fmt.Errorf("webhook secret cannot be empty")
	}
	return nil
}

// SafeCopy returns a shallow copy of the Pipeline with the secret stripped.
// Useful when returning data to the frontend.
func (p *Pipeline) SafeCopy() *Pipeline {
	c := *p
	c.WebhookSecret = ""
	return &c
}