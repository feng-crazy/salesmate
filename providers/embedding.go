package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	// DefaultEmbeddingModel is the default model for embeddings
	DefaultEmbeddingModel = "text-embedding-3-small"
	// Embedding3SmallDimension is the dimension for text-embedding-3-small
	Embedding3SmallDimension = 1536
	// Embedding3LargeDimension is the dimension for text-embedding-3-large
	Embedding3LargeDimension = 3072
)

// OpenAIEmbeddingProvider implements EmbeddingClient for OpenAI-compatible APIs
type OpenAIEmbeddingProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

// NewOpenAIEmbeddingProvider creates a new OpenAI embedding provider
func NewOpenAIEmbeddingProvider(apiKey, baseURL, model string) *OpenAIEmbeddingProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = DefaultEmbeddingModel
	}

	return &OpenAIEmbeddingProvider{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client:  &http.Client{},
	}
}

// Embed generates embeddings for the given texts
func (p *OpenAIEmbeddingProvider) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	url := fmt.Sprintf("%s/embeddings", p.baseURL)

	payload, err := json.Marshal(map[string]interface{}{
		"model": p.model,
		"input": texts,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	// Azure OpenAI uses api-key header instead
	if strings.Contains(p.baseURL, "azure") {
		httpReq.Header.Set("api-key", p.apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var apiResp struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(apiResp.Data) == 0 {
		return nil, fmt.Errorf("no embeddings returned from API")
	}

	// Sort by index to ensure correct order
	embeddings := make([][]float64, len(texts))
	for _, item := range apiResp.Data {
		if item.Index >= 0 && item.Index < len(texts) {
			embeddings[item.Index] = item.Embedding
		}
	}

	return embeddings, nil
}

// EmbeddingDimension returns the dimension size for the configured model
func (p *OpenAIEmbeddingProvider) EmbeddingDimension() int {
	switch p.model {
	case "text-embedding-3-large":
		return Embedding3LargeDimension
	case "text-embedding-3-small", "text-embedding-ada-002":
		return Embedding3SmallDimension
	default:
		// Default to 1536 for unknown models
		return Embedding3SmallDimension
	}
}
