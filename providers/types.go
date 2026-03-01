package providers

import (
	"context"
)

// LLMProvider defines the interface for LLM providers
type LLMProvider interface {
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	GetDefaultModel() string
}

// ChatRequest represents a request to the LLM
type ChatRequest struct {
	Messages  []Message      `json:"messages"`
	Tools     []ToolDef      `json:"tools,omitempty"`
	Model     string         `json:"model,omitempty"`
	Temperature float64      `json:"temperature,omitempty"`
	MaxTokens int           `json:"max_tokens,omitempty"`
}

// Message represents a message in the conversation
type Message struct {
	Role    string      `json:"role"`  // "system", "user", "assistant", "tool"
	Content interface{} `json:"content"` // String or array of content parts for multimodal
	Name    string      `json:"name,omitempty"` // For tool calls
}

// ToolDef defines a function/tool that can be called
type ToolDef struct {
	Type     string     `json:"type"`
	Function FunctionDef `json:"function"`
}

// FunctionDef describes a function
type FunctionDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall represents a call to a tool
type ToolCall struct {
	ID       string                 `json:"id"`
	Name     string                 `json:"name"`
	Args     map[string]interface{} `json:"arguments"`
	Type     string                 `json:"type"`
}

// ChatResponse is the response from the LLM
type ChatResponse struct {
	Content      string     `json:"content"`
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`
	HasToolCalls bool       `json:"has_tool_calls"`
}