package test

import (
	"salesmate/channels"
	"salesmate/config"
	"testing"
)

// TestEmailChannelFunctionality performs functional tests for the email channel
func TestEmailChannelFunctionality(t *testing.T) {
	// Create a config for email with basic valid configuration
	emailConfig := &config.EmailConfig{
		Enabled:      true,
		Consent:      true, // Required for the email channel to start
		IMAPHost:     "imap.example.com",
		IMAPPort:     993,
		IMAPUsername: "test@example.com",
		IMAPPassword: "password",
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		SMTPUsername: "test@example.com",
		SMTPPassword: "password",
		FromAddress:  "test@example.com",
		AllowFrom:    []string{"recipient@example.com", "another@test.com"},
	}

	channel := channels.NewEmailChannel(emailConfig)

	if channel.Name() != "email" {
		t.Errorf("Expected name 'email', got '%s'", channel.Name())
	}

	// Test that configuration is properly set
	cfg := channel.GetConfig()
	if cfg == nil {
		t.Fatal("GetConfig returned nil")
	}

	if cfg.FromAddress != "test@example.com" {
		t.Errorf("Expected FromAddress 'test@example.com', got '%s'", cfg.FromAddress)
	}

	// Test basic functionality - start
	err := channel.Start()
	if err == nil {
		// If no error, we should be able to test more
		t.Log("Email channel started successfully (no validation errors)")

		// Test sending to allowed recipient
		err = channel.Send("recipient@example.com", "Functional test message")
		if err != nil {
			// This is expected since we're using fake credentials
			t.Logf("Expected error when sending (due to fake credentials): %v", err)
		} else {
			t.Log("Email sent successfully (unexpected with fake credentials)")
		}

		// Test IsRunning
		running := channel.IsRunning()
		t.Logf("Email channel running status: %v", running)

		// Test Stop
		stopErr := channel.Stop()
		if stopErr != nil {
			t.Logf("Error stopping channel: %v", stopErr)
		}
	} else {
		// Check if the error is due to missing consent
		if err.Error() == "email consent not granted - check config.email.consent_granted" {
			t.Error("Email channel should have started since consent was granted")
		} else {
			// Other errors are expected due to fake credentials
			t.Logf("Expected error starting email channel (due to fake credentials): %v", err)
		}
	}

	t.Logf("✓ Email channel functional test completed")
}

// TestEmailChannelInvalidConfig tests behavior with invalid configuration
func TestEmailChannelInvalidConfig(t *testing.T) {
	// Test with consent not granted
	emailConfig := &config.EmailConfig{
		Enabled: true,
		Consent: false, // No consent
	}

	channel := channels.NewEmailChannel(emailConfig)
	err := channel.Start()
	if err == nil {
		t.Error("Expected error when consent is not granted")
	} else if err.Error() != "email consent not granted - check config.email.consent_granted" {
		t.Errorf("Unexpected error message: %v", err)
	} else {
		t.Logf("✓ Email channel properly rejected due to missing consent")
	}
}
