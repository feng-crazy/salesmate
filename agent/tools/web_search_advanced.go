package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// AdvancedWebSearchTool extends the basic web search with summarization capabilities
type AdvancedWebSearchTool struct {
	apiKey     string
	maxResults int
	summarize  bool
}

// NewAdvancedWebSearchTool creates a new advanced web search tool
func NewAdvancedWebSearchTool(apiKey string, maxResults int, summarize bool) *AdvancedWebSearchTool {
	if maxResults <= 0 {
		maxResults = 5 // default
	}
	return &AdvancedWebSearchTool{
		apiKey:     apiKey,
		maxResults: maxResults,
		summarize:  summarize,
	}
}

// Name returns the name of the tool
func (t *AdvancedWebSearchTool) Name() string {
	return "web_search_advanced"
}

// Description returns the description of the tool
func (t *AdvancedWebSearchTool) Description() string {
	return "Advanced search the web using the Brave Search API with optional summarization. Returns titles, URLs, and summaries/snippets."
}

// Call executes the tool with the given arguments
func (t *AdvancedWebSearchTool) Call(args map[string]interface{}) (string, error) {
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

	shouldSummarize, ok := args["summarize"].(bool)
	if !ok {
		shouldSummarize = t.summarize
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

				var content string
				if shouldSummarize && url != "" {
					// Attempt to fetch and summarize the content
					summary, err := t.fetchAndSummarizeContent(url)
					if err == nil && summary != "" {
						content = summary
					} else {
						// Fall back to snippet if summarization fails
						content = snippet
					}
				} else {
					// Use the snippet if not summarizing
					content = snippet
				}

				if content != "" {
					resultStr.WriteString(fmt.Sprintf("   %s\n", content))
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

// fetchAndSummarizeContent fetches the content of a URL and creates a summary
func (t *AdvancedWebSearchTool) fetchAndSummarizeContent(url string) (string, error) {
	// This is a simplified approach - in a real implementation, you might:
	// 1. Fetch the content from the URL
	// 2. Parse the HTML to extract main content
	// 3. Use an LLM to summarize the content

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	// Set a realistic user agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Nanobot/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Very basic extraction of content (in a real implementation, use a proper HTML parser)
	content := string(body)

	// Remove HTML tags and normalize whitespace
	re := regexp.MustCompile(`<[^>]+>`)
	text := re.ReplaceAllString(content, " ")

	// Normalize whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")

	// Extract the most relevant portion (first 500 chars after cleaning)
	if len(text) > 500 {
		text = text[:500] + "..."
	}

	return strings.TrimSpace(text), nil
}