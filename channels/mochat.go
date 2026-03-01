package channels

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// MochatChannel implements the Mochat channel
type MochatChannel struct {
	baseURL      string
	clawToken    string
	allowedChats []string
	name         string
	running      bool
	httpClient   *http.Client
}

// NewMochatChannel creates a new Mochat channel
func NewMochatChannel(baseURL, clawToken string, allowedChats []string) *MochatChannel {
	return &MochatChannel{
		baseURL:      baseURL,
		clawToken:    clawToken,
		allowedChats: allowedChats,
		name:         "mochat",
		running:      false,
		httpClient:   &http.Client{},
	}
}

// Start starts the Mochat channel
func (mc *MochatChannel) Start() error {
	if mc.baseURL == "" || mc.clawToken == "" {
		return fmt.Errorf("mochat base URL and claw token must be configured")
	}

	mc.running = true
	log.Printf("Mochat channel started (endpoint: %s)", mc.baseURL)

	return nil
}

// Stop stops the Mochat channel
func (mc *MochatChannel) Stop() error {
	mc.running = false
	log.Printf("Mochat channel stopped")
	return nil
}

// Name returns the channel name
func (mc *MochatChannel) Name() string {
	return mc.name
}

// Send sends a message to a Mochat channel
func (mc *MochatChannel) Send(chatID, message string) error {
	if !mc.running {
		return fmt.Errorf("mochat channel not running")
	}

	if !mc.isAllowed(chatID) {
		return fmt.Errorf("chat %s not allowed", chatID)
	}

	if mc.baseURL == "" || mc.clawToken == "" {
		return fmt.Errorf("mochat base URL and claw token not configured")
	}

	// Prepare the message payload
	payload := map[string]interface{}{
		"to_user": chatID,
		"message": message,
		"token":   mc.clawToken,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal message payload: %w", err)
	}

	// Create the HTTP request
	url := mc.baseURL + "/send_message"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+mc.clawToken)

	// Send the request
	resp, err := mc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send mochat message: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read mochat response: %w", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("Mochat message sent successfully to %s, response: %s", chatID, string(respBody))
		return nil
	} else {
		return fmt.Errorf("mochat API returned error (status: %d): %s", resp.StatusCode, string(respBody))
	}
}

// isAllowed checks if a chat is allowed
func (mc *MochatChannel) isAllowed(chatID string) bool {
	if len(mc.allowedChats) == 0 {
		// If no allowed chats specified, allow all
		return true
	}

	for _, allowed := range mc.allowedChats {
		if allowed == chatID {
			return true
		}
	}

	return false
}