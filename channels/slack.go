package channels

import (
	"fmt"
	"log"

	"github.com/slack-go/slack"
)

// SlackChannel implements the Slack channel
type SlackChannel struct {
	botToken     string
	appToken     string
	allowedChats []string
	name         string
	running      bool
	client       *slack.Client
}

// NewSlackChannel creates a new Slack channel
func NewSlackChannel(botToken, appToken string, allowedChats []string) *SlackChannel {
	return &SlackChannel{
		botToken:     botToken,
		appToken:     appToken,
		allowedChats: allowedChats,
		name:         "slack",
		running:      false,
	}
}

// Start starts the Slack channel
func (sc *SlackChannel) Start() error {
	if sc.botToken == "" {
		return fmt.Errorf("slack bot token must be configured")
	}

	client := slack.New(sc.botToken)
	sc.client = client
	sc.running = true
	log.Printf("Slack channel started")

	return nil
}

// Stop stops the Slack channel
func (sc *SlackChannel) Stop() error {
	sc.running = false
	log.Printf("Slack channel stopped")
	return nil
}

// Name returns the channel name
func (sc *SlackChannel) Name() string {
	return sc.name
}

// Send sends a message to a Slack channel
func (sc *SlackChannel) Send(chatID, message string) error {
	if !sc.running {
		return fmt.Errorf("slack channel not running")
	}

	if !sc.isAllowed(chatID) {
		return fmt.Errorf("channel %s not allowed", chatID)
	}

	if sc.client == nil {
		return fmt.Errorf("slack client not initialized")
	}

	// Send the message to the specified channel
	_, _, err := sc.client.PostMessage(chatID, slack.MsgOptionText(message, false))
	if err != nil {
		return fmt.Errorf("failed to send slack message: %w", err)
	}

	log.Printf("Slack message sent successfully to channel %s", chatID)

	return nil
}

// isAllowed checks if a chat/channel is allowed
func (sc *SlackChannel) isAllowed(chatID string) bool {
	if len(sc.allowedChats) == 0 {
		// If no allowed chats specified, allow all
		return true
	}

	for _, allowed := range sc.allowedChats {
		if allowed == chatID {
			return true
		}
	}

	return false
}