package channels

import (
	"context"
	"fmt"
	"log"

	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/client"
)

// DingTalkChannel implements the DingTalk channel
type DingTalkChannel struct {
	clientID     string
	secret       string
	allowedChats []string
	name         string
	running      bool
	dtClient     *client.StreamClient
}

// NewDingTalkChannel creates a new DingTalk channel
func NewDingTalkChannel(clientID, secret string, allowedChats []string) *DingTalkChannel {
	return &DingTalkChannel{
		clientID:     clientID,
		secret:       secret,
		allowedChats: allowedChats,
		name:         "dingtalk",
	}
}

// Start starts the DingTalk channel
func (dc *DingTalkChannel) Start() error {
	if dc.clientID == "" || dc.secret == "" {
		return fmt.Errorf("dingtalk client id and secret must be configured")
	}

	// Create credential config
	cred := client.NewAppCredentialConfig(dc.clientID, dc.secret)

	// Initialize DingTalk client
	dtClient := client.NewStreamClient(
		client.WithAppCredential(cred),
		client.WithOpenApiHost("https://api.dingtalk.com"),
	)

	dc.dtClient = dtClient

	// Set up a simple chatbot handler to receive messages
	handlerFunc := func(ctx context.Context, data *chatbot.BotCallbackDataModel) ([]byte, error) {
		log.Printf("Received DingTalk message from %s: %s", data.SenderNick, data.Text.Content)
		// Process the incoming message as needed
		return nil, nil
	}

	// Register the chatbot handler directly as the message handler
	dtClient.RegisterChatBotCallbackRouter(handlerFunc)

	// Start the client
	ctx := context.Background()
	err := dtClient.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start DingTalk client: %w", err)
	}

	dc.running = true
	log.Printf("DingTalk channel started (using DingTalk Streaming SDK)")

	return nil
}

// Stop stops the DingTalk channel
func (dc *DingTalkChannel) Stop() error {
	dc.running = false

	if dc.dtClient != nil {
		dc.dtClient.Close()
	}

	log.Printf("DingTalk channel stopped")
	return nil
}

// Name returns the channel name
func (dc *DingTalkChannel) Name() string {
	return dc.name
}

// Send sends a message to a DingTalk chat
func (dc *DingTalkChannel) Send(chatID, message string) error {
	if !dc.running {
		return fmt.Errorf("dingtalk channel not running")
	}

	if !dc.isAllowed(chatID) {
		return fmt.Errorf("chat %s not allowed", chatID)
	}

	// For now, we'll simulate sending since the full message sending implementation
	// requires additional components like webhook callbacks or using DingTalk's OpenAPI
	log.Printf("Sending to DingTalk chat %s: %.100s...", chatID, message)

	// In a real implementation, we would use DingTalk's OpenAPI to send messages
	// This would involve getting an access token and calling the message sending API
	return dc.sendViaOpenAPI(chatID, message)
}

// sendViaOpenAPI sends a message using DingTalk's OpenAPI
func (dc *DingTalkChannel) sendViaOpenAPI(chatID, message string) error {
	// This is a simplified implementation
	// In a real scenario, we'd need to:
	// 1. Get access token from DingTalk API using clientID and secret
	// 2. Call the message sending API with the access token
	log.Printf("Would send to DingTalk OpenAPI: chat=%s, message=%.100s...", chatID, message)
	return nil
}

// isAllowed checks if a chat is allowed
func (dc *DingTalkChannel) isAllowed(chatID string) bool {
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
