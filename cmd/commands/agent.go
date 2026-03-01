package commands

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"salesmate/agent"
	"salesmate/config"

	"github.com/spf13/cobra"
)

// agentCmd represents the agent command
var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Interact with the AI agent",
	Long: `Interact with the AI agent.

This command allows you to chat with the AI agent either in interactive mode
or by providing a single message.`,
	Run: func(cmd *cobra.Command, args []string) {
		sessionID, _ := cmd.Flags().GetString("session")
		message, _ := cmd.Flags().GetString("message")
		markdown, _ := cmd.Flags().GetBool("markdown")
		showLogs, _ := cmd.Flags().GetBool("logs")

		// Set up logging based on flag
		if !showLogs {
			log.SetOutput(os.Stdout)
		}

		// Load configuration
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		// Initialize agent
		agentLoop, err := agent.NewAgentLoop(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing agent: %v\n", err)
			os.Exit(1)
		}

		if message != "" {
			// Single message mode
			response, err := agentLoop.ProcessDirect(message, sessionID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error processing message: %v\n", err)
				os.Exit(1)
			}
			if markdown {
				fmt.Printf("🐈 salesmate\n%s\n", response)
			} else {
				fmt.Printf("%s\n", response)
			}
		} else {
			// Interactive mode
			fmt.Println("Interactive mode - type 'exit' or 'quit' to quit")
			scanner := bufio.NewScanner(os.Stdin)

			for {
				fmt.Print("You: ")
				if !scanner.Scan() {
					break // End of input
				}

				input := scanner.Text()
				input = strings.TrimSpace(input)

				if input == "" {
					continue
				}

				// Check for exit commands
				if strings.ToLower(input) == "exit" || strings.ToLower(input) == "quit" {
					fmt.Println("Goodbye!")
					break
				}

				response, err := agentLoop.ProcessDirect(input, sessionID)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error processing message: %v\n", err)
					continue
				}

				if markdown {
					fmt.Printf("🐈 salesmate\n%s\n", response)
				} else {
					fmt.Printf("%s\n", response)
				}
			}

			if err := scanner.Err(); err != nil {
				fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(agentCmd)

	agentCmd.Flags().StringP("message", "m", "", "Message to send to the agent")
	agentCmd.Flags().StringP("session", "s", "cli:direct", "Session ID")
	agentCmd.Flags().Bool("markdown", true, "Render assistant output as Markdown")
	agentCmd.Flags().Bool("logs", false, "Show salesmate runtime logs during chat")
}
