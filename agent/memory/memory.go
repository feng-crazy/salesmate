package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// MemoryStore represents the two-layer memory system
type MemoryStore struct {
	memoryDir   string
	memoryFile  string
	historyFile string
}

// NewMemoryStore creates a new memory store
func NewMemoryStore(workspace string) *MemoryStore {
	memoryDir := filepath.Join(workspace, "memory")

	// Ensure directory exists
	os.MkdirAll(memoryDir, 0755)

	return &MemoryStore{
		memoryDir:   memoryDir,
		memoryFile:  filepath.Join(memoryDir, "MEMORY.md"),
		historyFile: filepath.Join(memoryDir, "HISTORY.md"),
	}
}

// ReadLongTerm reads the long-term memory
func (ms *MemoryStore) ReadLongTerm() (string, error) {
	content, err := os.ReadFile(ms.memoryFile)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// WriteLongTerm writes to long-term memory
func (ms *MemoryStore) WriteLongTerm(content string) error {
	return os.WriteFile(ms.memoryFile, []byte(content), 0644)
}

// AppendHistory appends an entry to the history log
func (ms *MemoryStore) AppendHistory(entry string) error {
	f, err := os.OpenFile(ms.historyFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Remove trailing whitespace and add timestamp and newlines
	cleanEntry := strings.TrimRight(entry, " \t\n\r")
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	_, err = fmt.Fprintf(f, "[%s] %s\n\n", timestamp, cleanEntry)
	return err
}

// GetMemoryContext gets the memory context for inclusion in prompts
func (ms *MemoryStore) GetMemoryContext() (string, error) {
	longTerm, err := ms.ReadLongTerm()
	if err != nil {
		return "", err
	}

	if longTerm != "" {
		return fmt.Sprintf("## Long-term Memory\n%s", longTerm), nil
	}

	return "", nil
}

// ReadHistory reads the history file
func (ms *MemoryStore) ReadHistory() (string, error) {
	content, err := os.ReadFile(ms.historyFile)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return string(content), nil
}