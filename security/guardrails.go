package security

import (
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

// GuardrailType defines the type of guardrail
type GuardrailType string

const (
	PriceGuardrail      GuardrailType = "price"
	ContractGuardrail   GuardrailType = "contract"
	FeatureGuardrail    GuardrailType = "feature"
	CompetitorGuardrail GuardrailType = "competitor"
	ApprovalGuardrail   GuardrailType = "approval"
	ComplianceGuardrail GuardrailType = "compliance"
)

// Guardrail represents a security constraint that prevents inappropriate actions
type Guardrail struct {
	Type           GuardrailType
	Description    string
	Threshold      float64 // For numerical constraints
	AllowedValues  []string
	BlockedPhrases []string
	Enforcement    EnforcementLevel
	Message        string // Message to return when guardrail is triggered
}

// EnforcementLevel defines how strictly a guardrail is enforced
type EnforcementLevel string

const (
	HardBlock EnforcementLevel = "hard_block" // Completely prevents action
	SoftAlert EnforcementLevel = "soft_alert" // Warns but allows continuation
	Review    EnforcementLevel = "review"     // Requires human review
)

// Violation represents a breach of a guardrail
type Violation struct {
	Guardrail  *Guardrail
	Message    string
	Timestamp  time.Time
	SessionID  string
	Details    map[string]interface{}
	Severity   int // 1-5 scale, 5 being most severe
	Resolved   bool
	Resolution string
}

// GuardrailManager manages all security guardrails
type GuardrailManager struct {
	guardrails    map[GuardrailType][]*Guardrail
	violations    []*Violation
	maxViolations int
	strictMode    bool
	callback      func(violation *Violation)
}

// NewGuardrailManager creates a new guardrail manager
func NewGuardrailManager() *GuardrailManager {
	gm := &GuardrailManager{
		guardrails:    make(map[GuardrailType][]*Guardrail),
		violations:    make([]*Violation, 0),
		maxViolations: 100, // Keep max 100 violations in memory
		strictMode:    true,
	}

	// Initialize with default guardrails
	gm.initializeDefaultGuardrails()

	return gm
}

// initializeDefaultGuardrails sets up standard guardrails for sales operations
func (gm *GuardrailManager) initializeDefaultGuardrails() {
	// Price guardrails - prevent quoting unauthorized prices
	priceGuardrail := &Guardrail{
		Type:          PriceGuardrail,
		Description:   "Prevents quoting prices outside authorized ranges",
		AllowedValues: []string{}, // Will be populated with approved prices
		BlockedPhrases: []string{
			"our lowest price is", "cheaper than", "discount of",
			"special rate for you", "personal discount",
		},
		Enforcement: SoftAlert,
		Message:     "I need to connect you with a sales representative for pricing discussions.",
	}

	// Contract guardrails - prevent making unauthorized commitments
	contractGuardrail := &Guardrail{
		Type:        ContractGuardrail,
		Description: "Prevents making contractual commitments without approval",
		BlockedPhrases: []string{
			"I agree to", "we guarantee", "contract says", "legally bound",
			"officially promise", "written agreement", "binding commitment",
		},
		Enforcement: HardBlock,
		Message:     "I cannot make contractual commitments. Please speak with a sales representative.",
	}

	// Feature guardrails - prevent claiming unverified features
	featureGuardrail := &Guardrail{
		Type:        FeatureGuardrail,
		Description: "Prevents claiming features that aren't verified in our system",
		BlockedPhrases: []string{
			"we definitely have", "our system supports", "we offer",
			"of course we can", "yes, we do that",
		},
		Enforcement: Review,
		Message:     "Let me verify that feature with our product team before confirming.",
	}

	// Competitor guardrails - prevent making false claims about competitors
	competitorGuardrail := &Guardrail{
		Type:        CompetitorGuardrail,
		Description: "Prevents making unsubstantiated claims about competitors",
		BlockedPhrases: []string{
			"they can't", "only we", "unlike them", "better than their",
			"we're the only", "they don't offer",
		},
		Enforcement: Review,
		Message:     "Let me provide factual information about our offerings versus alternatives.",
	}

	// Add all default guardrails
	gm.AddGuardrail(priceGuardrail)
	gm.AddGuardrail(contractGuardrail)
	gm.AddGuardrail(featureGuardrail)
	gm.AddGuardrail(competitorGuardrail)
}

// AddGuardrail adds a new guardrail to the manager
func (gm *GuardrailManager) AddGuardrail(guardrail *Guardrail) {
	if _, exists := gm.guardrails[guardrail.Type]; !exists {
		gm.guardrails[guardrail.Type] = make([]*Guardrail, 0)
	}
	gm.guardrails[guardrail.Type] = append(gm.guardrails[guardrail.Type], guardrail)
}

// CheckMessage checks if a message violates any guardrails
func (gm *GuardrailManager) CheckMessage(message, sessionID string) []*Violation {
	var violations []*Violation

	lowerMessage := strings.ToLower(message)

	for _, guardrailList := range gm.guardrails {
		for _, guardrail := range guardrailList {
			// Check blocked phrases
			for _, phrase := range guardrail.BlockedPhrases {
				if strings.Contains(lowerMessage, strings.ToLower(phrase)) {
					violation := &Violation{
						Guardrail: guardrail,
						Message:   fmt.Sprintf("Message contains blocked phrase: '%s'", phrase),
						Timestamp: time.Now(),
						SessionID: sessionID,
						Details:   map[string]interface{}{"blocked_phrase": phrase, "message": message},
						Severity:  gm.calculateSeverity(guardrail.Enforcement),
					}
					violations = append(violations, violation)

					// Record violation and trigger callback
					gm.recordViolation(violation)
					if gm.callback != nil {
						gm.callback(violation)
					}
				}
			}

			// Additional checks could go here based on guardrail type
			switch guardrail.Type {
			case PriceGuardrail:
				if gm.checkPriceGuardrail(message, guardrail) {
					violation := &Violation{
						Guardrail: guardrail,
						Message:   "Potential unauthorized price disclosure detected",
						Timestamp: time.Now(),
						SessionID: sessionID,
						Details:   map[string]interface{}{"message": message},
						Severity:  gm.calculateSeverity(guardrail.Enforcement),
					}
					violations = append(violations, violation)

					gm.recordViolation(violation)
					if gm.callback != nil {
						gm.callback(violation)
					}
				}
			}
		}
	}

	return violations
}

// checkPriceGuardrail specifically checks for price-related violations
func (gm *GuardrailManager) checkPriceGuardrail(message string, guardrail *Guardrail) bool {
	// Look for price patterns like "$XX.XX", "XX dollars", etc.
	pricePattern := `\$\d+\.?\d*|\d+\s+dollars|\d+\s+USD|\d+\s+euros`
	re := regexp.MustCompile(`(?i)` + pricePattern)

	matches := re.FindAllString(message, -1)

	// If we found price mentions but no authorized values are set, flag it
	if len(matches) > 0 && len(guardrail.AllowedValues) == 0 {
		return true
	}

	// Check if mentioned prices are in allowed values
	for _, match := range matches {
		isAllowed := false
		for _, allowed := range guardrail.AllowedValues {
			if strings.Contains(strings.ToLower(match), strings.ToLower(allowed)) {
				isAllowed = true
				break
			}
		}
		if !isAllowed {
			return true
		}
	}

	return false
}

// calculateSeverity converts enforcement level to a severity score
func (gm *GuardrailManager) calculateSeverity(level EnforcementLevel) int {
	switch level {
	case HardBlock:
		return 5
	case Review:
		return 3
	case SoftAlert:
		return 1
	default:
		return 2
	}
}

// recordViolation stores a violation and maintains the list within size limits
func (gm *GuardrailManager) recordViolation(violation *Violation) {
	gm.violations = append(gm.violations, violation)

	// Maintain max violations limit
	if len(gm.violations) > gm.maxViolations {
		gm.violations = gm.violations[len(gm.violations)-gm.maxViolations:]
	}
}

// IsActionAllowed checks if an action complies with all relevant guardrails
func (gm *GuardrailManager) IsActionAllowed(actionType GuardrailType, content, sessionID string) (bool, string, []*Violation) {
	violations := gm.CheckMessage(content, sessionID)

	// Filter violations by type if specific type requested
	if actionType != "" {
		filtered := make([]*Violation, 0)
		for _, v := range violations {
			if v.Guardrail.Type == actionType {
				filtered = append(filtered, v)
			}
		}
		violations = filtered
	}

	// Check if any violations require hard blocks
	for _, v := range violations {
		if v.Guardrail.Enforcement == HardBlock {
			return false, v.Guardrail.Message, violations
		}
	}

	// If strict mode and any violations exist, block
	if gm.strictMode && len(violations) > 0 {
		return false, "Action requires review due to potential policy violation", violations
	}

	// Otherwise allow with warnings for soft alerts
	var warning string
	if len(violations) > 0 {
		warning = "Action allowed but noted potential issues"
	}

	return true, warning, violations
}

// ResolveViolation marks a violation as resolved
func (gm *GuardrailManager) ResolveViolation(violationID int, resolution string) error {
	if violationID < 0 || violationID >= len(gm.violations) {
		return fmt.Errorf("violation ID %d not found", violationID)
	}

	gm.violations[violationID].Resolved = true
	gm.violations[violationID].Resolution = resolution

	return nil
}

// GetViolations returns all recorded violations
func (gm *GuardrailManager) GetViolations() []*Violation {
	return gm.violations
}

// GetViolationsByType returns violations of a specific type
func (gm *GuardrailManager) GetViolationsByType(guardrailType GuardrailType) []*Violation {
	var result []*Violation
	for _, v := range gm.violations {
		if v.Guardrail.Type == guardrailType {
			result = append(result, v)
		}
	}
	return result
}

// SetStrictMode enables or disables strict enforcement
func (gm *GuardrailManager) SetStrictMode(strict bool) {
	gm.strictMode = strict
}

// SetCallback sets a callback function to be called when violations occur
func (gm *GuardrailManager) SetCallback(callback func(violation *Violation)) {
	gm.callback = callback
}

// SetAllowedPrices sets the prices that are authorized to be quoted
func (gm *GuardrailManager) SetAllowedPrices(prices []string) {
	priceGuardrails := gm.guardrails[PriceGuardrail]
	for _, guardrail := range priceGuardrails {
		guardrail.AllowedValues = prices
	}
}

// SetApprovedFeatures sets the features that are authorized to be claimed
func (gm *GuardrailManager) SetApprovedFeatures(features []string) {
	featureGuardrails := gm.guardrails[FeatureGuardrail]
	for _, guardrail := range featureGuardrails {
		guardrail.AllowedValues = features
	}
}

type EmotionType string

const (
	EmotionAnger          EmotionType = "anger"
	EmotionFrustration    EmotionType = "frustration"
	EmotionDisappointment EmotionType = "disappointment"
	EmotionConfusion      EmotionType = "confusion"
	EmotionHesitation     EmotionType = "hesitation"
	EmotionInterest       EmotionType = "interest"
	EmotionTrust          EmotionType = "trust"
	EmotionExcitement     EmotionType = "excitement"
)

type EmotionSignal struct {
	Type      EmotionType
	Intensity float64
	Keywords  []string
}

type EmotionFuseState struct {
	Triggered       bool
	TriggerCount    int
	LastTriggerTime time.Time
	EmotionHistory  []EmotionSignal
	SessionID       string
	HumanNotified   bool
	AutoReplyPaused bool
}

type EmotionFuser struct {
	mu                     sync.RWMutex
	threshold              float64
	sessionStates          map[string]*EmotionFuseState
	notifyCallback         func(sessionID string, state *EmotionFuseState)
	angerKeywords          []string
	frustrationKeywords    []string
	disappointmentKeywords []string
	triggerCooldown        time.Duration
}

func NewEmotionFuser(threshold float64) *EmotionFuser {
	return &EmotionFuser{
		threshold:     threshold,
		sessionStates: make(map[string]*EmotionFuseState),
		angerKeywords: []string{
			"unacceptable", "ridiculous", "terrible", "worst",
			"hate", "angry", "furious", "speak to manager",
			"complaint", "lawyer", "sue", "cancel",
		},
		frustrationKeywords: []string{
			"frustrated", "annoying", "waste of time",
			"not working", "broken", "useless", "stupid",
			"doesn't work", "never works", "always fails",
		},
		disappointmentKeywords: []string{
			"disappointed", "expected better", "not what I expected",
			"let down", "underwhelming", "not worth it",
		},
		triggerCooldown: 5 * time.Minute,
	}
}

func (ef *EmotionFuser) SetNotifyCallback(callback func(sessionID string, state *EmotionFuseState)) {
	ef.notifyCallback = callback
}

func (ef *EmotionFuser) Analyze(message string) []EmotionSignal {
	signals := []EmotionSignal{}
	lowerMsg := strings.ToLower(message)

	angerMatches := ef.countKeywordMatches(lowerMsg, ef.angerKeywords)
	if angerMatches > 0 {
		intensity := float64(angerMatches) * 0.3
		if intensity > 1.0 {
			intensity = 1.0
		}
		signals = append(signals, EmotionSignal{
			Type:      EmotionAnger,
			Intensity: intensity,
		})
	}

	frustrationMatches := ef.countKeywordMatches(lowerMsg, ef.frustrationKeywords)
	if frustrationMatches > 0 {
		intensity := float64(frustrationMatches) * 0.25
		if intensity > 1.0 {
			intensity = 1.0
		}
		signals = append(signals, EmotionSignal{
			Type:      EmotionFrustration,
			Intensity: intensity,
		})
	}

	disappointmentMatches := ef.countKeywordMatches(lowerMsg, ef.disappointmentKeywords)
	if disappointmentMatches > 0 {
		intensity := float64(disappointmentMatches) * 0.2
		if intensity > 1.0 {
			intensity = 1.0
		}
		signals = append(signals, EmotionSignal{
			Type:      EmotionDisappointment,
			Intensity: intensity,
		})
	}

	exclamationCount := strings.Count(message, "!")
	if exclamationCount >= 3 {
		for i := range signals {
			signals[i].Intensity += 0.2
			if signals[i].Intensity > 1.0 {
				signals[i].Intensity = 1.0
			}
		}
	}

	return signals
}

func (ef *EmotionFuser) countKeywordMatches(message string, keywords []string) int {
	count := 0
	for _, keyword := range keywords {
		if strings.Contains(message, keyword) {
			count++
		}
	}
	return count
}

func (ef *EmotionFuser) Check(sessionID, message string) *EmotionFuseState {
	ef.mu.Lock()
	defer ef.mu.Unlock()

	state, exists := ef.sessionStates[sessionID]
	if !exists {
		state = &EmotionFuseState{
			SessionID:      sessionID,
			EmotionHistory: []EmotionSignal{},
		}
		ef.sessionStates[sessionID] = state
	}

	signals := ef.Analyze(message)
	state.EmotionHistory = append(state.EmotionHistory, signals...)

	maxIntensity := 0.0
	for _, signal := range signals {
		if signal.Intensity > maxIntensity {
			maxIntensity = signal.Intensity
		}
	}

	shouldTrigger := false
	for _, signal := range signals {
		if (signal.Type == EmotionAnger || signal.Type == EmotionFrustration) && signal.Intensity >= ef.threshold {
			shouldTrigger = true
			break
		}
	}

	if shouldTrigger {
		now := time.Now()
		if !state.Triggered || now.Sub(state.LastTriggerTime) > ef.triggerCooldown {
			state.Triggered = true
			state.TriggerCount++
			state.LastTriggerTime = now
			state.AutoReplyPaused = true

			if ef.notifyCallback != nil && !state.HumanNotified {
				state.HumanNotified = true
				go ef.notifyCallback(sessionID, state)
			}
		}
	}

	return state
}

func (ef *EmotionFuser) IsFused(sessionID string) bool {
	ef.mu.RLock()
	defer ef.mu.RUnlock()

	state, exists := ef.sessionStates[sessionID]
	if !exists {
		return false
	}

	return state.Triggered && state.AutoReplyPaused
}

func (ef *EmotionFuser) Reset(sessionID string) {
	ef.mu.Lock()
	defer ef.mu.Unlock()

	delete(ef.sessionStates, sessionID)
}

func (ef *EmotionFuser) Resume(sessionID string) {
	ef.mu.Lock()
	defer ef.mu.Unlock()

	if state, exists := ef.sessionStates[sessionID]; exists {
		state.AutoReplyPaused = false
		state.Triggered = false
		state.HumanNotified = false
	}
}

func (ef *EmotionFuser) GetState(sessionID string) *EmotionFuseState {
	ef.mu.RLock()
	defer ef.mu.RUnlock()

	if state, exists := ef.sessionStates[sessionID]; exists {
		return state
	}
	return nil
}

func (ef *EmotionFuser) SetThreshold(threshold float64) {
	ef.mu.Lock()
	defer ef.mu.Unlock()

	ef.threshold = threshold
}

type SafetySystem struct {
	guardrails       *GuardrailManager
	emotionFuser     *EmotionFuser
	humanHandler     func(sessionID, reason, summary string)
	autoReplyEnabled bool
}

func NewSafetySystem(emotionThreshold float64) *SafetySystem {
	return &SafetySystem{
		guardrails:       NewGuardrailManager(),
		emotionFuser:     NewEmotionFuser(emotionThreshold),
		autoReplyEnabled: true,
	}
}

func (ss *SafetySystem) SetHumanHandler(handler func(sessionID, reason, summary string)) {
	ss.humanHandler = handler
	ss.emotionFuser.SetNotifyCallback(func(sessionID string, state *EmotionFuseState) {
		if ss.humanHandler != nil {
			summary := fmt.Sprintf("Emotion fuse triggered. Trigger count: %d, Last emotions: %v",
				state.TriggerCount, state.EmotionHistory)
			ss.humanHandler(sessionID, "emotion_fuse", summary)
		}
	})
}

type SafetyCheckResult struct {
	AllowAutoReply  bool
	RequiresReview  bool
	TriggeredRules  []string
	Violations      []*Violation
	EmotionState    *EmotionFuseState
	SuggestedAction string
}

func (ss *SafetySystem) Check(sessionID, message string) *SafetyCheckResult {
	result := &SafetyCheckResult{
		AllowAutoReply: ss.autoReplyEnabled,
	}

	violations := ss.guardrails.CheckMessage(message, sessionID)
	result.Violations = violations

	for _, v := range violations {
		result.TriggeredRules = append(result.TriggeredRules, string(v.Guardrail.Type))
		if v.Guardrail.Enforcement == HardBlock {
			result.AllowAutoReply = false
			result.RequiresReview = true
		}
	}

	emotionState := ss.emotionFuser.Check(sessionID, message)
	result.EmotionState = emotionState

	if emotionState.Triggered && emotionState.AutoReplyPaused {
		result.AllowAutoReply = false
		result.RequiresReview = true
		result.TriggeredRules = append(result.TriggeredRules, "emotion_fuse")
	}

	if len(violations) > 0 && result.AllowAutoReply {
		result.RequiresReview = true
	}

	switch {
	case !result.AllowAutoReply && len(result.TriggeredRules) > 0:
		result.SuggestedAction = "human_takeover"
	case result.RequiresReview:
		result.SuggestedAction = "draft_review"
	default:
		result.SuggestedAction = "auto_reply"
	}

	return result
}

func (ss *SafetySystem) CheckReply(sessionID, reply string) *SafetyCheckResult {
	result := &SafetyCheckResult{
		AllowAutoReply: true,
	}

	violations := ss.guardrails.CheckMessage(reply, sessionID)
	result.Violations = violations

	for _, v := range violations {
		result.TriggeredRules = append(result.TriggeredRules, string(v.Guardrail.Type))
		if v.Guardrail.Enforcement == HardBlock {
			result.AllowAutoReply = false
			result.RequiresReview = true
		}
	}

	if !result.AllowAutoReply {
		result.SuggestedAction = "block_and_revise"
	} else if len(violations) > 0 {
		result.RequiresReview = true
		result.SuggestedAction = "review_before_send"
	} else {
		result.SuggestedAction = "approve"
	}

	return result
}

func (ss *SafetySystem) IsEmotionFused(sessionID string) bool {
	return ss.emotionFuser.IsFused(sessionID)
}

func (ss *SafetySystem) ResetEmotionFuse(sessionID string) {
	ss.emotionFuser.Reset(sessionID)
}

func (ss *SafetySystem) ResumeSession(sessionID string) {
	ss.emotionFuser.Resume(sessionID)
}

func (ss *SafetySystem) SetAutoReplyEnabled(enabled bool) {
	ss.autoReplyEnabled = enabled
}

func (ss *SafetySystem) GetGuardrailManager() *GuardrailManager {
	return ss.guardrails
}

func (ss *SafetySystem) GetEmotionFuser() *EmotionFuser {
	return ss.emotionFuser
}
