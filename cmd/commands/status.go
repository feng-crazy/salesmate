package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"salesmate/config"

	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show salesmate status",
	Long:  `Show the status of salesmate installation and configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting home directory: %v\n", err)
			os.Exit(1)
		}

		configPath := filepath.Join(homeDir, ".salesmate", "config.yaml")
		workspace := cfg.GetWorkspacePath()

		fmt.Println("🐈 salesmate Status")
		fmt.Println()

		fmt.Printf("Config: %s %s\n", configPath, checkMark(fileExists(configPath)))
		fmt.Printf("Workspace: %s %s\n", workspace, checkMark(dirExists(workspace)))

		if fileExists(configPath) {
			fmt.Printf("Model: %s\n", cfg.Agents.Defaults.Model)

			// Check API keys
			apiKey := cfg.Providers.GetAPIKey(cfg.Agents.Defaults.Model)
			if apiKey != "" {
				fmt.Printf("API Key: %s (found)\n", trimApiKey(apiKey))
			} else {
				fmt.Println("API Key: [not set]")
			}
		}

		fmt.Println()
		fmt.Println("Features:")
		fmt.Printf("  Agent: %s\n", checkMark(true))
		fmt.Printf("  Tools: %s\n", checkMark(true))
		fmt.Printf("  Sessions: %s\n", checkMark(true))
		fmt.Printf("  Channels: %s\n", checkMark(len(cfg.Channels.Telegram.Token) > 0))
		fmt.Printf("  Cron: %s\n", checkMark(true))
		fmt.Printf("  Memory: %s\n", checkMark(dirExists(filepath.Join(workspace, "memory"))))
		fmt.Printf("  Heartbeat: %s\n", checkMark(cfg.Gateway.Heartbeat.Enabled))
	},
}

// Helper function to check if file exists
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// Helper function to check if directory exists
func dirExists(dirname string) bool {
	info, err := os.Stat(dirname)
	if os.IsNotExist(err) {
		return false
	}
	return info.IsDir()
}

// Helper function to return checkmark or cross
func checkMark(ok bool) string {
	if ok {
		return "[✓]"
	}
	return "[✗]"
}

// Helper function to trim API key for display
func trimApiKey(key string) string {
	if len(key) > 10 {
		return key[:10] + "..."
	}
	return key
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
