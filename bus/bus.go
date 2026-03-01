package bus

import (
	"sync"
)

// InboundMessage represents a message received by the system
type InboundMessage struct {
	Channel  string                 `json:"channel"`
	SenderID string                `json:"sender_id"`
	ChatID   string                `json:"chat_id"`
	Content  string                `json:"content"`
	SessionKey string              `json:"session_key"`
	Media    []string              `json:"media,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// OutboundMessage represents a message to be sent out
type OutboundMessage struct {
	Channel  string                 `json:"channel"`
	ChatID   string                `json:"chat_id"`
	Content  string                `json:"content"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// MessageBus handles routing of messages between components
type MessageBus struct {
	inboundQueue  chan InboundMessage
	outboundQueue chan OutboundMessage
	subscribers   map[string]chan InboundMessage
	mutex         sync.RWMutex
}

// NewMessageBus creates a new message bus
func NewMessageBus() *MessageBus {
	return &MessageBus{
		inboundQueue:  make(chan InboundMessage, 100),
		outboundQueue: make(chan OutboundMessage, 100),
		subscribers:   make(map[string]chan InboundMessage),
	}
}

// PublishInbound publishes an inbound message to the bus
func (mb *MessageBus) PublishInbound(msg InboundMessage) error {
	select {
	case mb.inboundQueue <- msg:
		return nil
	default:
		return nil // Drop message if queue is full
	}
}

// PublishOutbound publishes an outbound message to the bus
func (mb *MessageBus) PublishOutbound(msg OutboundMessage) error {
	select {
	case mb.outboundQueue <- msg:
		return nil
	default:
		return nil // Drop message if queue is full
	}
}

// ConsumeInbound waits for and returns an inbound message
func (mb *MessageBus) ConsumeInbound() (InboundMessage, error) {
	msg := <-mb.inboundQueue
	return msg, nil
}

// ConsumeOutbound waits for and returns an outbound message
func (mb *MessageBus) ConsumeOutbound() (OutboundMessage, error) {
	msg := <-mb.outboundQueue
	return msg, nil
}

// Subscribe creates a subscription to receive messages of a specific type
func (mb *MessageBus) Subscribe(subscriberID string) <-chan InboundMessage {
	mb.mutex.Lock()
	defer mb.mutex.Unlock()

	ch := make(chan InboundMessage, 10)
	mb.subscribers[subscriberID] = ch
	return ch
}

// Unsubscribe removes a subscription
func (mb *MessageBus) Unsubscribe(subscriberID string) {
	mb.mutex.Lock()
	defer mb.mutex.Unlock()

	if ch, exists := mb.subscribers[subscriberID]; exists {
		close(ch)
		delete(mb.subscribers, subscriberID)
	}
}