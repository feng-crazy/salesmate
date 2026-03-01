package channels

import (
	"fmt"
	"log"
)

// QQChannel implements the QQ channel
type QQChannel struct {
	appID         string
	secret        string
	allowedUsers  []string
	name          string
	running       bool
}

// NewQQChannel creates a new QQ channel
func NewQQChannel(appID, secret string, allowedUsers []string) *QQChannel {
	return &QQChannel{
		appID:        appID,
		secret:       secret,
		allowedUsers: allowedUsers,
		name:         "qq",
	}
}

// Start starts the QQ channel
func (qq *QQChannel) Start() error {
	if qq.appID == "" || qq.secret == "" {
		return fmt.Errorf("qq app id and secret not configured")
	}

	qq.running = true
	log.Printf("QQ channel started")

	// In a real implementation, we would authenticate with QQ and establish
	// a connection to the QQ bot API

	return nil
}

// Stop stops the QQ channel
func (qq *QQChannel) Stop() error {
	qq.running = false
	log.Printf("QQ channel stopped")
	return nil
}

// Name returns the channel name
func (qq *QQChannel) Name() string {
	return qq.name
}

// Send sends a message to a QQ user
func (qq *QQChannel) Send(userID, message string) error {
	// In a real implementation, we would make an HTTP request to the QQ Bot API
	// For now, we'll just print the message

	// Validate that the recipient is allowed
	if !qq.isAllowed(userID) {
		return fmt.Errorf("user %s not allowed", userID)
	}

	log.Printf("Sending to QQ user %s: %s", userID, message)

	// Actually send via HTTP request to QQ Bot API
	return qq.sendToQQAPI(userID, message)
}

// sendToQQAPI sends a message to QQ via HTTP API
func (qq *QQChannel) sendToQQAPI(userID, message string) error {
	// In a real implementation, we would make HTTP requests to QQ Bot API:
	// POST https://api.sgroup.qq.com/v2/users/{userID}/messages
	// With proper authentication headers and message body

	// For this placeholder implementation, we'll just log the attempt
	log.Printf("Would send to QQ API: appID=%s, user=%s, message=%.100s...", qq.appID, userID, message)
	return nil
}

// isAllowed checks if a user is allowed
func (qq *QQChannel) isAllowed(userID string) bool {
	if len(qq.allowedUsers) == 0 {
		// If no allowed users specified, allow all
		return true
	}

	for _, allowed := range qq.allowedUsers {
		if allowed == userID {
			return true
		}
	}

	return false
}