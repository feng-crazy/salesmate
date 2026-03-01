package channels_test

import (
	"salesmate/channels"
	"testing"
)

func TestWhatsAppChannel(t *testing.T) {
	config := &channels.WhatsAppConfig{
		Enabled:   true,
		AllowFrom: []string{"test-user"},
	}

	channel := channels.NewWhatsAppChannel(config)

	if channel.Name() != "whatsapp" {
		t.Errorf("Expected name 'whatsapp', got '%s'", channel.Name())
	}

	// Test start
	err := channel.Start()
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}

	// Test send to allowed user
	err = channel.Send("test-user", "Hello World")
	if err != nil {
		t.Errorf("Send to allowed user failed: %v", err)
	}

	// Test send to disallowed user (with empty AllowFrom list it should pass)
	channel.Config.AllowFrom = []string{}
	err = channel.Send("other-user", "Hello World")
	if err != nil {
		t.Errorf("Send to user when AllowFrom is empty failed: %v", err)
	}

	// Test send to disallowed user (when AllowFrom is not empty)
	channel.Config.AllowFrom = []string{"allowed-user"}
	err = channel.Send("disallowed-user", "Hello World")
	if err == nil {
		t.Error("Send to disallowed user should have failed")
	}

	// Test stop
	err = channel.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}

func TestWhatsAppChannelDisabled(t *testing.T) {
	config := &channels.WhatsAppConfig{
		Enabled:   false,
		AllowFrom: []string{"test-user"},
	}

	channel := channels.NewWhatsAppChannel(config)

	// Should fail when disabled
	err := channel.Start()
	if err == nil {
		t.Error("Start should fail when channel is disabled")
	}
}
