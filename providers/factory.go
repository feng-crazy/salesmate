package providers

import (
	"fmt"
	"strings"

	"salesmate/config"
)

// ProviderFactory creates the appropriate LLM provider based on the config
func ProviderFactory(cfg *config.Config) (LLMProvider, error) {
	model := cfg.Agents.Defaults.Model

	// Determine the provider based on model prefix
	providerName := ""
	parts := strings.Split(model, "/")
	if len(parts) > 1 {
		providerName = strings.ToLower(parts[0])
	}

	// Get the API key and base URL from config
	apiKey := cfg.Providers.GetAPIKey(model)
	if apiKey == "" && !strings.HasPrefix(model, "bedrock/") {
		return nil, fmt.Errorf("no API key configured for model: %s", model)
	}

	baseURL := cfg.Providers.GetAPIBase(model)

	// Create the appropriate provider based on the model
	switch providerName {
	case "custom":
		return NewCustomProvider(apiKey, baseURL, model), nil
	case "openai":
		return NewOpenAIProvider(apiKey, baseURL, model), nil
	case "anthropic", "claude":
		// For Anthropic models, we'll use a compatible API wrapper
		return NewCustomProvider(apiKey, "https://api.anthropic.com/v1", model), nil
	case "openrouter":
		// Use a custom provider with OpenRouter's endpoint
		return NewCustomProvider(apiKey, "https://openrouter.ai/api/v1", model), nil
	case "groq":
		// Use a custom provider with Groq's endpoint
		return NewCustomProvider(apiKey, "https://api.groq.com/openai/v1", model), nil
	case "gemini":
		// Use a custom provider with Gemini's endpoint
		return NewCustomProvider(apiKey, "https://generativelanguage.googleapis.com/v1beta", model), nil
	default:
		// Default to LiteLLM provider for compatibility
		return NewLiteLLMProvider(apiKey, baseURL, model), nil
	}
}
