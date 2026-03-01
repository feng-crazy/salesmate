package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

// FeishuChannel implements the Feishu channel
type FeishuChannel struct {
	appID        string
	appSecret    string
	encryptKey   string
	verification string
	allowedChats []string
	name         string
	running      bool
	larkClient   *lark.Client
}

// NewFeishuChannel creates a new Feishu channel
func NewFeishuChannel(appID, appSecret, encryptKey, verification string, allowedChats []string) *FeishuChannel {
	return &FeishuChannel{
		appID:        appID,
		appSecret:    appSecret,
		encryptKey:   encryptKey,
		verification: verification,
		allowedChats: allowedChats,
		name:         "feishu",
	}
}

// Start starts the Feishu channel
func (fc *FeishuChannel) Start() error {
	if fc.appID == "" || fc.appSecret == "" {
		return fmt.Errorf("feishu app id and secret must be configured")
	}

	// Initialize Lark client
	larkClient := lark.NewClient(
		fc.appID,
		fc.appSecret,
	)

	fc.larkClient = larkClient
	fc.running = true
	log.Printf("Feishu channel started (using Lark OAPI SDK)")

	// In a real implementation, we would set up event handlers and webhook receivers
	// For now, this is a functional implementation using Feishu API

	return nil
}

// Stop stops the Feishu channel
func (fc *FeishuChannel) Stop() error {
	fc.running = false
	log.Printf("Feishu channel stopped")
	return nil
}

// Name returns the channel name
func (fc *FeishuChannel) Name() string {
	return fc.name
}

// Send sends a message to a Feishu chat
func (fc *FeishuChannel) Send(chatID, message string) error {
	if !fc.running {
		return fmt.Errorf("feishu channel not running")
	}

	if !fc.isAllowed(chatID) {
		return fmt.Errorf("chat %s not allowed", chatID)
	}

	if fc.larkClient == nil {
		return fmt.Errorf("lark client not initialized")
	}

	// For now, simulate sending the message until we fix the SDK API call
	// Create the content JSON
	content, _ := json.Marshal(map[string]interface{}{
		"text": message,
	})

	// Make the API call - simplified approach
	resp, err := fc.larkClient.Im.Message.Create(context.Background(),
		larkim.NewCreateMessageReqBuilder().
			Body(larkim.NewCreateMessageReqBodyBuilder().
				ReceiveId(chatID).
				MsgType(larkim.MsgTypeText).
				Content(string(content)).
				Build()).
			Build(),
	)
	if err != nil {
		return fmt.Errorf("failed to send feishu message: %w", err)
	}

	if !resp.Success() {
		return fmt.Errorf("feishu API returned error: %s", resp.CodeError.Msg)
	}

	log.Printf("Feishu message sent successfully to %s, message id: %s", chatID, *resp.Data.MessageId)

	return nil
}

// isAllowed checks if a chat is allowed
func (fc *FeishuChannel) isAllowed(chatID string) bool {
	if len(fc.allowedChats) == 0 {
		// If no allowed chats specified, allow all
		return true
	}

	for _, allowed := range fc.allowedChats {
		if allowed == chatID {
			return true
		}
	}

	return false
}
