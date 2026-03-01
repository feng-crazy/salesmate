package sales_agent

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type ConfidenceLevel string

const (
	HighConfidence   ConfidenceLevel = "high"
	MediumConfidence ConfidenceLevel = "medium"
	LowConfidence    ConfidenceLevel = "low"
)

const (
	HighThreshold   = 0.90
	MediumThreshold = 0.60
)

type RoutingDecision struct {
	Level         ConfidenceLevel
	Action        RoutingAction
	HumanNotify   bool
	AutoSend      bool
	Reason        string
	Score         float64
	Timestamp     time.Time
	SessionID     string
	OriginalMsg   string
	ProposedReply string
}

type RoutingAction string

const (
	ActionAutoReply     RoutingAction = "auto_reply"
	ActionDraftReview   RoutingAction = "draft_review"
	ActionHumanTakeover RoutingAction = "human_takeover"
	ActionEmotionFuse   RoutingAction = "emotion_fuse"
)

type ConfidenceRouter struct {
	mu               sync.RWMutex
	sessionScores    map[string]float64
	emotionAnalyzer  *EmotionAnalyzer
	intentRecognizer interface {
		RecognizeIntent(message string) string
	}
	humanCallback    func(decision *RoutingDecision)
	emotionCallback  func(decision *RoutingDecision)
	strictMode       bool
	autoReplyEnabled bool
}

type ConfidenceRouterConfig struct {
	StrictMode       bool
	AutoReplyEnabled bool
}

func NewConfidenceRouter(cfg *ConfidenceRouterConfig) *ConfidenceRouter {
	return &ConfidenceRouter{
		sessionScores:    make(map[string]float64),
		emotionAnalyzer:  NewEmotionAnalyzer(),
		strictMode:       cfg.StrictMode,
		autoReplyEnabled: cfg.AutoReplyEnabled,
	}
}

func (cr *ConfidenceRouter) SetEmotionAnalyzer(analyzer *EmotionAnalyzer) {
	cr.emotionAnalyzer = analyzer
}

func (cr *ConfidenceRouter) SetIntentRecognizer(recognizer interface {
	RecognizeIntent(message string) string
}) {
	cr.intentRecognizer = recognizer
}

func (cr *ConfidenceRouter) SetHumanCallback(callback func(decision *RoutingDecision)) {
	cr.humanCallback = callback
}

func (cr *ConfidenceRouter) SetEmotionCallback(callback func(decision *RoutingDecision)) {
	cr.emotionCallback = callback
}

func (cr *ConfidenceRouter) CalculateConfidence(sessionID, message string, proposedReply string) float64 {
	score := 0.5

	if cr.intentRecognizer != nil {
		intent := cr.intentRecognizer.RecognizeIntent(message)
		score += cr.getIntentConfidenceBonus(intent)
	}

	if cr.emotionAnalyzer != nil {
		emotions := cr.emotionAnalyzer.AnalyzeEmotion(message)
		score += cr.getEmotionConfidenceModifier(emotions)
	}

	score += cr.getReplyQualityScore(proposedReply)

	score += cr.getSessionContextBonus(sessionID)

	if score > 1.0 {
		score = 1.0
	} else if score < 0.0 {
		score = 0.0
	}

	cr.mu.Lock()
	cr.sessionScores[sessionID] = score
	cr.mu.Unlock()

	return score
}

func (cr *ConfidenceRouter) getIntentConfidenceBonus(intent string) float64 {
	bonusMap := map[string]float64{
		"inquiry":    0.15,
		"feature":    0.10,
		"pricing":    0.05,
		"demo":       0.20,
		"competitor": -0.05,
		"objection":  -0.10,
		"budget":     0.10,
		"authority":  0.15,
		"timeline":   0.10,
		"contract":   0.25,
		"churn_risk": -0.30,
	}

	if bonus, exists := bonusMap[intent]; exists {
		return bonus
	}
	return 0.0
}

func (cr *ConfidenceRouter) getEmotionConfidenceModifier(emotions []EmotionSignal) float64 {
	var modifier float64
	for _, emotion := range emotions {
		switch emotion.Type {
		case "interest", "excitement":
			modifier += emotion.Intensity * 0.15
		case "hesitation":
			modifier -= emotion.Intensity * 0.10
		case "frustration", "anger":
			modifier -= emotion.Intensity * 0.20
		case "trust":
			modifier += emotion.Intensity * 0.10
		}
	}
	return modifier
}

func (cr *ConfidenceRouter) getReplyQualityScore(reply string) float64 {
	if len(reply) == 0 {
		return -0.20
	}

	score := 0.0

	uncertainPhrases := []string{
		"i'm not sure", "i don't know", "i cannot",
		"i'm unable to", "i apologize", "i'm sorry",
	}
	for _, phrase := range uncertainPhrases {
		if strings.Contains(strings.ToLower(reply), phrase) {
			score -= 0.10
		}
	}

	confidentPhrases := []string{
		"i can confirm", "absolutely", "definitely",
		"our solution", "i recommend", "based on",
	}
	for _, phrase := range confidentPhrases {
		if strings.Contains(strings.ToLower(reply), phrase) {
			score += 0.05
		}
	}

	overcommitPhrases := []string{
		"i guarantee", "we promise", "we will definitely",
		"100%", "always", "never fail",
	}
	for _, phrase := range overcommitPhrases {
		if strings.Contains(strings.ToLower(reply), phrase) {
			score -= 0.15
		}
	}

	return score
}

