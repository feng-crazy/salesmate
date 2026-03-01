package commands

import (
	"fmt"
	"os"

	"salesmate/config"

	"github.com/spf13/cobra"
)

// channelsCmd represents the channels command
var channelsCmd = &cobra.Command{
	Use:   "channels",
	Short: "Manage chat channels",
	Long:  `Manage chat channels and their configurations.`,
}

// channelsStatusCmd represents the channels status command
var channelsStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show channel status",
	Long:  `Show the status of configured channels.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration
		cfg, err := config.LoadConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Channel Status:")
		fmt.Printf("  %-12s %s\n", "Channel", "Enabled")
		fmt.Printf("  %-12s %s\n", "-------", "-------")

		// WhatsApp
		enabled := "✗"
		if cfg.Channels.WhatsApp.Enabled {
			enabled = "✓"
		}
		fmt.Printf("  %-12s %s\n", "WhatsApp", enabled)

		// Telegram
		enabled = "✗"
		if cfg.Channels.Telegram.Enabled {
			enabled = "✓"
		}
		telegramConfig := "[not configured]"
		if cfg.Channels.Telegram.Token != "" {
			if len(cfg.Channels.Telegram.Token) > 10 {
				telegramConfig = cfg.Channels.Telegram.Token[:10] + "..."
			} else {
				telegramConfig = cfg.Channels.Telegram.Token
			}
		}
		fmt.Printf("  %-12s %s (%s)\n", "Telegram", enabled, telegramConfig)

		// Discord
		enabled = "✗"
		if cfg.Channels.Discord.Enabled {
			enabled = "✓"
		}
		discordConfig := "[not configured]"
		if cfg.Channels.Discord.Token != "" {
			if len(cfg.Channels.Discord.Token) > 10 {
				discordConfig = cfg.Channels.Discord.Token[:10] + "..."
			} else {
				discordConfig = cfg.Channels.Discord.Token
			}
		}
		fmt.Printf("  %-12s %s (%s)\n", "Discord", enabled, discordConfig)

		// Feishu
		enabled = "✗"
		if cfg.Channels.Feishu.Enabled {
			enabled = "✓"
		}
		fmt.Printf("  %-12s %s\n", "Feishu", enabled)

		// Mochat
		enabled = "✗"
		if cfg.Channels.Mochat.Enabled {
			enabled = "✓"
		}
		mochatConfig := "[not configured]"
		if cfg.Channels.Mochat.BaseURL != "" {
			mochatConfig = cfg.Channels.Mochat.BaseURL
		}
		fmt.Printf("  %-12s %s (%s)\n", "Mochat", enabled, mochatConfig)

		// Slack
		enabled = "✗"
		if cfg.Channels.Slack.Enabled {
			enabled = "✓"
		}
		slackConfig := "[not configured]"
		if cfg.Channels.Slack.BotToken != "" || cfg.Channels.Slack.AppToken != "" {
			slackConfig = "configured"
		}
		fmt.Printf("  %-12s %s (%s)\n", "Slack", enabled, slackConfig)

		// DingTalk
		enabled = "✗"
		if cfg.Channels.DingTalk.Enabled {
			enabled = "✓"
		}
		dingtalkConfig := "[not configured]"
		if cfg.Channels.DingTalk.ClientID != "" {
			if len(cfg.Channels.DingTalk.ClientID) > 10 {
				dingtalkConfig = cfg.Channels.DingTalk.ClientID[:10] + "..."
			} else {
				dingtalkConfig = cfg.Channels.DingTalk.ClientID
			}
		}
		fmt.Printf("  %-12s %s (%s)\n", "DingTalk", enabled, dingtalkConfig)

		// QQ
		enabled = "✗"
		if cfg.Channels.QQ.Enabled {
			enabled = "✓"
		}
		qqConfig := "[not configured]"
		if cfg.Channels.QQ.AppID != "" {
			if len(cfg.Channels.QQ.AppID) > 10 {
				qqConfig = cfg.Channels.QQ.AppID[:10] + "..."
			} else {
				qqConfig = cfg.Channels.QQ.AppID
			}
		}
		fmt.Printf("  %-12s %s (%s)\n", "QQ", enabled, qqConfig)

		// Email
		enabled = "✗"
		if cfg.Channels.Email.Enabled {
			enabled = "✓"
		}
		emailConfig := "[not configured]"
		if cfg.Channels.Email.IMAPHost != "" {
			emailConfig = cfg.Channels.Email.IMAPHost
		}
		fmt.Printf("  %-12s %s (%s)\n", "Email", enabled, emailConfig)
	},
}

func init() {
	rootCmd.AddCommand(channelsCmd)

	// Add subcommands
	channelsCmd.AddCommand(channelsStatusCmd)
}
