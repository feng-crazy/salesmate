package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// WebSearchTool implements a tool to search the web using real API
type WebSearchTool struct {
	apiKey     string
	maxResults int
}

// NewWebSearchTool creates a new web search tool
func NewWebSearchTool(apiKey string, maxResults int) *WebSearchTool {
	if maxResults <= 0 {
		maxResults = 5 // default
	}
	return &WebSearchTool{
		apiKey:     apiKey,
		maxResults: maxResults,
	}
}

// Name returns the name of the tool
func (t *WebSearchTool) Name() string {
	return "web_search"
}

// Description returns the description of the tool
func (t *WebSearchTool) Description() string {
	return "Search the web using the Brave Search API. Returns titles, URLs, and snippets."
}

// Call executes the tool with the given arguments
func (t *WebSearchTool) Call(args map[string]interface{}) (string, error) {
	query, ok := args["query"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'query' argument")
	}

	countFloat, ok := args["count"].(float64) // JSON unmarshals numbers as float64
	count := int(countFloat)
	if !ok {
		count = t.maxResults
	}
	if count > 10 {
		count = 10 // cap at 10
	}
	if count < 1 {
		count = 1 // minimum 1
	}

	if t.apiKey == "" {
		return "Error: Brave Search API key not configured. Set it in config under tools.web.search.apiKey", nil
	}

	// Construct the API request
	searchURL := fmt.Sprintf("https://api.search.brave.com/res/v1/web/search?q=%s&count=%d",
		url.QueryEscape(query), count)

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", t.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute search: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("search API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Parse the response
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Format the results
	var resultStr strings.Builder
	resultStr.WriteString(fmt.Sprintf("Results for: %s\n\n", query))

	if webResults, ok := result["web"].(map[string]interface{})["results"].([]interface{}); ok {
		for i, item := range webResults {
			if i >= count {
				break
			}

			if resultMap, ok := item.(map[string]interface{}); ok {
				title, _ := resultMap["title"].(string)
				url, _ := resultMap["url"].(string)
				snippet, _ := resultMap["description"].(string)

				if title != "" {
					resultStr.WriteString(fmt.Sprintf("%d. %s\n", i+1, title))
				}
				if url != "" {
					resultStr.WriteString(fmt.Sprintf("   %s\n", url))
				}
				if snippet != "" {
					resultStr.WriteString(fmt.Sprintf("   %s\n", snippet))
				}
				resultStr.WriteString("\n")
			}
		}
	}

	if resultStr.Len() == 0 {
		return fmt.Sprintf("No results found for: %s", query), nil
	}

	return resultStr.String(), nil
}