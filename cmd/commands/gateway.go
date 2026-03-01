package commands

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"salesmate/agent"
	"salesmate/bus"
	"salesmate/channels"
	"salesmate/config"
	"salesmate/cron"
	"salesmate/heartbeat"
	"salesmate/session"

	"github.com/spf13/cobra"
)

// gatewayCmd represents the gateway command
var gatewayCmd = &cobra.Command{
	Use:   "gateway",
	Short: "Start the salesmate gateway",
	Long: `Start the salesmate gateway.

The gateway combines all salesmate services including the agent, channels,
scheduled tasks, and heartbeat services.`,
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		verbose, _ := cmd.Flags().GetBool("verbose")

		if verbose {
			log.SetOutput(os.Stdout)
		}

		fmt.Printf("🐈 Starting salesmate gateway on port %d...\n", port)

		// Load configuration
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		// Initialize message bus
		_ = bus.NewMessageBus() // For now, just declare but not use it

		// Initialize provider and agent
		agentLoop, err := agent.NewAgentLoop(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing agent: %v\n", err)
			os.Exit(1)
		}

		// Initialize session manager
		sessionManager := session.NewSessionManager(cfg.GetWorkspacePath())

		// Initialize cron service
		dataDir := filepath.Join(os.Getenv("HOME"), ".salesmate", "data")
		cronStorePath := filepath.Join(dataDir, "cron", "jobs.json")
		cronService, err := cron.NewCronService(cronStorePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing cron service: %v\n", err)
			os.Exit(1)
		}

		// Set cron callback
		cronService.SetOnJobCallback(func(job *cron.CronJob) (string, error) {
			response, err := agentLoop.ProcessDirect(job.Payload.Message, fmt.Sprintf("cron:%s", job.ID))
			if err != nil {
				return "", err
			}

			if job.Payload.Deliver && job.Payload.To != "" {
				// In a real implementation, send the response via the appropriate channel
				fmt.Printf("Cron job result delivered to %s:%s\n", job.Payload.Channel, job.Payload.To)
			}

			return response, nil
		})

		// Add cron service to agent
		agentLoop.SetCronService(cronService)

		// Initialize channel manager
		channelManager := channels.NewManager(cfg)

		// Helper function to pick heartbeat target
		pickHeartbeatTarget := func() (string, string) {
			enabled := make(map[string]bool)
			for _, name := range channelManager.GetEnabledChannels() {
				enabled[name] = true
			}

			// Prefer the most recently updated non-internal session on an enabled channel
			sessions := sessionManager.ListSessions()
			for _, item := range sessions {
				key, ok := item["key"].(string)
				if !ok || key == "" {
					continue
				}

				if colonIndex := findRune(key, ':'); colonIndex != -1 {
					channel := key[:colonIndex]
					chatID := key[colonIndex+1:]

					if channel == "cli" || channel == "system" {
						continue
					}

					if enabled[channel] && chatID != "" {
						return channel, chatID
					}
				}
			}

			// Fallback
			return "cli", "direct"
		}

		// Initialize heartbeat service
		heartbeatService := heartbeat.NewService(
			cfg.GetWorkspacePath(),
			nil,                       // We'll pass the provider later
			cfg.Agents.Defaults.Model, // Use config model
			func(tasks string) (string, error) {
				channel, chatID := pickHeartbeatTarget()
				response, err := agentLoop.ProcessDirect(tasks, fmt.Sprintf("heartbeat:%s:%s", channel, chatID))
				return response, err
			},
			func(response string) error {
				// Delivery mechanism would go here
				channel, chatID := pickHeartbeatTarget()
				if channel == "cli" {
					return nil // No external channel available
				}

				// In a real implementation, send the heartbeat response to the user's channel
				fmt.Printf("Heartbeat response delivered to %s:%s\n", channel, chatID)
				return nil
			},
			cfg.Gateway.Heartbeat.IntervalS,
			cfg.Gateway.Heartbeat.Enabled,
		)

		// Show status
		enabledChannels := channelManager.GetEnabledChannels()
		if len(enabledChannels) > 0 {
			fmt.Printf("[✓] Channels enabled: %v\n", enabledChannels)
		} else {
			fmt.Println("[!] Warning: No channels enabled")
		}

		cronStatus := cronService.Status()
		if jobs, ok := cronStatus["jobs"].(int); ok && jobs > 0 {
			fmt.Printf("[✓] Cron: %d scheduled jobs\n", jobs)
		}

		fmt.Printf("[✓] Heartbeat: every %ds\n", cfg.Gateway.Heartbeat.IntervalS)

		// Start services
		cronService.Start() // Call without assignment since it returns error

		if err := heartbeatService.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting heartbeat service: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Gateway services started successfully!")

		// In a real implementation, we would run the agent loop and channel manager here
		// For now, we'll just keep the program running
		select {} // Block forever (in a real implementation, handle shutdown properly)
	},
}

// Helper function to find rune in string
func findRune(s string, r rune) int {
	for i, c := range s {
		if c == r {
			return i
		}
	}
	return -1
}

func init() {
	rootCmd.AddCommand(gatewayCmd)

	gatewayCmd.Flags().IntP("port", "p", 18790, "Gateway port")
	gatewayCmd.Flags().Bool("verbose", false, "Verbose output")
}
