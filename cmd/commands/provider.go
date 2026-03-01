package commands

import (
	"fmt"
	"os"

	"salesmate/config"

	"github.com/spf13/cobra"
)

// providerCmd represents the provider command
var providerCmd = &cobra.Command{
	Use:   "provider",
	Short: "Manage providers",
	Long:  `Manage LLM providers and authentication.`,
}

// providerLoginCmd represents the provider login command
var providerLoginCmd = &cobra.Command{
	Use:   "login [provider]",
	Short: "Authenticate with a provider",
	Long:  `Authenticate with an OAuth provider.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		provider := args[0]

		// Load configuration
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Provider: %s\n", provider)

		// For now, we just display configuration info
		// In a real implementation, this would handle OAuth flows
		switch provider {
		case "openrouter", "open-router":
			apiKey := cfg.Providers.OpenRouter.APIKey
			if apiKey != "" {
				fmt.Println("OpenRouter is already configured")
			} else {
				fmt.Println("Please set your OpenRouter API key in the config file")
			}
		case "anthropic":
			apiKey := cfg.Providers.Anthropic.APIKey
			if apiKey != "" {
				fmt.Println("Anthropic is already configured")
			} else {
				fmt.Println("Please set your Anthropic API key in the config file")
			}
		case "openai":
			apiKey := cfg.Providers.OpenAI.APIKey
			if apiKey != "" {
				fmt.Println("OpenAI is already configured")
			} else {
				fmt.Println("Please set your OpenAI API key in the config file")
			}
		default:
			fmt.Printf("Unknown provider: %s\n", provider)
			fmt.Println("Supported providers: openrouter, anthropic, openai")
		}
	},
}

func init() {
	rootCmd.AddCommand(providerCmd)

	// Add subcommands
	providerCmd.AddCommand(providerLoginCmd)
}
