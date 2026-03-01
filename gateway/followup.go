package gateway

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"salesmate/sales_agent"
	"salesmate/security"
)

// FollowUpEngine handles proactive customer follow-ups
type FollowUpEngine struct {
	salesLoop    *sales_agent.SalesLoop
	safetySystem *security.SafetySystem
	mu           sync.RWMutex
	running      bool
	sessions     map[string]*FollowUpSession
	interval     time.Duration
	maxFollowUps int
	notifyCh     chan FollowUpEvent
}

// FollowUpSession tracks follow-up state for a customer
type FollowUpSession struct {
	SessionID         string         `json:"sessionId"`
	LastContact       time.Time      `json:"lastContact"`
	LastStage         string         `json:"lastStage"`
	FollowUpCount     int            `json:"followUpCount"`
	NextFollowUp      time.Time      `json:"nextFollowUp"`
	LastTopic         string         `json:"lastTopic"`
	EngagementScore   float64        `json:"engagementScore"`
	Qualified         bool           `json:"qualified"`
	Priority          FollowUpPriority `json:"priority"`
}

// FollowUpPriority indicates the urgency of follow-up
type FollowUpPriority string

const (
	PriorityHigh   FollowUpPriority = "high"
	PriorityMedium FollowUpPriority = "medium"
	PriorityLow    FollowUpPriority = "low"
)

// FollowUpEvent represents a follow-up trigger event
type FollowUpEvent struct {
	SessionID   string          `json:"sessionId"`
	Type        FollowUpType    `json:"type"`
	Priority    FollowUpPriority `json:"priority"`
	Message     string          `json:"message"`
	ScheduledAt time.Time       `json:"scheduledAt"`
}

// FollowUpType defines the reason for follow-up
type FollowUpType string

const (
	FollowUpTimer   FollowUpType = "timer"   // 24h+ since last contact
	FollowUpStage   FollowUpType = "stage"   // Stage-specific follow-up
	FollowUpIntent  FollowUpType = "intent"  // Intent-based follow-up
	FollowUpWinback FollowUpType = "winback" // Re-engage cold leads
)

// FollowUpConfig holds configuration for the follow-up engine
type FollowUpConfig struct {
	IntervalHours    int
	MaxFollowUps     int
	HighPriorityHours int
	EnableAutoSend   bool
}

// DefaultFollowUpConfig returns default configuration
func DefaultFollowUpConfig() *FollowUpConfig {
	return &FollowUpConfig{
		IntervalHours:     24,
		MaxFollowUps:      5,
		HighPriorityHours: 4,
		EnableAutoSend:    false, // Requires human approval by default
	}
}

// NewFollowUpEngine creates a new follow-up engine
func NewFollowUpEngine(salesLoop *sales_agent.SalesLoop, safetySystem *security.SafetySystem) *FollowUpEngine {
	return &FollowUpEngine{
		salesLoop:    salesLoop,
		safetySystem: safetySystem,
		sessions:     make(map[string]*FollowUpSession),
		interval:     1 * time.Hour, // Check every hour
		maxFollowUps: 5,
		notifyCh:     make(chan FollowUpEvent, 100),
	}
}

// Start begins the follow-up engine
func (f *FollowUpEngine) Start(ctx context.Context) {
	f.mu.Lock()
	f.running = true
	f.mu.Unlock()

	ticker := time.NewTicker(f.interval)
	defer ticker.Stop()

	log.Println("Follow-up engine started")

	for {
		select {
		case <-ctx.Done():
			f.Stop()
			return
		case <-ticker.C:
			f.checkFollowUps(ctx)
		case event := <-f.notifyCh:
			f.processFollowUpEvent(ctx, event)
		}
	}
}

// Stop stops the follow-up engine
func (f *FollowUpEngine) Stop() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.running = false
	log.Println("Follow-up engine stopped")
}

// UpdateSession updates follow-up state when a customer interacts
func (f *FollowUpEngine) UpdateSession(sessionID string, stage sales_agent.SalesStage, topic string, engagement float64) {
	f.mu.Lock()
	defer f.mu.Unlock()

	session, exists := f.sessions[sessionID]
	if !exists {
		session = &FollowUpSession{
			SessionID: sessionID,
		}
		f.sessions[sessionID] = session
	}

	session.LastContact = time.Now()
	session.LastStage = string(stage)
	session.LastTopic = topic
	session.EngagementScore = engagement

	// Calculate next follow-up time based on stage and engagement
	session.NextFollowUp = f.calculateNextFollowUp(session)
	session.FollowUpCount = 0 // Reset on interaction

	// Determine priority
	session.Priority = f.calculatePriority(session)

	// Check if qualified
	session.Qualified = engagement >= 0.6 && stage != sales_agent.NewContact
}

