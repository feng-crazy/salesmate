package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EditFileTool implements a tool to edit files by replacing content
type EditFileTool struct {
	workspace   string
	allowedDir  string  // If set, restricts operations to this directory
}

// NewEditFileTool creates a new edit file tool
func NewEditFileTool(workspace string, allowedDir string) *EditFileTool {
	return &EditFileTool{
		workspace:  workspace,
		allowedDir: allowedDir,
	}
}

// Name returns the name of the tool
func (t *EditFileTool) Name() string {
	return "edit_file"
}

// Description returns the description of the tool
func (t *EditFileTool) Description() string {
	return "Edit a file by replacing old_text with new_text. The old_text must exist exactly in the file."
}

// Call executes the tool with the given arguments
func (t *EditFileTool) Call(args map[string]interface{}) (string, error) {
	filePath, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'path' argument")
	}

	oldText, ok := args["old_text"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'old_text' argument")
	}

	newText, ok := args["new_text"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'new_text' argument")
	}

	// Verify path is allowed if restriction is in place
	if t.allowedDir != "" {
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			return "", fmt.Errorf("error resolving path: %w", err)
		}
		absAllowedDir, err := filepath.Abs(t.allowedDir)
		if err != nil {
			return "", fmt.Errorf("error resolving allowed directory: %w", err)
		}

		if !filepath.HasPrefix(absPath, absAllowedDir) {
			return "", fmt.Errorf("path %s is outside allowed directory %s", filePath, t.allowedDir)
		}
	}

	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	fileContent := string(content)

	// Check if oldText exists
	if !strings.Contains(fileContent, oldText) {
		return "", fmt.Errorf("old_text not found in %s. No similar text found. Verify the file content.", filePath)
	}

	// Count occurrences to provide helpful feedback
	count := strings.Count(fileContent, oldText)
	if count > 1 {
		return "", fmt.Errorf("warning: old_text appears %d times. Please provide more context to make it unique.", count)
	}

	// Replace the content
	newContent := strings.Replace(fileContent, oldText, newText, 1)

	// Write the new content back to the file
	if err := os.WriteFile(filePath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("error writing file: %w", err)
	}

	return fmt.Sprintf("Successfully edited %s - replaced %d characters with %d characters", filePath, len(oldText), len(newText)), nil
}