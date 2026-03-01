package tools

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// WebFetchTool implements a tool to fetch web content
type WebFetchTool struct{}

// NewWebFetchTool creates a new web fetch tool
func NewWebFetchTool() *WebFetchTool {
	return &WebFetchTool{}
}

// Name returns the name of the tool
func (t *WebFetchTool) Name() string {
	return "web_fetch"
}

// Description returns the description of the tool
func (t *WebFetchTool) Description() string {
	return "Fetch content from a web page"
}

// Call executes the tool with the given arguments
func (t *WebFetchTool) Call(args map[string]interface{}) (string, error) {
	urlStr, ok := args["url"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'url' argument")
	}

	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("invalid URL scheme: %s", parsedURL.Scheme)
	}

	// Fetch the content
	client := &http.Client{}
	resp, err := client.Get(urlStr)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetching URL returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Extract text content (basic implementation - in practice would parse HTML)
	content := string(body)

	// Limit the response size
	if len(content) > 4000 { // Similar to Python version's limit
		content = content[:4000] + "\n... (content truncated)"
	}

	return fmt.Sprintf("Content from %s:\n\n%s", urlStr, content), nil
}