// ScheduleFollowUp schedules a follow-up for a specific time
func (f *FollowUpEngine) ScheduleFollowUp(sessionID string, followUpTime time.Time, reason FollowUpType) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	session, exists := f.sessions[sessionID]
	if !exists {
		session = &FollowUpSession{
			SessionID: sessionID,
		}
		f.sessions[sessionID] = session
	}

	session.NextFollowUp = followUpTime

	// Queue the event
	select {
	case f.notifyCh <- FollowUpEvent{
		SessionID:   sessionID,
		Type:        reason,
		Priority:    session.Priority,
		ScheduledAt: followUpTime,
	}:
	default:
		log.Printf("Warning: follow-up queue full, dropping event for %s", sessionID)
	}

	return nil
}

// GetPendingFollowUps returns all pending follow-ups
func (f *FollowUpEngine) GetPendingFollowUps() []*FollowUpSession {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var pending []*FollowUpSession
	now := time.Now()

	for _, session := range f.sessions {
		if session.NextFollowUp.Before(now) && session.FollowUpCount < f.maxFollowUps {
			pending = append(pending, session)
		}
	}

	return pending
}

// checkFollowUps periodically checks for follow-up opportunities
func (f *FollowUpEngine) checkFollowUps(ctx context.Context) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	now := time.Now()

	for sessionID, session := range f.sessions {
		// Check if follow-up is due
		if session.NextFollowUp.After(now) {
			continue
		}

		// Check if max follow-ups reached
		if session.FollowUpCount >= f.maxFollowUps {
			continue
		}

		// Determine follow-up type
		followUpType := f.determineFollowUpType(session)

		// Create follow-up event
		event := FollowUpEvent{
			SessionID:   sessionID,
			Type:        followUpType,
			Priority:    session.Priority,
			ScheduledAt: now,
		}

		// Process the event
		go f.processFollowUpEvent(ctx, event)
	}
}

// processFollowUpEvent handles a follow-up event
func (f *FollowUpEngine) processFollowUpEvent(ctx context.Context, event FollowUpEvent) {
	f.mu.Lock()
	session, exists := f.sessions[event.SessionID]
	if !exists {
		f.mu.Unlock()
		return
	}

	// Generate follow-up message
	message := f.generateFollowUpMessage(session, event.Type)

	// Update session state
	session.FollowUpCount++
	session.NextFollowUp = time.Now().Add(24 * time.Hour)

	// Log the follow-up
	log.Printf("Follow-up triggered: session=%s type=%s priority=%s count=%d",
		event.SessionID, event.Type, event.Priority, session.FollowUpCount)

	f.mu.Unlock()

	// If auto-send is enabled and safety check passes
	// In production, this would send via the appropriate channel
	// For now, we just log it
	log.Printf("Follow-up message for %s: %s", event.SessionID, message)
}

// determineFollowUpType decides the type of follow-up needed
func (f *FollowUpEngine) determineFollowUpType(session *FollowUpSession) FollowUpType {
	hoursSinceContact := time.Since(session.LastContact).Hours()

	// Time-based follow-up (24h+)
	if hoursSinceContact >= 24 {
		return FollowUpTimer
	}

	// Stage-specific follow-ups
	switch session.LastStage {
	case "Presentation":
		if session.EngagementScore > 0.5 {
			return FollowUpStage
		}
	case "Negotiation":
		return FollowUpStage
	case "Discovery":
		if hoursSinceContact >= 12 {
			return FollowUpIntent
		}
	}

	// Win-back for cold leads
	if session.EngagementScore < 0.3 && hoursSinceContact >= 48 {
		return FollowUpWinback
	}

	return FollowUpTimer
}

// generateFollowUpMessage creates a personalized follow-up message
func (f *FollowUpEngine) generateFollowUpMessage(session *FollowUpSession, followUpType FollowUpType) string {
	// Get knowledge base for context
	kb := f.salesLoop.GetSalesKnowledgeBase()

	switch followUpType {
	case FollowUpTimer:
		return f.generateTimerFollowUp(session, kb)
	case FollowUpStage:
		return f.generateStageFollowUp(session, kb)
	case FollowUpIntent:
		return f.generateIntentFollowUp(session, kb)
	case FollowUpWinback:
		return f.generateWinbackFollowUp(session, kb)
	default:
		return f.generateTimerFollowUp(session, kb)
	}
}

