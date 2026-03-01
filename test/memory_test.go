package test

import (
	"salesmate/agent/memory"
	"strings"
	"testing"
)

// TestMemoryFunctionality performs functional tests for the memory functionality
func TestMemoryFunctionality(t *testing.T) {
	// Create a temporary directory for memory storage
	tempDir := t.TempDir()

	// Create a memory store
	store := memory.NewMemoryStore(tempDir)

	// Test writing to long-term memory
	longTermContent := "# Test Memory\nThis is a test entry for long-term memory.\n\n## Additional Notes\nSome additional information."

	err := store.WriteLongTerm(longTermContent)
	if err != nil {
		t.Errorf("Failed to write to long-term memory: %v", err)
	} else {
		t.Logf("✓ Wrote content to long-term memory")
	}

	// Test reading from long-term memory
	retrievedContent, err := store.ReadLongTerm()
	if err != nil {
		t.Errorf("Failed to read from long-term memory: %v", err)
	} else if retrievedContent != longTermContent {
		t.Errorf("Retrieved content doesn't match written content")
	} else {
		t.Logf("✓ Successfully read content from long-term memory")
	}

	// Test appending to history
	historyEntry := "Test history entry: user asked about Go programming."
	err = store.AppendHistory(historyEntry)
	if err != nil {
		t.Errorf("Failed to append to history: %v", err)
	} else {
		t.Logf("✓ Appended entry to history")
	}

	// Test reading history
	historyContent, err := store.ReadHistory()
	if err != nil {
		t.Errorf("Failed to read history: %v", err)
	} else if !strings.Contains(historyContent, historyEntry) {
		t.Errorf("History doesn't contain the appended entry")
	} else {
		t.Logf("✓ Successfully read history, contains %d characters", len(historyContent))
	}

	// Test getting memory context
	context, err := store.GetMemoryContext()
	if err != nil {
		t.Errorf("Failed to get memory context: %v", err)
	} else if context == "" {
		t.Error("Memory context is empty")
	} else if !strings.Contains(context, "Test Memory") {
		t.Errorf("Memory context doesn't contain expected content")
	} else {
		t.Logf("✓ Successfully retrieved memory context: %s", truncateString(context, 50)+"...")
	}

	t.Logf("✓ Memory functionality test completed")
}

// TestMemoryWithContext performs functional tests with context information
func TestMemoryWithContext(t *testing.T) {
	// Create a temporary directory for memory storage
	tempDir := t.TempDir()

	// Create a memory store
	store := memory.NewMemoryStore(tempDir)

	// Write initial long-term memory
	initialMemory := `# Project Context
Project: salesmate AI Assistant
Purpose: Personal AI assistant with multiple channel support
Features: Chat channels, scheduling, memory, tools
`

	err := store.WriteLongTerm(initialMemory)
	if err != nil {
		t.Fatalf("Failed to write initial memory: %v", err)
	}

	// Append multiple history entries
	historyEntries := []string{
		"User asked about project status",
		"System reported current features",
		"User requested new channel setup",
		"System provided configuration instructions",
	}

	for i, entry := range historyEntries {
		err = store.AppendHistory(entry)
		if err != nil {
			t.Errorf("Failed to append history entry %d: %v", i, err)
		}
	}

	// Read back the history
	history, err := store.ReadHistory()
	if err != nil {
		t.Errorf("Failed to read accumulated history: %v", err)
	} else {
		count := 0
		for _, entry := range historyEntries {
			if strings.Contains(history, entry) {
				count++
			}
		}
		if count != len(historyEntries) {
			t.Errorf("Expected to find %d entries in history, found %d", len(historyEntries), count)
		} else {
			t.Logf("✓ Found all %d history entries in accumulated history", len(historyEntries))
		}
	}

	// Get memory context and verify it contains project info
	context, err := store.GetMemoryContext()
	if err != nil {
		t.Errorf("Failed to get memory context: %v", err)
	} else if !strings.Contains(context, "Project: salesmate AI Assistant") {
		t.Error("Memory context doesn't contain project information")
	} else {
		t.Logf("✓ Memory context contains expected project information")
	}

	// Test with empty long-term memory
	emptyTempDir := t.TempDir()
	emptyStore := memory.NewMemoryStore(emptyTempDir)

	emptyContext, err := emptyStore.GetMemoryContext()
	if err != nil {
		t.Errorf("Failed to get empty memory context: %v", err)
	} else if emptyContext != "" {
		t.Error("Empty memory context should be empty string")
	} else {
		t.Logf("✓ Empty memory context correctly returns empty string")
	}

	t.Logf("✓ Memory context functionality test completed")
}

// Helper function to truncate string
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
