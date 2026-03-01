package channels

import (
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// EmailReceiver handles incoming emails via IMAP
type EmailReceiver struct {
	channel    *EmailChannel
	imapClient *client.Client
	stopChan   chan struct{}
}

// NewEmailReceiver creates a new email receiver
func NewEmailReceiver(channel *EmailChannel) *EmailReceiver {
	return &EmailReceiver{
		channel:  channel,
		stopChan: make(chan struct{}),
	}
}

// Start begins polling for incoming emails
func (er *EmailReceiver) Start() error {
	cfg := er.channel.GetConfig()

	if cfg.IMAPHost == "" || cfg.IMAPPort == 0 {
		log.Println("IMAP settings not configured, skipping incoming email monitoring")
		return nil // Not an error, just disabled
	}

	// Connect to the IMAP server
	tlsConfig := &tls.Config{ServerName: cfg.IMAPHost}

	var err error
	er.imapClient, err = client.DialTLS(fmt.Sprintf("%s:%d", cfg.IMAPHost, cfg.IMAPPort), tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to IMAP server: %v", err)
	}

	// Login
	if err := er.imapClient.Login(cfg.IMAPUsername, cfg.IMAPPassword); err != nil {
		return fmt.Errorf("failed to login to IMAP server: %v", err)
	}

	log.Printf("Successfully connected to IMAP server: %s", cfg.IMAPHost)

	// Start monitoring for new emails in a separate goroutine
	go er.monitorMailbox()

	return nil
}

// Stop stops the email receiver
func (er *EmailReceiver) Stop() error {
	close(er.stopChan)

	if er.imapClient != nil {
		return er.imapClient.Logout()
	}

	return nil
}

// monitorMailbox continuously monitors the mailbox for new emails
func (er *EmailReceiver) monitorMailbox() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-er.stopChan:
			log.Println("Email receiver stopped")
			return
		case <-ticker.C:
			if err := er.checkNewEmails(); err != nil {
				log.Printf("Error checking new emails: %v", err)
			}
		}
	}
}

// checkNewEmails checks for new unread emails and processes them
func (er *EmailReceiver) checkNewEmails() error {
	// Select the INBOX
	mbox, err := er.imapClient.Select("INBOX", false)
	if err != nil {
		return fmt.Errorf("failed to select INBOX: %v", err)
	}

	if mbox.Messages == 0 {
		return nil // No messages in mailbox
	}

	// Search for unread messages
	criteria := &imap.SearchCriteria{}
	criteria.WithoutFlags = []string{imap.SeenFlag}
	uids, err := er.imapClient.Search(criteria)
	if err != nil {
		return fmt.Errorf("failed to search for unread messages: %v", err)
	}

	if len(uids) == 0 {
		return nil // No unread messages
	}

	// For each unread message, we'll just mark it as read without processing the content
	// In a real implementation, you would fetch the full message content
	for _, uid := range uids {
		seqSet := new(imap.SeqSet)
		seqSet.AddNum(uid)

		// In a real implementation, you would fetch the full message body
		// For now we just log the UID
		log.Printf("Found unread message with UID: %d", uid)

		// Check sender and mark as read
		seenSeqSet := new(imap.SeqSet)
		seenSeqSet.AddNum(uid)

		err = er.imapClient.UidStore(seenSeqSet, imap.AddFlags, []interface{}{imap.SeenFlag}, nil)
		if err != nil {
			log.Printf("Error marking message as read: %v", err)
		}
	}

	return nil
}

// IsConnected returns whether the IMAP client is connected
func (er *EmailReceiver) IsConnected() bool {
	if er.imapClient == nil {
		return false
	}

	// We can test the connection by trying a simple command
	_, err := er.imapClient.Capability()
	return err == nil
}