func (f *FollowUpEngine) generateTimerFollowUp(session *FollowUpSession, kb *sales_agent.SalesKnowledgeBase) string {
	templates := []string{
		"您好！上次我们聊到%s，想问问您这边考虑得怎么样了？有什么我可以帮您解答的吗？",
		"Hi! Just wanted to check in about %s. Do you have any questions I can help with?",
		"您好，我是SalesMate AI。之前您对%s表现出兴趣，请问现在进展如何？",
	}

	topic := session.LastTopic
	if topic == "" {
		topic = "我们的产品方案"
	}

	return fmt.Sprintf(templates[0], topic)
}

func (f *FollowUpEngine) generateStageFollowUp(session *FollowUpSession, kb *sales_agent.SalesKnowledgeBase) string {
	switch session.LastStage {
	case "Presentation":
		return "根据我们之前的讨论，我准备了一份更适合您需求的方案。您看什么时候方便详细聊聊？"
	case "Negotiation":
		return "关于我们讨论的条款，我这边可以帮您申请一些优惠。您看是否可以安排一个简短的电话确认一下细节？"
	case "Discovery":
		return "我想确认一下您的具体需求，这样可以为您提供更精准的解决方案。您方便的时候我们可以继续聊聊。"
	default:
		return "您好！想了解一下您对之前讨论的内容有什么新的想法吗？"
	}
}

func (f *FollowUpEngine) generateIntentFollowUp(session *FollowUpSession, kb *sales_agent.SalesKnowledgeBase) string {
	return fmt.Sprintf("关于您之前提到的%s，我这边找到了一些相关的成功案例，可以分享给您参考。您有兴趣了解吗？", session.LastTopic)
}

func (f *FollowUpEngine) generateWinbackFollowUp(session *FollowUpSession, kb *sales_agent.SalesKnowledgeBase) string {
	return "您好！很久没有联系了。我们最近有一些新的产品更新和优惠活动，不知道您是否还有兴趣了解？"
}

// calculateNextFollowUp determines the next follow-up time
func (f *FollowUpEngine) calculateNextFollowUp(session *FollowUpSession) time.Time {
	// Base interval based on stage
	var hours int
	switch session.LastStage {
	case "Negotiation", "Close":
		hours = 4
	case "Presentation":
		hours = 12
	case "Discovery":
		hours = 24
	default:
		hours = 48
	}

	// Adjust for engagement
	if session.EngagementScore > 0.7 {
		hours = hours * 2 / 3 // Higher engagement = shorter interval
	} else if session.EngagementScore < 0.3 {
		hours = hours * 4 / 3 // Lower engagement = longer interval
	}

	return time.Now().Add(time.Duration(hours) * time.Hour)
}

// calculatePriority determines follow-up priority
func (f *FollowUpEngine) calculatePriority(session *FollowUpSession) FollowUpPriority {
	// High priority: high engagement, advanced stage
	if session.EngagementScore > 0.7 && (session.LastStage == "Negotiation" || session.LastStage == "Close") {
		return PriorityHigh
	}

	// Low priority: low engagement, early stage
	if session.EngagementScore < 0.3 || session.LastStage == "NewContact" {
		return PriorityLow
	}

	return PriorityMedium
}

// GetSession retrieves a follow-up session
func (f *FollowUpEngine) GetSession(sessionID string) (*FollowUpSession, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	session, exists := f.sessions[sessionID]
	return session, exists
}

// DeleteSession removes a follow-up session
func (f *FollowUpEngine) DeleteSession(sessionID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.sessions, sessionID)
}

// GetStats returns follow-up engine statistics
func (f *FollowUpEngine) GetStats() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var high, medium, low int
	for _, session := range f.sessions {
		switch session.Priority {
		case PriorityHigh:
			high++
		case PriorityMedium:
			medium++
		case PriorityLow:
			low++
		}
	}

	return map[string]interface{}{
		"totalSessions":   len(f.sessions),
		"highPriority":    high,
		"mediumPriority":  medium,
		"lowPriority":     low,
		"pendingFollowUps": len(f.GetPendingFollowUps()),
	}
}