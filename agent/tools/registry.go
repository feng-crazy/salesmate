package tools

import (
	"fmt"
	"os"
	"path/filepath"
)

// Tool defines the interface for a tool
type Tool interface {
	Name() string
	Description() string
	Call(args map[string]interface{}) (string, error)
}

// ReadFileTool implements a tool to read files
type ReadFileTool struct {
	workspace   string
	allowedDir  string  // If set, restricts operations to this directory
}

// NewReadFileTool creates a new read file tool
func NewReadFileTool(workspace string, allowedDir string) *ReadFileTool {
	return &ReadFileTool{
		workspace:  workspace,
		allowedDir: allowedDir,
	}
}

// Name returns the name of the tool
func (t *ReadFileTool) Name() string {
	return "read_file"
}

// Description returns the description of the tool
func (t *ReadFileTool) Description() string {
	return "Read the content of a file"
}

// Call executes the tool with the given arguments
func (t *ReadFileTool) Call(args map[string]interface{}) (string, error) {
	filePath, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'path' argument")
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

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	return string(content), nil
}

// WriteFileTool implements a tool to write files
type WriteFileTool struct {
	workspace   string
	allowedDir  string  // If set, restricts operations to this directory
}

// NewWriteFileTool creates a new write file tool
func NewWriteFileTool(workspace string, allowedDir string) *WriteFileTool {
	return &WriteFileTool{
		workspace:  workspace,
		allowedDir: allowedDir,
	}
}

// Name returns the name of the tool
func (t *WriteFileTool) Name() string {
	return "write_file"
}

// Description returns the description of the tool
func (t *WriteFileTool) Description() string {
	return "Write content to a file"
}

// Call executes the tool with the given arguments
func (t *WriteFileTool) Call(args map[string]interface{}) (string, error) {
	filePath, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'path' argument")
	}

	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'content' argument")
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

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("error creating directory: %w", err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("error writing file: %w", err)
	}

	return fmt.Sprintf("Successfully wrote %d characters to %s", len(content), filePath), nil
}

// ListDirTool implements a tool to list directory contents
type ListDirTool struct {
	workspace   string
	allowedDir  string  // If set, restricts operations to this directory
}

// NewListDirTool creates a new list directory tool
func NewListDirTool(workspace string, allowedDir string) *ListDirTool {
	return &ListDirTool{
		workspace:  workspace,
		allowedDir: allowedDir,
	}
}

// Name returns the name of the tool
func (t *ListDirTool) Name() string {
	return "list_directory"
}

// Description returns the description of the tool
func (t *ListDirTool) Description() string {
	return "List the contents of a directory"
}

// Call executes the tool with the given arguments
func (t *ListDirTool) Call(args map[string]interface{}) (string, error) {
	dirPath, ok := args["path"].(string)
	if !ok {
		// Use current directory if no path is provided
		dirPath = "."
	}

	// Verify path is allowed if restriction is in place
	if t.allowedDir != "" {
		absPath, err := filepath.Abs(dirPath)
		if err != nil {
			return "", fmt.Errorf("error resolving path: %w", err)
		}
		absAllowedDir, err := filepath.Abs(t.allowedDir)
		if err != nil {
			return "", fmt.Errorf("error resolving allowed directory: %w", err)
		}

		if !filepath.HasPrefix(absPath, absAllowedDir) {
			return "", fmt.Errorf("path %s is outside allowed directory %s", dirPath, t.allowedDir)
		}
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", fmt.Errorf("error reading directory: %w", err)
	}

	result := fmt.Sprintf("Contents of %s:\n", dirPath)
	for _, entry := range entries {
		entryType := "file"
		if entry.IsDir() {
			entryType = "dir"
		}
		result += fmt.Sprintf("  %s (%s)\n", entry.Name(), entryType)
	}

	return result, nil
}

// ToolRegistry manages available tools
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry
func (tr *ToolRegistry) Register(tool Tool) {
	tr.tools[tool.Name()] = tool
}

// Get retrieves a tool by name
func (tr *ToolRegistry) Get(name string) Tool {
	return tr.tools[name]
}

// GetDefinitions returns tool definitions for API
func (tr *ToolRegistry) GetDefinitions() []interface{} {
	definitions := make([]interface{}, 0, len(tr.tools))

	for _, tool := range tr.tools {
		def := map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name(),
				"description": tool.Description(),
				"parameters": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						// Define parameters based on tool type
						// For now, we'll use a simple structure
					},
					"required": []string{}, // Define required parameters based on tool
				},
			},
		}
		definitions = append(definitions, def)
	}

	return definitions
}

// Execute runs a tool with the given arguments
func (tr *ToolRegistry) Execute(name string, args map[string]interface{}) (string, error) {
	tool, exists := tr.tools[name]
	if !exists {
		return "", fmt.Errorf("unknown tool: %s", name)
	}

	return tool.Call(args)
}