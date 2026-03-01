package channels_test

import (
	"salesmate/channels"
	"testing"
)

func TestDiscordChannel(t *testing.T) {
	channel := channels.NewDiscordChannel("fake-token", []string{"allowed-channel"})

	if channel.Name() != "discord" {
		t.Errorf("Expected name 'discord', got '%s'", channel.Name())
	}

	// Test start with no token
	channelEmpty := channels.NewDiscordChannel("", []string{})
	err := channelEmpty.Start()
	if err == nil {
		t.Error("Start should fail with empty token")
	}

	// Test the basic functionality without expecting immediate validation of fake token
}