func (cr *ConfidenceRouter) getSessionContextBonus(sessionID string) float64 {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	if score, exists := cr.sessionScores[sessionID]; exists {
		return (score - 0.5) * 0.1
	}
	return 0.0
}

func (cr *ConfidenceRouter) Route(sessionID, message, proposedReply string) *RoutingDecision {
	confidence := cr.CalculateConfidence(sessionID, message, proposedReply)

	decision := &RoutingDecision{
		Score:         confidence,
		Timestamp:     time.Now(),
		SessionID:     sessionID,
		OriginalMsg:   message,
		ProposedReply: proposedReply,
	}

	emotionFuse := cr.checkEmotionFuse(message)
	if emotionFuse {
		decision.Level = LowConfidence
		decision.Action = ActionEmotionFuse
		decision.AutoSend = false
		decision.HumanNotify = true
		decision.Reason = "Emotion fuse triggered - customer appears frustrated or angry"

		if cr.emotionCallback != nil {
			cr.emotionCallback(decision)
		}
		return decision
	}

	switch {
	case confidence >= HighThreshold:
		decision.Level = HighConfidence
		decision.Action = ActionAutoReply
		decision.AutoSend = cr.autoReplyEnabled
		decision.HumanNotify = false
		decision.Reason = fmt.Sprintf("High confidence (%.2f) - safe to auto-respond", confidence)

	case confidence >= MediumThreshold:
		decision.Level = MediumConfidence
		decision.Action = ActionDraftReview
		decision.AutoSend = false
		decision.HumanNotify = true
		decision.Reason = fmt.Sprintf("Medium confidence (%.2f) - requires human review before sending", confidence)

	default:
		decision.Level = LowConfidence
		decision.Action = ActionHumanTakeover
		decision.AutoSend = false
		decision.HumanNotify = true
		decision.Reason = fmt.Sprintf("Low confidence (%.2f) - human takeover recommended", confidence)
	}

	if cr.strictMode && decision.Level != HighConfidence {
		decision.AutoSend = false
		decision.HumanNotify = true
	}

	if decision.HumanNotify && cr.humanCallback != nil {
		cr.humanCallback(decision)
	}

	return decision
}

func (cr *ConfidenceRouter) checkEmotionFuse(message string) bool {
	if cr.emotionAnalyzer == nil {
		return false
	}

	emotions := cr.emotionAnalyzer.AnalyzeEmotion(message)
	for _, emotion := range emotions {
		if (emotion.Type == "frustration" || emotion.Type == "anger") && emotion.Intensity > 0.7 {
			return true
		}
	}

	angerKeywords := []string{
		"speak to manager", "complaint", "unacceptable",
		"ridiculous", "terrible", "worst", "hate",
		"cancel", "refund", "lawyer", "sue",
	}
	lowerMsg := strings.ToLower(message)
	for _, keyword := range angerKeywords {
		if strings.Contains(lowerMsg, keyword) {
			return true
		}
	}

	return false
}

func (cr *ConfidenceRouter) GetSessionConfidence(sessionID string) float64 {
	cr.mu.RLock()
	defer cr.mu.RUnlock()

	if score, exists := cr.sessionScores[sessionID]; exists {
		return score
	}
	return 0.5
}

func (cr *ConfidenceRouter) UpdateSessionConfidence(sessionID string, delta float64) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	if score, exists := cr.sessionScores[sessionID]; exists {
		newScore := score + delta
		if newScore > 1.0 {
			newScore = 1.0
		} else if newScore < 0.0 {
			newScore = 0.0
		}
		cr.sessionScores[sessionID] = newScore
	} else {
		cr.sessionScores[sessionID] = 0.5 + delta
	}
}

func (cr *ConfidenceRouter) ResetSessionConfidence(sessionID string) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	delete(cr.sessionScores, sessionID)
}

func (cr *ConfidenceRouter) SetStrictMode(strict bool) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	cr.strictMode = strict
}

func (cr *ConfidenceRouter) SetAutoReplyEnabled(enabled bool) {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	cr.autoReplyEnabled = enabled
}

func (d *RoutingDecision) String() string {
	return fmt.Sprintf(
		"RoutingDecision{Level: %s, Action: %s, Score: %.2f, AutoSend: %v, Reason: %s}",
		d.Level, d.Action, d.Score, d.AutoSend, d.Reason,
	)
}

func (d *RoutingDecision) ShouldAutoSend() bool {
	return d.AutoSend && d.Action == ActionAutoReply
}

func (d *RoutingDecision) NeedsHumanReview() bool {
	return d.Action == ActionDraftReview || d.Action == ActionHumanTakeover || d.Action == ActionEmotionFuse
}

func (d *RoutingDecision) IsEmotionFuse() bool {
	return d.Action == ActionEmotionFuse
}
