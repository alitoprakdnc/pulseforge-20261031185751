package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v9"
	"github.com/pkg/errors"

	"pulseforge/backend/config"
	"pulseforge/backend/logger"
	"pulseforge/backend/models"
	"pulseforge/backend/repository"
	"pulseforge/backend/utils"
)

// AIService provides methods to generate AI suggestions for CI/CD job logs.
type AIService struct {
	cfg    *config.Config
	log    *logger.Logger
	aiRepo repository.AIRepository
	llm    utils.LLMClient
	cache  *redis.Client
}

// NewAIService creates a new AIService instance.
// It requires a configuration, logger, AI repository, LLM client and a Redis client for caching.
func NewAIService(
	cfg *config.Config,
	log *logger.Logger,
	aiRepo repository.AIRepository,
	llm utils.LLMClient,
	cache *redis.Client,
) *AIService {
	return &AIService{
		cfg:    cfg,
		log:    log,
		aiRepo: aiRepo,
		llm:    llm,
		cache:  cache,
	}
}

// GenerateSuggestion creates an AI suggestion for a given job.
// It formats the logs, calls the LLM, parses the response and persists the suggestion.
// The result is also cached in Redis for quick subsequent reads.
func (s *AIService) GenerateSuggestion(ctx context.Context, jobID int64, logs string) (*models.AISuggestion, error) {
	if logs == "" {
		return nil, errors.New("empty logs provided")
	}

	// 1️⃣ Try to fetch from cache first.
	cached, err := s.getCachedSuggestion(ctx, jobID)
	if err != nil {
		s.log.Error("redis cache miss", "jobID", jobID, "err", err)
	}
	if cached != nil {
		s.log.Info("ai suggestion cache hit", "jobID", jobID)
		return cached, nil
	}

	// 2️⃣ Build the prompt for the LLM.
	prompt := s.buildPrompt(logs)

	// 3️⃣ Call the LLM with a timeout derived from config.
	llmCtx, cancel := context.WithTimeout(ctx, time.Duration(s.cfg.LLMTimeoutSeconds)*time.Second)
	defer cancel()

	rawResponse, err := s.llm.Generate(llmCtx, prompt)
	if err != nil {
		s.log.Error("llm generation failed", "jobID", jobID, "err", err)
		return nil, errors.Wrap(err, "llm generation")
	}

	// 4️⃣ Parse the LLM response.
	suggestion, err := s.parseResponse(jobID, rawResponse)
	if err != nil {
		s.log.Error("failed to parse llm response", "jobID", jobID, "raw", rawResponse, "err", err)
		// Fallback: store the raw response as suggestion with zero confidence.
		suggestion = &models.AISuggestion{
			JobID:      jobID,
			Text:       rawResponse,
			Confidence: 0.0,
			CreatedAt:  time.Now().UTC(),
		}
	}

	// 5️⃣ Persist the suggestion.
	if err := s.aiRepo.CreateSuggestion(ctx, suggestion); err != nil {
		s.log.Error("failed to store ai suggestion", "jobID", jobID, "err", err)
		// Continue – we still want to return the suggestion even if DB write fails.
	}

	// 6️⃣ Cache the suggestion for future calls.
	if err := s.cacheSuggestion(ctx, suggestion); err != nil {
		s.log.Error("failed to cache ai suggestion", "jobID", jobID, "err", err)
	}

	return suggestion, nil
}

// buildPrompt creates a consistent prompt for the LLM, embedding the logs.
func (s *AIService) buildPrompt(logs string) string {
	return fmt.Sprintf(`You are an expert DevOps engineer. Analyze the following CI/CD job logs and provide a concise, actionable recommendation to improve reliability, performance, or fix failures. Return the answer as a JSON object with the following fields:
{
  "suggestion": "string – the recommendation",
  "confidence": number – a value between 0 and 1 indicating confidence in the suggestion
}
If you cannot produce a confident recommendation, set confidence to 0 and provide a brief explanation.

Logs:
%s`, logs)
}

// parseResponse attempts to unmarshal the LLM output into an AISuggestion.
// It expects a JSON payload; if unmarshalling fails, an error is returned.
func (s *AIService) parseResponse(jobID int64, raw string) (*models.AISuggestion, error) {
	type llmPayload struct {
		Suggestion string  `json:"suggestion"`
		Confidence float64 `json:"confidence"`
	}

	var payload llmPayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, errors.Wrap(err, "json unmarshal")
	}

	// Clamp confidence to [0,1]
	if payload.Confidence < 0 {
		payload.Confidence = 0
	}
	if payload.Confidence > 1 {
		payload.Confidence = 1
	}

	return &models.AISuggestion{
		JobID:      jobID,
		Text:       payload.Suggestion,
		Confidence: payload.Confidence,
		CreatedAt:  time.Now().UTC(),
	}, nil
}

// getCachedSuggestion retrieves a cached suggestion from Redis, if present.
func (s *AIService) getCachedSuggestion(ctx context.Context, jobID int64) (*models.AISuggestion, error) {
	key := fmt.Sprintf("ai:suggestion:%d", jobID)
	data, err := s.cache.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil // cache miss
	}
	if err != nil {
		return nil, err
	}

	var cached models.AISuggestion
	if err := json.Unmarshal([]byte(data), &cached); err != nil {
		return nil, err
	}
	return &cached, nil
}

// cacheSuggestion stores the suggestion in Redis with a TTL.
func (s *AIService) cacheSuggestion(ctx context.Context, suggestion *models.AISuggestion) error {
	key := fmt.Sprintf("ai:suggestion:%d", suggestion.JobID)
	payload, err := json.Marshal(suggestion)
	if err != nil {
		return err
	}
	// Cache for 24 hours – enough for UI refreshes while keeping storage bounded.
	return s.cache.Set(ctx, key, payload, 24*time.Hour).Err()
}