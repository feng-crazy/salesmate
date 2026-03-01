package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// onboardCmd represents the onboard command
var onboardCmd = &cobra.Command{
	Use:   "onboard",
	Short: "Initialize salesmate configuration and workspace",
	Long: `Initialize salesmate configuration and workspace.

This command creates the necessary configuration files and workspace directory
for salesmate to operate.`,
	Run: func(cmd *cobra.Command, args []string) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			os.Exit(1)
		}

		configDir := filepath.Join(homeDir, ".salesmate")
		configPath := filepath.Join(configDir, "config.yaml")

		// Create config directory
		if err := os.MkdirAll(configDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating config directory: %v\n", err)
			os.Exit(1)
		}

		// Check if config already exists
		if _, err := os.Stat(configPath); err == nil {
			fmt.Printf("Config already exists at %s\n", configPath)
			fmt.Print("Overwrite with defaults? (y/N): ")
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Keeping existing config.")
				return
			}
		}

		// Create default config
		defaultConfig := `# salesmate configuration
agents:
  defaults:
    model: "anthropic/claude-opus-4-5"
    max_tokens: 8192
    temperature: 0.1
    max_tool_iterations: 40
    memory_window: 100

providers:
  openrouter:
    api_key: ""
  anthropic:
    api_key: ""
  openai:
    api_key: ""

channels:
  send_progress: true
  send_tool_hints: false
  telegram:
    enabled: false
    token: ""
    allow_from: []
  discord:
    enabled: false
    token: ""
    allow_from: []

gateway:
  heartbeat:
    enabled: true
    interval_s: 1800  # 30 minutes

tools:
  exec:
    timeout: 60
  restrict_to_workspace: false
`

		if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing config file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✓ Created config at %s\n", configPath)

		// Create workspace directory
		workspacePath := filepath.Join(configDir, "workspace")
		if err := os.MkdirAll(workspacePath, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating workspace directory: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("✓ Created workspace at %s\n", workspacePath)

		// Create default workspace files
		createWorkspaceTemplates(workspacePath)

		fmt.Println("\n🐈 salesmate is ready!")
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Add your API key to ~/.salesmate/config.yaml")
		fmt.Println("     Get one at: https://openrouter.ai/keys")
		fmt.Println("  2. Chat: salesmate agent -m \"Hello!\"")
	},
}

func createWorkspaceTemplates(workspace string) {
	// Create memory directory
	memoryDir := filepath.Join(workspace, "memory")
	if err := os.MkdirAll(memoryDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating memory directory: %v\n", err)
		return
	}

	// Create default memory files
	memoryContent := `# Memory
## Important Information
`

	historyContent := `# History
`

	if err := os.WriteFile(filepath.Join(memoryDir, "MEMORY.md"), []byte(memoryContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating MEMORY.md: %v\n", err)
	}

	if err := os.WriteFile(filepath.Join(memoryDir, "HISTORY.md"), []byte(historyContent), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating HISTORY.md: %v\n", err)
	}

	// Create skills directory
	skillsDir := filepath.Join(workspace, "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating skills directory: %v\n", err)
	}

	fmt.Println("  Created default workspace files")
}

func init() {
	rootCmd.AddCommand(onboardCmd)
}
