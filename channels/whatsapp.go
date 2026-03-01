package channels

import (
	"fmt"
	"log"
)

// WhatsAppConfig contains WhatsApp channel configuration
type WhatsAppConfig struct {
	Enabled   bool     `mapstructure:"enabled"`
	AllowFrom []string `mapstructure:"allow_from"`
}

// WhatsAppChannel implements the WhatsApp channel
type WhatsAppChannel struct {
	Config       *WhatsAppConfig
	name         string
	running      bool
}

// NewWhatsAppChannel creates a new WhatsApp channel
func NewWhatsAppChannel(config *WhatsAppConfig) *WhatsAppChannel {
	return &WhatsAppChannel{
		Config:  config,
		name:    "whatsapp",
		running: false,
	}
}

// Start starts the WhatsApp channel
func (wc *WhatsAppChannel) Start() error {
	if !wc.Config.Enabled {
		return fmt.Errorf("whatsapp channel not enabled in config")
	}

	wc.running = true
	log.Printf("WhatsApp channel started")
	// In a real implementation, this would connect to WhatsApp Business API or similar
	return nil
}

// Stop stops the WhatsApp channel
func (wc *WhatsAppChannel) Stop() error {
	wc.running = false
	log.Printf("WhatsApp channel stopped")
	return nil
}

// Name returns the channel name
func (wc *WhatsAppChannel) Name() string {
	return wc.name
}

// Send sends a message to a WhatsApp chat
func (wc *WhatsAppChannel) Send(chatID, message string) error {
	if !wc.running {
		return fmt.Errorf("whatsapp channel not running")
	}

	if !wc.isAllowed(chatID) {
		return fmt.Errorf("chat %s not allowed", chatID)
	}

	// Log the message as sent (would integrate with WhatsApp Business API in real implementation)
	log.Printf("WhatsApp message sent to %s: %s", chatID, message)

	return nil
}

// isAllowed checks if a chat is allowed
func (wc *WhatsAppChannel) isAllowed(chatID string) bool {
	if len(wc.Config.AllowFrom) == 0 {
		// If no allowed chats specified, allow all
		return true
	}

	for _, allowed := range wc.Config.AllowFrom {
		if allowed == chatID {
			return true
		}
	}

	return false
}