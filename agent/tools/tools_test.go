package tools_test

import (
	"os"
	"path/filepath"
	"salesmate/agent/tools"
	"strings"
	"testing"
)

func TestFileTools(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()

	// Test ReadFileTool - allow all paths by setting allowedDir to ""
	readTool := tools.NewReadFileTool(tempDir, "")

	// Create a test file inside the temp directory
	testFilePath := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, salesmate!"
	err := os.WriteFile(testFilePath, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test reading the file - pass the full path to the file
	params := map[string]interface{}{
		"path": testFilePath,
	}
	result, err := readTool.Call(params)
	if err != nil {
		t.Errorf("ReadFileTool failed: %v", err)
	} else if result != testContent {
		t.Errorf("ReadFileTool returned unexpected result: got %s, want %s", result, testContent)
	} else {
		t.Logf("ReadFileTool result: %v", result)
	}

	// Test WriteFileTool
	writeTool := tools.NewWriteFileTool(tempDir, "") // Allow all paths
	writeParams := map[string]interface{}{
		"path":    filepath.Join(tempDir, "output.txt"),
		"content": "Written by salesmate!",
	}
	writeResult, err := writeTool.Call(writeParams)
	if err != nil {
		t.Errorf("WriteFileTool failed: %v", err)
	} else if !strings.Contains(writeResult, "Successfully") {
		t.Error("WriteFileTool returned unexpected result")
	} else {
		t.Logf("WriteFileTool result: %v", writeResult)

		// Verify the file was written
		outputPath := filepath.Join(tempDir, "output.txt")
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Errorf("Failed to read written file: %v", err)
		} else if string(content) != "Written by salesmate!" {
			t.Errorf("File content mismatch: got %s, want %s", string(content), "Written by salesmate!")
		} else {
			t.Log("WriteFileTool verified successfully")
		}
	}

	// Test ListDirTool
	listTool := tools.NewListDirTool(tempDir, "") // Allow all paths
	listParams := map[string]interface{}{
		"path": tempDir, // Use the temp directory
	}
	listResult, err := listTool.Call(listParams)
	if err != nil {
		t.Errorf("ListDirTool failed: %v", err)
	} else if !strings.Contains(listResult, "test.txt") || !strings.Contains(listResult, "output.txt") {
		t.Errorf("ListDirTool result doesn't contain expected files: %v", listResult)
	} else {
		t.Logf("ListDirTool result: %v", listResult)
	}
}
