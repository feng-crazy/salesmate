package heartbeat

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"salesmate/providers"
)

// Service represents the heartbeat service
type Service struct {
	intervalS int
	enabled   bool
	workspace string
	provider  providers.LLMProvider
	model     string
	onExecute func(tasks string) (string, error)
	onNotify  func(response string) error
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.Mutex
	running   bool
}

// NewService creates a new heartbeat service
func NewService(
	workspace string,
	provider providers.LLMProvider,
	model string,
	onExecute func(tasks string) (string, error),
	onNotify func(response string) error,
	intervalS int,
	enabled bool,
) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	return &Service{
		workspace: workspace,
		provider:  provider,
		model:     model,
		onExecute: onExecute,
		onNotify:  onNotify,
		intervalS: intervalS,
		enabled:   enabled,
		ctx:       ctx,
		cancel:    cancel,
	}
}

// Start starts the heartbeat service
func (s *Service) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("heartbeat service is already running")
	}

	if !s.enabled {
		return fmt.Errorf("heartbeat service is disabled")
	}

	s.running = true

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		ticker := time.NewTicker(time.Duration(s.intervalS) * time.Second)
		defer ticker.Stop()

		// Execute immediately on startup
		s.executeHeartbeat()

		for {
			select {
			case <-ticker.C:
				s.executeHeartbeat()
			case <-s.ctx.Done():
				return
			}
		}
	}()

	return nil
}

// Stop stops the heartbeat service
func (s *Service) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.cancel()
	s.wg.Wait()
	s.running = false
}

// executeHeartbeat performs the heartbeat logic
func (s *Service) executeHeartbeat() {
	if !s.enabled {
		return
	}

	tasks, err := s.getHeartbeatTasks()
	if err != nil {
		fmt.Printf("Error getting heartbeat tasks: %v\n", err)
		return
	}

	if tasks == "" {
		// No tasks to execute
		return
	}

	if s.onExecute != nil {
		response, err := s.onExecute(tasks)
		if err != nil {
			fmt.Printf("Error executing heartbeat tasks: %v\n", err)
			return
		}

		if response != "" && s.onNotify != nil {
			if notifyErr := s.onNotify(response); notifyErr != nil {
				fmt.Printf("Error notifying heartbeat response: %v\n", notifyErr)
			}
		}
	}
}

// getHeartbeatTasks reads heartbeat tasks from MEMORY.md file
func (s *Service) getHeartbeatTasks() (string, error) {
	memoryPath := filepath.Join(s.workspace, "memory", "MEMORY.md")

	content, err := os.ReadFile(memoryPath)
	if os.IsNotExist(err) {
		// If MEMORY.md doesn't exist, check for other potential memory files
		files, readDirErr := os.ReadDir(filepath.Join(s.workspace, "memory"))
		if readDirErr != nil {
			return "", nil // No memory directory or can't read it, return empty
		}

		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
				memoryPath = filepath.Join(s.workspace, "memory", file.Name())
				content, err = os.ReadFile(memoryPath)
				if err == nil {
					break
				}
			}
		}

		if err != nil {
			return "", nil // Could not find any memory files
		}
	} else if err != nil {
		return "", fmt.Errorf("failed to read memory file: %w", err)
	}

	contentStr := string(content)

	// Extract heartbeat tasks from the memory file
	// Look for specific sections that might contain heartbeat tasks
	lines := strings.Split(contentStr, "\n")
	var heartbeatTasks []string

	inHeartbeatSection := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for heartbeat-related section headers
		if strings.Contains(strings.ToLower(trimmed), "heartbeat") ||
			strings.Contains(strings.ToLower(trimmed), "check-in") ||
			strings.Contains(strings.ToLower(trimmed), "periodic") ||
			strings.Contains(strings.ToLower(trimmed), "daily") {
			inHeartbeatSection = true
			continue
		}

		// Check for section ends
		if strings.HasPrefix(trimmed, "# ") && inHeartbeatSection {
			// New section starts, so end the heartbeat section
			break
		}

		if inHeartbeatSection && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			heartbeatTasks = append(heartbeatTasks, trimmed)
		}
	}

	if len(heartbeatTasks) == 0 {
		// If no specific heartbeat section found, check for general periodic tasks
		inTasksSection := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)

			if strings.Contains(strings.ToLower(trimmed), "task") ||
				strings.Contains(strings.ToLower(trimmed), "todo") ||
				strings.Contains(strings.ToLower(trimmed), "check") {
				inTasksSection = true
				continue
			}

			if strings.HasPrefix(trimmed, "# ") && inTasksSection {
				// New section starts, so end the tasks section
				break
			}

			if inTasksSection && trimmed != "" && !strings.HasPrefix(trimmed, "#") &&
				(strings.Contains(strings.ToLower(trimmed), "daily") ||
					strings.Contains(strings.ToLower(trimmed), "weekly") ||
					strings.Contains(strings.ToLower(trimmed), "periodic") ||
					strings.Contains(strings.ToLower(trimmed), "every")) {
				heartbeatTasks = append(heartbeatTasks, trimmed)
			}
		}
	}

	return strings.Join(heartbeatTasks, "\n"), nil
}
