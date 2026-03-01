package channels

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// TelegramChannel implements the Telegram channel
type TelegramChannel struct {
	bot          *tgbotapi.BotAPI
	token        string
	allowedChats []string
	name         string
	running      bool
	httpClient   *http.Client
}

// NewTelegramChannel creates a new Telegram channel
func NewTelegramChannel(token string, allowedChats []string) *TelegramChannel {
	return &TelegramChannel{
		token:        token,
		allowedChats: allowedChats,
		name:         "telegram",
		httpClient:   &http.Client{},
	}
}

// Start starts the Telegram channel
func (tc *TelegramChannel) Start() error {
	if tc.token == "" {
		return fmt.Errorf("telegram token not configured")
	}

	bot, err := tgbotapi.NewBotAPI(tc.token)
	if err != nil {
		return fmt.Errorf("failed to create telegram bot: %w", err)
	}

	tc.bot = bot
	tc.running = true

	log.Printf("Authorized on account %s", bot.Self.UserName)

	// Set up updates configuration
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	// Handle incoming messages in a goroutine
	go func() {
		for update := range updates {
			if update.Message != nil && update.Message.IsCommand() {
				tc.handleCommand(update.Message)
			} else if update.Message != nil {
				tc.handleMessage(update.Message)
			}
		}
	}()

	log.Printf("Telegram channel started")
	return nil
}

// handleCommand processes commands from Telegram
func (tc *TelegramChannel) handleCommand(message *tgbotapi.Message) {
	log.Printf("[%s] %s", message.From.UserName, message.Command())

	switch message.Command() {
	case "help":
		msg := tgbotapi.NewMessage(message.Chat.ID, "This is Nanobot Telegram integration.")
		tc.bot.Send(msg)
	case "start":
		msg := tgbotapi.NewMessage(message.Chat.ID, "Welcome to Nanobot!")
		tc.bot.Send(msg)
	}
}

// handleMessage processes incoming messages from Telegram
func (tc *TelegramChannel) handleMessage(message *tgbotapi.Message) {
	senderID := strconv.FormatInt(message.From.ID, 10)
	content := message.Text

	log.Printf("Received message from %s: %s", senderID, content)

	// Here you would normally process the message with the bot
	// For now, just log it
}

// Stop stops the Telegram channel
func (tc *TelegramChannel) Stop() error {
	tc.running = false
	log.Printf("Telegram channel stopped")
	return nil
}

// Name returns the channel name
func (tc *TelegramChannel) Name() string {
	return tc.name
}

// Send sends a message to a Telegram chat
func (tc *TelegramChannel) Send(chatID, message string) error {
	if !tc.running {
		return fmt.Errorf("telegram channel not running")
	}

	if !tc.isChatAllowed(chatID) {
		return fmt.Errorf("chat %s not allowed", chatID)
	}

	// Convert chatID string to int64
	chatIDInt, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID: %w", err)
	}

	// Telegram has a message length limit of 4096 characters
	const maxLength = 4000
	if len(message) > maxLength {
		// Split the message into chunks if needed
		for len(message) > 0 {
			chunkSize := maxLength
			if len(message) < maxLength {
				chunkSize = len(message)
			}

			chunk := message[:chunkSize]
			msg := tgbotapi.NewMessage(chatIDInt, chunk)
			_, err := tc.bot.Send(msg)
			if err != nil {
				return fmt.Errorf("failed to send telegram message: %w", err)
			}

			message = message[chunkSize:]
		}
	} else {
		msg := tgbotapi.NewMessage(chatIDInt, message)
		_, err := tc.bot.Send(msg)
		if err != nil {
			return fmt.Errorf("failed to send telegram message: %w", err)
		}
	}

	return nil
}

// isChatAllowed checks if a chat is allowed
func (tc *TelegramChannel) isChatAllowed(chatID string) bool {
	if len(tc.allowedChats) == 0 {
		// If no allowed chats specified, allow all
		return true
	}

	for _, allowed := range tc.allowedChats {
		if allowed == chatID {
			return true
		}
	}

	return false
}