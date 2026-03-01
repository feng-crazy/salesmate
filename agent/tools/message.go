package tools

import (
	"fmt"

	"salesmate/bus"
)

// MessageTool implements a tool to send messages to various channels
type MessageTool struct {
	sendCallback func(msg bus.OutboundMessage) error
	channel      string
	chatID       string
	messageID    string
}

// NewMessageTool creates a new message tool
func NewMessageTool(sendCallback func(bus.OutboundMessage) error) *MessageTool {
	return &MessageTool{
		sendCallback: sendCallback,
	}
}

// Name returns the name of the tool
func (t *MessageTool) Name() string {
	return "send_message"
}

// Description returns the description of the tool
func (t *MessageTool) Description() string {
	return "Send a message to a specific channel/chat"
}

// Call executes the tool with the given arguments
func (t *MessageTool) Call(args map[string]interface{}) (string, error) {
	content, ok := args["content"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'content' argument")
	}

	channel, _ := args["channel"].(string) // optional, use default if not provided
	chatID, _ := args["chat_id"].(string)  // optional, use default if not provided

	if channel == "" {
		channel = t.channel
	}
	if chatID == "" {
		chatID = t.chatID
	}

	if channel == "" || chatID == "" {
		return "", fmt.Errorf("either specify 'channel' and 'chat_id' in arguments or set context first")
	}

	msg := bus.OutboundMessage{
		Channel: channel,
		ChatID:  chatID,
		Content: content,
	}

	if t.sendCallback != nil {
		if err := t.sendCallback(msg); err != nil {
			return "", fmt.Errorf("failed to send message: %w", err)
		}
		return fmt.Sprintf("Message sent to %s:%s", channel, chatID), nil
	}

	return fmt.Sprintf("Message ready to send to %s:%s: %s", channel, chatID, content), nil
}

// SetContext updates the context for routing information
func (t *MessageTool) SetContext(channel, chatID, messageID string) {
	t.channel = channel
	t.chatID = chatID
	t.messageID = messageID
}
