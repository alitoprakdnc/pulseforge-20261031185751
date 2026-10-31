package utils

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"go.uber.org/zap"
)

// LLMClient abstracts a Large Language Model provider.
type LLMClient interface {
	// Generate returns the full completion for the given prompt.
	Generate(ctx context.Context, prompt string) (string, error)
	// Stream returns a channel that yields incremental token fragments as they arrive.
	// The second channel delivers a single error (or nil) when the stream ends.
	Stream(ctx context.Context, prompt string) (<-chan string, <-chan error)
}

// ---------------------------------------------------------------------
// OpenAI implementation
// ---------------------------------------------------------------------

type openAIClient struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewOpenAIClient creates a client for the OpenAI API.
// It reads configuration from environment variables:
//   OPENAI_API_KEY   – required
//   OPENAI_MODEL     – optional (default: gpt-4o-mini)
//   OPENAI_BASE_URL  – optional (default: https://api.openai.com/v1)
func NewOpenAIClient(logger *zap.Logger) (LLMClient, error) {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return nil, errors.New("OPENAI_API_KEY is not set")
	}
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o-mini"
	}
	base := os.Getenv("OPENAI_BASE_URL")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	return &openAIClient{
		apiKey:  key,
		model:   model,
		baseURL: base,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
		logger: logger,
	}, nil
}

// Generate performs a non‑streaming request.
func (c *openAIClient) Generate(ctx context.Context, prompt string) (string, error) {
	streamCh, errCh := c.Stream(ctx, prompt)
	var buf bytes.Buffer
	for {
		select {
		case token, ok := <-streamCh:
			if !ok {
				// stream finished successfully
				return buf.String(), nil
			}
			buf.WriteString(token)
		case err, ok := <-errCh:
			if ok && err != nil {
				return "", err
			}
			// err channel closed without error – treat as success
			return buf.String(), nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

// Stream initiates a streaming chat completion request.
func (c *openAIClient) Stream(ctx context.Context, prompt string) (<-chan string, <-chan error) {
	tokenCh := make(chan string)
	errCh := make(chan error, 1)

	go func() {
		defer close(tokenCh)
		defer close(errCh)

		payload := map[string]any{
			"model": c.model,
			"messages": []map[string]string{
				{
					"role":    "user",
					"content": prompt,
				},
			},
			"stream": true,
		}
		body, _ := json.Marshal(payload)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
		if err != nil {
			errCh <- err
			return
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			errCh <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			errCh <- errors.New("openai: non‑200 response: " + resp.Status + " – " + string(b))
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			// OpenAI streams lines prefixed with "data: "
			if len(line) < 6 || line[:6] != "data: " {
				continue
			}
			data := line[6:]
			if data == "[DONE]" {
				// Normal termination
				return
			}
			var chunk openAIChatChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				c.logger.Warn("failed to unmarshal openai chunk", zap.Error(err), zap.String("raw", data))
				continue
			}
			if len(chunk.Choices) == 0 {
				continue
			}
			content := chunk.Choices[0].Delta.Content
			if content != "" {
				select {
				case tokenCh <- content:
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				}
			}
		}
		if err := scanner.Err(); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- err
		}
	}()

	return tokenCh, errCh
}

// openAIChatChunk mirrors the streaming response format.
type openAIChatChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content,omitempty"`
		} `json:"delta,omitempty"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices,omitempty"`
}

// ---------------------------------------------------------------------
// Ollama implementation
// ---------------------------------------------------------------------

type ollamaClient struct {
	baseURL    string
	model      string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewOllamaClient creates a client for an Ollama server.
// Configuration via env vars:
//   OLLAMA_BASE_URL – required (e.g., http://localhost:11434)
//   OLLAMA_MODEL   – optional (default: llama3)
func NewOllamaClient(logger *zap.Logger) (LLMClient, error) {
	base := os.Getenv("OLLAMA_BASE_URL")
	if base == "" {
		return nil, errors.New("OLLAMA_BASE_URL is not set")
	}
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "llama3"
	}
	return &ollamaClient{
		baseURL: base,
		model:   model,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
		logger: logger,
	}, nil
}

// Generate uses the streaming implementation and concatenates the result.
func (c *ollamaClient) Generate(ctx context.Context, prompt string) (string, error) {
	streamCh, errCh := c.Stream(ctx, prompt)
	var buf bytes.Buffer
	for {
		select {
		case token, ok := <-streamCh:
			if !ok {
				return buf.String(), nil
			}
			buf.WriteString(token)
		case err, ok := <-errCh:
			if ok && err != nil {
				return "", err
			}
			return buf.String(), nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

// Stream initiates a streaming request to Ollama.
func (c *ollamaClient) Stream(ctx context.Context, prompt string) (<-chan string, <-chan error) {
	tokenCh := make(chan string)
	errCh := make(chan error, 1)

	go func() {
		defer close(tokenCh)
		defer close(errCh)

		payload := map[string]any{
			"model":  c.model,
			"messages": []map[string]string{
				{
					"role":    "user",
					"content": prompt,
				},
			},
			"stream": true,
		}
		body, _ := json.Marshal(payload)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(body))
		if err != nil {
			errCh <- err
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			errCh <- err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			errCh <- errors.New("ollama: non‑200 response: " + resp.Status + " – " + string(b))
			return
		}

		decoder := json.NewDecoder(resp.Body)
		for {
			var chunk ollamaChatChunk
			if err := decoder.Decode(&chunk); err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				if !errors.Is(err, context.Canceled) {
					errCh <- err
				}
				return
			}
			if chunk.Done {
				// Stream finished
				return
			}
			if chunk.Message.Content != "" {
				select {
				case tokenCh <- chunk.Message.Content:
				case <-ctx.Done():
					errCh <- ctx.Err()
					return
				}
			}
		}
	}()

	return tokenCh, errCh
}

// ollamaChatChunk mirrors Ollama's streaming JSON format.
type ollamaChatChunk struct {
	Model   string `json:"model,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	Message struct {
		Role    string `json:"role,omitempty"`
		Content string `json:"content,omitempty"`
	} `json:"message,omitempty"`
	Done bool `json:"done,omitempty"`
}

// ---------------------------------------------------------------------
// Factory helper – selects provider based on env configuration
// ---------------------------------------------------------------------

// GetLLMClient returns an LLMClient based on environment variables.
// If OLLAMA_BASE_URL is set, Ollama takes precedence; otherwise OpenAI is used.
func GetLLMClient(logger *zap.Logger) (LLMClient, error) {
	if os.Getenv("OLLAMA_BASE_URL") != "" {
		return NewOllamaClient(logger)
	}
	if os.Getenv("OPENAI_API_KEY") != "" {
		return NewOpenAIClient(logger)
	}
	return nil, errors.New("no LLM provider configured; set either OLLAMA_BASE_URL or OPENAI_API_KEY")
}