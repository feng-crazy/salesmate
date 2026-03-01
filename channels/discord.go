package channels

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

// DiscordChannel implements the Discord channel
type DiscordChannel struct {
	token        string
	allowedChats []string
	name         string
	running      bool
	session      *discordgo.Session
}

// NewDiscordChannel creates a new Discord channel
func NewDiscordChannel(token string, allowedChats []string) *DiscordChannel {
	return &DiscordChannel{
		token:        token,
		allowedChats: allowedChats,
		name:         "discord",
		running:      false,
	}
}

// Start starts the Discord channel
func (dc *DiscordChannel) Start() error {
	if dc.token == "" {
		return fmt.Errorf("discord token must be configured")
	}

	// Create a new Discord session using the provided bot token
	session, err := discordgo.New("Bot " + dc.token)
	if err != nil {
		return fmt.Errorf("error creating Discord session: %w", err)
	}

	// Try to open the websocket and connect
	err = session.Open()
	if err != nil {
		session.Close() // Clean up session if connection fails
		return fmt.Errorf("error opening Discord session: %w", err)
	}

	dc.session = session
	dc.running = true
	log.Printf("Discord channel started")

	return nil
}

// Stop stops the Discord channel
func (dc *DiscordChannel) Stop() error {
	if dc.session != nil {
		dc.session.Close()
	}
	dc.running = false
	log.Printf("Discord channel stopped")
	return nil
}

// Name returns the channel name
func (dc *DiscordChannel) Name() string {
	return dc.name
}

// Send sends a message to a Discord channel
func (dc *DiscordChannel) Send(chatID, message string) error {
	if !dc.running {
		return fmt.Errorf("discord channel not running")
	}

	if !dc.isAllowed(chatID) {
		return fmt.Errorf("channel %s not allowed", chatID)
	}

	if dc.session == nil {
		return fmt.Errorf("discord session not initialized")
	}

	// Send the message to the specified channel
	_, err := dc.session.ChannelMessageSend(chatID, message)
	if err != nil {
		return fmt.Errorf("failed to send discord message: %w", err)
	}

	log.Printf("Discord message sent successfully to channel %s", chatID)

	return nil
}

// isAllowed checks if a chat/channel is allowed
func (dc *DiscordChannel) isAllowed(chatID string) bool {
	if len(dc.allowedChats) == 0 {
		// If no allowed chats specified, allow all
		return true
	}

	for _, allowed := range dc.allowedChats {
		if allowed == chatID {
			return true
		}
	}

	return false
}