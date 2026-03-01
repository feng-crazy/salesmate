package channels_test

import (
	"salesmate/channels"
	"salesmate/config"
	"testing"
)

func TestChannelManager(t *testing.T) {
	// Create a minimal config for testing
	cfg := &config.Config{
		Channels: config.ChannelsConfig{
			Telegram: config.TelegramConfig{
				Enabled: true,
				Token:   "test-token",
			},
			WhatsApp: config.WhatsAppConfig{
				Enabled: true,
			},
		},
	}

	manager := channels.NewManager(cfg)

	if manager == nil {
		t.Fatal("NewManager returned nil")
	}

	// Check that it registered the enabled channels
	enabledChannels := manager.GetEnabledChannels()
	expectedCount := 2 // Telegram and WhatsApp are enabled
	if len(enabledChannels) != expectedCount {
		t.Errorf("Expected %d enabled channels, got %d: %v", expectedCount, len(enabledChannels), enabledChannels)
	}

	// Test StartAll
	err := manager.StartAll()
	if err != nil {
		t.Logf("StartAll returned error (expected due to fake tokens): %v", err)
		// Don't fail the test as fake tokens are expected to cause errors
	}

	// Test registering a mock channel
	mockChannel := &mockChannel{name: "test-channel"}
	manager.Register(mockChannel)

	// Verify it was registered
	channel, exists := manager.Get("test-channel")
	if !exists {
		t.Error("Registered channel was not found")
	}
	if channel.Name() != "test-channel" {
		t.Errorf("Expected channel name 'test-channel', got '%s'", channel.Name())
	}

	// Test sending to channel
	err = manager.SendToChannel("test-channel", "test-chat", "Hello World")
	if err != nil {
		t.Errorf("SendToChannel failed: %v", err)
	}

	// Test sending to non-existent channel
	err = manager.SendToChannel("non-existent", "test-chat", "Hello World")
	if err == nil {
		t.Error("SendToChannel to non-existent channel should fail")
	}

	// Test StopAll
	err = manager.StopAll()
	if err != nil {
		t.Logf("StopAll returned error (may be expected): %v", err)
		// Don't fail the test as some errors are expected during shutdown
	}
}

// Mock channel for testing
type mockChannel struct {
	name    string
	started bool
	stopped bool
}

func (mc *mockChannel) Start() error {
	mc.started = true
	return nil
}

func (mc *mockChannel) Stop() error {
	mc.stopped = true
	return nil
}

func (mc *mockChannel) Name() string {
	return mc.name
}

func (mc *mockChannel) Send(chatID, message string) error {
	// Simulate sending a message
	return nil
}
