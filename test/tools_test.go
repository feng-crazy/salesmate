package test

import (
	"os"
	"path/filepath"
	"salesmate/agent/tools"
	"testing"
)

// TestToolsFunctionality performs functional tests for the tools functionality
func TestToolsFunctionality(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Test ReadFileTool
	readTool := tools.NewReadFileTool(tempDir, "")

	// Create a test file
	testFilePath := filepath.Join(tempDir, "functional_test.txt")
	testContent := "Hello from functional test!"
	err := os.WriteFile(testFilePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test reading the file
	params := map[string]interface{}{
		"path": testFilePath,
	}
	result, err := readTool.Call(params)
	if err != nil {
		t.Errorf("ReadFileTool failed: %v", err)
	} else if result != testContent {
		t.Errorf("ReadFileTool returned unexpected result: got %s, want %s", result, testContent)
	} else {
		t.Logf("✓ ReadFileTool functional test passed")
	}

	// Test WriteFileTool
	writeTool := tools.NewWriteFileTool(tempDir, "")
	writeParams := map[string]interface{}{
		"path":    filepath.Join(tempDir, "output_func_test.txt"),
		"content": "Written by salesmate functional test!",
	}
	writeResult, err := writeTool.Call(writeParams)
	if err != nil {
		t.Errorf("WriteFileTool failed: %v", err)
	} else {
		t.Logf("WriteFileTool result: %v", writeResult)

		// Verify the file was written
		outputPath := filepath.Join(tempDir, "output_func_test.txt")
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Errorf("Failed to read written file: %v", err)
		} else if string(content) != "Written by salesmate functional test!" {
			t.Errorf("File content mismatch: got %s, want %s", string(content), "Written by salesmate functional test!")
		} else {
			t.Logf("✓ WriteFileTool functional test passed")
		}
	}

	// Test ListDirTool
	listTool := tools.NewListDirTool(tempDir, "")
	listParams := map[string]interface{}{
		"path": tempDir,
	}
	listResult, err := listTool.Call(listParams)
	if err != nil {
		t.Errorf("ListDirTool failed: %v", err)
	} else if listResult == "" {
		t.Error("ListDirTool returned empty result")
	} else {
		t.Logf("✓ ListDirTool functional test passed: Contains %d characters of directory listing", len(listResult))
	}
}
