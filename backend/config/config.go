package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	secretmanagerpb "google.golang.org/genproto/googleapis/cloud/secretmanager/v1"
	vault "github.com/hashicorp/vault/api"
)

// Config holds all runtime configuration for the application.
type Config struct {
	// General
	ServerPort string
	Env        string // development | production

	// PostgreSQL
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Redis
	RedisAddr     string
	RedisPassword string

	// RabbitMQ
	RabbitMQURL string

	// JWT
	JWTSecret string

	// HMAC for webhook validation
	HMACSecret string

	// OpenAI / Ollama
	OpenAIKey string

	// Secret manager integration
	SecretManagerEnabled bool
	SecretManagerType     string // "gcp" or "vault"

	// GCP specific
	GCPProjectID string

	// Vault specific
	VaultAddress string
	VaultToken   string
}

// LoadConfig reads environment variables, applies defaults, and optionally
// overrides secret values using a configured secret manager.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		// General
		ServerPort: getEnv("SERVER_PORT", "8080"),
		Env:        getEnv("ENV", "development"),

		// PostgreSQL
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", ""), // may be overridden
		DBName:     getEnv("DB_NAME", "pulseforge"),

		// Redis
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),

		// RabbitMQ
		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://guest:guest@localhost:5672/"),

		// JWT
		JWTSecret: getEnv("JWT_SECRET", ""),

		// HMAC
		HMACSecret: getEnv("HMAC_SECRET", ""),

		// OpenAI / Ollama
		OpenAIKey: getEnv("OPENAI_API_KEY", ""),

		// Secret manager
		SecretManagerEnabled: getEnvAsBool("SECRET_MANAGER_ENABLED", false),
		SecretManagerType:   getEnv("SECRET_MANAGER_TYPE", ""),
		GCPProjectID:         getEnv("GCP_PROJECT_ID", ""),
		VaultAddress:         getEnv("VAULT_ADDR", ""),
		VaultToken:           getEnv("VAULT_TOKEN", ""),
	}

	// If secret manager is enabled, fetch secrets and override values.
	if cfg.SecretManagerEnabled {
		if err := cfg.loadSecretsFromManager(); err != nil {
			return nil, fmt.Errorf("failed to load secrets from manager: %w", err)
		}
	}

	// Validate required fields.
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET must be set")
	}
	if cfg.HMACSecret == "" {
		return nil, fmt.Errorf("HMAC_SECRET must be set")
	}
	if cfg.DBPassword == "" {
		return nil, fmt.Errorf("DB_PASSWORD must be set")
	}
	if cfg.OpenAIKey == "" {
		// OpenAI key is optional for environments that don't use AI features,
		// but we log a warning for visibility.
		fmt.Fprintln(os.Stderr, "warning: OPENAI_API_KEY is not set – AI features will be disabled")
	}
	return cfg, nil
}

// loadSecretsFromManager dispatches to the appropriate secret manager implementation.
func (c *Config) loadSecretsFromManager() error {
	switch strings.ToLower(c.SecretManagerType) {
	case "gcp":
		return c.loadFromGCP()
	case "vault":
		return c.loadFromVault()
	case "":
		return fmt.Errorf("SECRET_MANAGER_TYPE must be set when SECRET_MANAGER_ENABLED is true")
	default:
		return fmt.Errorf("unsupported secret manager type: %s", c.SecretManagerType)
	}
}

// loadFromGCP fetches secrets from Google Cloud Secret Manager.
func (c *Config) loadFromGCP() error {
	if c.GCPProjectID == "" {
		return fmt.Errorf("GCP_PROJECT_ID must be set for GCP secret manager")
	}
	ctx := context.Background()
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("creating GCP secret manager client: %w", err)
	}
	defer client.Close()

	access := func(secretName string) (string, error) {
		if secretName == "" {
			return "", fmt.Errorf("secret name cannot be empty")
		}
		req := &secretmanagerpb.AccessSecretVersionRequest{
			Name: fmt.Sprintf("projects/%s/secrets/%s/versions/latest", c.GCPProjectID, secretName),
		}
		result, err := client.AccessSecretVersion(ctx, req)
		if err != nil {
			return "", fmt.Errorf("accessing secret %s: %w", secretName, err)
		}
		return string(result.Payload.Data), nil
	}

	if name := getEnv("DB_PASSWORD_SECRET_NAME", "db-password"); name != "" {
		if pwd, err := access(name); err == nil {
			c.DBPassword = pwd
		} else {
			return err
		}
	}
	if name := getEnv("JWT_SECRET_NAME", "jwt-secret"); name != "" {
		if s, err := access(name); err == nil {
			c.JWTSecret = s
		} else {
			return err
		}
	}
	if name := getEnv("HMAC_SECRET_NAME", "hmac-secret"); name != "" {
		if s, err := access(name); err == nil {
			c.HMACSecret = s
		} else {
			return err
		}
	}
	if name := getEnv("OPENAI_API_KEY_SECRET_NAME", "openai-key"); name != "" {
		if s, err := access(name); err == nil {
			c.OpenAIKey = s
		} else {
			return err
		}
	}
	return nil
}

// loadFromVault fetches secrets from HashiCorp Vault.
func (c *Config) loadFromVault() error {
	if c.VaultAddress == "" {
		return fmt.Errorf("VAULT_ADDR must be set for Vault secret manager")
	}
	if c.VaultToken == "" {
		return fmt.Errorf("VAULT_TOKEN must be set for Vault secret manager")
	}
	cfg := vault.DefaultConfig()
	cfg.Address = c.VaultAddress
	client, err := vault.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("creating Vault client: %w", err)
	}
	client.SetToken(c.VaultToken)

	read := func(path string) (string, error) {
		secret, err := client.Logical().Read(path)
		if err != nil {
			return "", fmt.Errorf("reading secret %s: %w", path, err)
		}
		if secret == nil || secret.Data == nil {
			return "", fmt.Errorf("no data at %s", path)
		}
		// Expect the secret value under the key "value"
		if v, ok := secret.Data["value"]; ok {
			if str, ok := v.(string); ok {
				return str, nil
			}
		}
		return "", fmt.Errorf("secret %s does not contain a 'value' field", path)
	}

	if p := getEnv("DB_PASSWORD_VAULT_PATH", "secret/data/db/password"); p != "" {
		if pwd, err := read(p); err == nil {
			c.DBPassword = pwd
		} else {
			return err
		}
	}
	if p := getEnv("JWT_SECRET_VAULT_PATH", "secret/data/jwt/secret"); p != "" {
		if s, err := read(p); err == nil {
			c.JWTSecret = s
		} else {
			return err
		}
	}
	if p := getEnv("HMAC_SECRET_VAULT_PATH", "secret/data/hmac/secret"); p != "" {
		if s, err := read(p); err == nil {
			c.HMACSecret = s
		} else {
			return err
		}
	}
	if p := getEnv("OPENAI_API_KEY_VAULT_PATH", "secret/data/openai/key"); p != "" {
		if s, err := read(p); err == nil {
			c.OpenAIKey = s
		} else {
			return err
		}
	}
	return nil
}

// getEnv returns the value of the environment variable named by key or fallback if not set.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvAsBool parses an environment variable into a bool, returning fallback on error or if unset.
func getEnvAsBool(key string, fallback bool) bool {
	valStr := os.Getenv(key)
	if valStr == "" {
		return fallback
	}
	val, err := strconv.ParseBool(valStr)
	if err != nil {
		return fallback
	}
	return val
}