package channels

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/mail"
	"sync"

	"salesmate/config"

	"gopkg.in/gomail.v2"
)

// EmailChannel implements the Email channel
type EmailChannel struct {
	enabled       bool
	config        *config.EmailConfig
	allowedEmails []string
	name          string
	running       bool
	mutex         sync.Mutex
}

// NewEmailChannel creates a new Email channel from config
func NewEmailChannel(cfg *config.EmailConfig) *EmailChannel {
	return &EmailChannel{
		enabled:       cfg.Enabled,
		config:        cfg,
		allowedEmails: cfg.AllowFrom,
		name:          "email",
		running:       false,
	}
}

// Start starts the Email channel
func (ec *EmailChannel) Start() error {
	if !ec.enabled {
		return fmt.Errorf("email channel not enabled")
	}

	if !ec.config.Consent {
		return fmt.Errorf("email consent not granted - check config.email.consent_granted")
	}

	if ec.config.SMTPUsername == "" || ec.config.SMTPPassword == "" {
		return fmt.Errorf("email SMTP credentials not configured")
	}

	// Basic validation of email addresses
	if ec.config.FromAddress != "" {
		_, err := mail.ParseAddress(ec.config.FromAddress)
		if err != nil {
			return fmt.Errorf("invalid from email address: %v", err)
		}
	}

	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	ec.running = true
	log.Printf("Email channel started")

	// In a full implementation, we would start an IMAP listener to poll for emails
	// This would involve connecting to the IMAP server and monitoring for new messages
	// For now, we only support outbound email via SMTP

	return nil
}

// Stop stops the Email channel
func (ec *EmailChannel) Stop() error {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	ec.running = false
	log.Printf("Email channel stopped")
	return nil
}

// Name returns the channel name
func (ec *EmailChannel) Name() string {
	return ec.name
}

// Send sends a message to an email address
func (ec *EmailChannel) Send(toAddr, message string) error {
	ec.mutex.Lock()
	running := ec.running
	ec.mutex.Unlock()

	if !running {
		return fmt.Errorf("email channel not running")
	}

	// Parse the recipient email address to ensure it's valid
	_, err := mail.ParseAddress(toAddr)
	if err != nil {
		return fmt.Errorf("invalid recipient email address: %v", err)
	}

	// Check if the recipient is allowed to receive messages (using AllowFrom as allowed recipients)
	if !ec.isAllowed(toAddr) {
		return fmt.Errorf("email address %s not allowed to receive messages", toAddr)
	}

	// Log the email sending attempt
	log.Printf("Sending email to %s: %s", toAddr, message)

	// Send via SMTP
	return ec.sendViaSMTP(toAddr, message)
}

// sendViaSMTP sends an email via SMTP
func (ec *EmailChannel) sendViaSMTP(toAddr, message string) error {
	// Create a new message
	m := gomail.NewMessage()

	// Set the sender
	fromAddr := ec.config.FromAddress
	if fromAddr == "" {
		// If no explicit from address is set, use the username as a fallback
		fromAddr = ec.config.SMTPUsername
	}

	m.SetHeader("From", fromAddr)
	m.SetHeader("To", toAddr)
	m.SetHeader("Subject", "Nanobot Message")
	m.SetBody("text/plain", message)

	// Create the dialer for SMTP connection
	port := ec.config.SMTPPort
	if port == 0 {
		port = 587 // Default SMTP port
	}

	dialer := gomail.NewDialer(ec.config.SMTPHost, port, ec.config.SMTPUsername, ec.config.SMTPPassword)

	// Configure TLS based on the port (common convention)
	if port == 465 {
		dialer.TLSConfig = &tls.Config{InsecureSkipVerify: false}
	}

	// Attempt to send the email
	err := dialer.DialAndSend(m)
	if err != nil {
		log.Printf("Failed to send email via SMTP: %v", err)
		return fmt.Errorf("failed to send email: %v", err)
	}

	log.Printf("Successfully sent email to %s", toAddr)
	return nil
}

// SendHTML sends an HTML formatted message to an email address
func (ec *EmailChannel) SendHTML(toAddr, subject, htmlContent string) error {
	ec.mutex.Lock()
	running := ec.running
	ec.mutex.Unlock()

	if !running {
		return fmt.Errorf("email channel not running")
	}

	// Parse the recipient email address to ensure it's valid
	_, err := mail.ParseAddress(toAddr)
	if err != nil {
		return fmt.Errorf("invalid recipient email address: %v", err)
	}

	// Check if the recipient is allowed
	if !ec.isAllowed(toAddr) {
		return fmt.Errorf("email address %s not allowed to receive messages", toAddr)
	}

	// Log the email sending attempt
	log.Printf("Sending HTML email to %s with subject: %s", toAddr, subject)

	// Create a new message
	m := gomail.NewMessage()

	// Set the sender
	fromAddr := ec.config.FromAddress
	if fromAddr == "" {
		// If no explicit from address is set, use the username as a fallback
		fromAddr = ec.config.SMTPUsername
	}

	m.SetHeader("From", fromAddr)
	m.SetHeader("To", toAddr)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", htmlContent)

	// Create the dialer for SMTP connection
	port := ec.config.SMTPPort
	if port == 0 {
		port = 587 // Default SMTP port
	}

	dialer := gomail.NewDialer(ec.config.SMTPHost, port, ec.config.SMTPUsername, ec.config.SMTPPassword)

	// Configure TLS based on the port (common convention)
	if port == 465 {
		dialer.TLSConfig = &tls.Config{InsecureSkipVerify: false}
	}

	// Attempt to send the email
	err = dialer.DialAndSend(m)
	if err != nil {
		log.Printf("Failed to send HTML email via SMTP: %v", err)
		return fmt.Errorf("failed to send HTML email: %v", err)
	}

	log.Printf("Successfully sent HTML email to %s", toAddr)
	return nil
}

// IsRunning returns whether the channel is currently running
func (ec *EmailChannel) IsRunning() bool {
	ec.mutex.Lock()
	defer ec.mutex.Unlock()
	return ec.running
}

// GetConfig returns the email channel configuration
func (ec *EmailChannel) GetConfig() *config.EmailConfig {
	return ec.config
}

// isAllowed checks if an email address is allowed
func (ec *EmailChannel) isAllowed(emailAddr string) bool {
	if len(ec.allowedEmails) == 0 {
		// If no allowed emails specified, allow all
		return true
	}

	for _, allowed := range ec.allowedEmails {
		if allowed == emailAddr {
			return true
		}
	}

	return false
}
