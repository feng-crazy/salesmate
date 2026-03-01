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

// OpenAIProvider implements LLMProvider for OpenAI-compatible APIs
type OpenAIProvider struct {
	apiKey       string
	baseURL      string
	defaultModel string
	client       *http.Client
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey, baseURL, defaultModel string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1" // Default OpenAI endpoint
	}

	return &OpenAIProvider{
		apiKey:       apiKey,
		baseURL:      strings.TrimRight(baseURL, "/"),
		defaultModel: defaultModel,
		client:       &http.Client{},
	}
}

// Chat implements the LLMProvider interface
func (p *OpenAIProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		req.Model = p.defaultModel
	}

	url := fmt.Sprintf("%s/chat/completions", p.baseURL)

	payload, err := json.Marshal(map[string]interface{}{
		"model":       req.Model,
		"messages":    req.Messages,
		"temperature": req.Temperature,
		"max_tokens":  req.MaxTokens,
		"tools":       req.Tools,
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
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from API")
	}

	choice := apiResp.Choices[0].Message

	response := &ChatResponse{
		Content: choice.Content,
	}

	if len(choice.ToolCalls) > 0 {
		response.HasToolCalls = true
		for _, tc := range choice.ToolCalls {
			args := make(map[string]interface{})
			if tc.Function.Arguments != "" {
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					return nil, fmt.Errorf("failed to unmarshal tool arguments: %w", err)
				}
			}

			response.ToolCalls = append(response.ToolCalls, ToolCall{
				ID:   tc.ID,
				Name: tc.Function.Name,
				Args: args,
				Type: tc.Type,
			})
		}
	}

	return response, nil
}

// GetDefaultModel returns the default model for this provider
func (p *OpenAIProvider) GetDefaultModel() string {
	return p.defaultModel
}