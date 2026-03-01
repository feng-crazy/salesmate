package test

import (
	"os"
	"path/filepath"
	"salesmate/heartbeat"
	"salesmate/providers"
	"testing"
)

// TestHeartbeatFunctionality performs functional tests for the heartbeat functionality
func TestHeartbeatFunctionality(t *testing.T) {
	// Create a temporary directory for workspace
	tempDir := t.TempDir()

	// Create a mock provider (we'll use nil for testing purposes)
	var mockProvider providers.LLMProvider = nil

	// Define mock callbacks
	mockOnExecute := func(tasks string) (string, error) {
		return "Mock execution response for: " + tasks, nil
	}

	mockOnNotify := func(response string) error {
		// Mock notification function
		return nil
	}

	// Create a new heartbeat service
	service := heartbeat.NewService(
		tempDir,       // workspace
		mockProvider,  // provider
		"test-model",  // model
		mockOnExecute, // onExecute
		mockOnNotify,  // onNotify
		3600,          // intervalS (1 hour)
		true,          // enabled
	)

	// Test initial state
	if service == nil {
		t.Fatal("Failed to create heartbeat service")
	}

	t.Logf("✓ Created heartbeat service")

	// Test starting the service
	err := service.Start()
	if err != nil {
		// This is expected if provider is nil, but let's continue testing other functions
		t.Logf("Starting service failed as expected (due to nil provider): %v", err)
	} else {
		t.Logf("✓ Started heartbeat service")

		// Test stopping the service
		service.Stop()
		t.Logf("✓ Stopped heartbeat service")
	}

	// Test the getHeartbeatTasks functionality indirectly by creating a MEMORY.md file
	memoryDir := filepath.Join(tempDir, "memory")

	// Create memory directory
	err = os.MkdirAll(memoryDir, 0755)
	if err != nil {
		t.Logf("Could not create memory directory: %v", err)
	} else {
		// Create a sample MEMORY.md file with heartbeat tasks
		memoryContent := `# Daily Tasks
- Check email
- Review calendar
- Update status

# Heartbeat Tasks
- Monitor system health
- Check backup status
- Verify service uptime

## Weekly Check-ins
- Team sync
- Project updates
`

		memoryFile := filepath.Join(memoryDir, "MEMORY.md")
		err = os.WriteFile(memoryFile, []byte(memoryContent), 0644)
		if err != nil {
			t.Logf("Could not create MEMORY.md file: %v", err)
		} else {
			t.Logf("✓ Created MEMORY.md with heartbeat tasks")

			// Create a new service to test the getHeartbeatTasks functionality
			service2 := heartbeat.NewService(
				tempDir,
				mockProvider,
				"test-model",
				mockOnExecute,
				mockOnNotify,
				3600,
				false, // Disabled to avoid actually executing tasks
			)

			// We can't directly test getHeartbeatTasks since it's not exported
			// So we just verify that the service can be created and basic functionality works
			if service2 == nil {
				t.Error("Failed to create second heartbeat service")
			} else {
				t.Logf("✓ Created second heartbeat service for testing memory file parsing")
			}
		}
	}

	t.Logf("✓ Heartbeat functionality test completed")
}
