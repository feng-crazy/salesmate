package sales_agent

import (
	"strings"
	"time"
)

// EmotionAnalyzer analyzes emotional signals in customer communications
type EmotionAnalyzer struct {
	interestKeywords []string
	concernKeywords  []string
	enthusiasmKeywords []string
	hesitationKeywords []string
	frustrationKeywords []string
}

// EmotionSignal represents emotional signals detected in customer communications
type EmotionSignal struct {
	Type      string  // e.g., "interest", "concern", "hesitation", "excitement", "frustration"
	Intensity float64 // 0.0 to 1.0 representing strength of emotion
	Timestamp time.Time
}

// NewEmotionAnalyzer creates a new emotion analyzer
func NewEmotionAnalyzer() *EmotionAnalyzer {
	return &EmotionAnalyzer{
		interestKeywords: []string{
			"interesting", "good", "like", "love", "cool", "awesome", "great", "amazing",
			"wonderful", "fantastic", "brilliant", "exciting", "appealing", "attractive",
			"curious", "intriguing", "fascinating", "remarkable", "impressive", "stunning",
		},
		concernKeywords: []string{
			"worry", "concern", "worried", "nervous", "scared", "afraid", "anxious",
			"doubt", "doubtful", "uncertain", "unsure", "hesitant", "apprehensive",
			"question", "questions", "worries", "concerns", "hesitation", "caution",
		},
		enthusiasmKeywords: []string{
			"excited", "exciting", "love", "amazing", "fantastic", "incredible",
			"awesome", "wonderful", "fantastic", "superb", "outstanding", "brilliant",
			"excellent", "fabulous", "marvelous", "terrific", "super", "phenomenal",
		},
		hesitationKeywords: []string{
			"maybe", "perhaps", "possibly", "might", "could", "should",
			"not sure", "unsure", "undecided", "indecisive", "hesitant",
			"wait", "slow", "think", "consider", "pause", "delay", "postpone",
		},
		frustrationKeywords: []string{
			"angry", "frustrated", "annoyed", "mad", "upset", "irritated",
			"problem", "issue", "difficult", "hard", "struggle", "trouble",
			"doesn't work", "bug", "error", "failure", "broken", "incorrect",
		},
	}
}

// AnalyzeEmotion analyzes emotional signals in a message
func (ea *EmotionAnalyzer) AnalyzeEmotion(message string) []EmotionSignal {
	message = strings.ToLower(message)
	signals := []EmotionSignal{}

	// Analyze for interest
	interestCount := ea.countKeywords(message, ea.interestKeywords)
	if interestCount > 0 {
		intensity := float64(interestCount) / 5.0
		if intensity > 1.0 {
			intensity = 1.0
		}
		signals = append(signals, EmotionSignal{
			Type:      "interest",
			Intensity: intensity,
			Timestamp: time.Now(),
		})
	}

	// Analyze for concern
	concernCount := ea.countKeywords(message, ea.concernKeywords)
	if concernCount > 0 {
		intensity := float64(concernCount) / 3.0
		if intensity > 1.0 {
			intensity = 1.0
		}
		signals = append(signals, EmotionSignal{
			Type:      "concern",
			Intensity: intensity,
			Timestamp: time.Now(),
		})
	}

	// Analyze for enthusiasm
	enthusiasmCount := ea.countKeywords(message, ea.enthusiasmKeywords)
	if enthusiasmCount > 0 {
		intensity := float64(enthusiasmCount) / 3.0
		if intensity > 1.0 {
			intensity = 1.0
		}
		signals = append(signals, EmotionSignal{
			Type:      "excitement",
			Intensity: intensity,
			Timestamp: time.Now(),
		})
	}

	// Analyze for hesitation
	hesitationCount := ea.countKeywords(message, ea.hesitationKeywords)
	if hesitationCount > 0 {
		intensity := float64(hesitationCount) / 4.0
		if intensity > 1.0 {
			intensity = 1.0
		}
		signals = append(signals, EmotionSignal{
			Type:      "hesitation",
			Intensity: intensity,
			Timestamp: time.Now(),
		})
	}

	// Analyze for frustration
	frustrationCount := ea.countKeywords(message, ea.frustrationKeywords)
	if frustrationCount > 0 {
		intensity := float64(frustrationCount) / 3.0
		if intensity > 1.0 {
			intensity = 1.0
		}
		signals = append(signals, EmotionSignal{
			Type:      "frustration",
			Intensity: intensity,
			Timestamp: time.Now(),
		})
	}

	return signals
}

// countKeywords counts occurrences of keywords in a message
func (ea *EmotionAnalyzer) countKeywords(message string, keywords []string) int {
	count := 0
	for _, keyword := range keywords {
		if strings.Contains(message, keyword) {
			count++
		}
	}
	return count
}