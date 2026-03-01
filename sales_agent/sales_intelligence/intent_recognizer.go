package sales_intelligence

import (
	"strings"
)

// IntentRecognizer identifies sales-related intents from customer messages
type IntentRecognizer struct {
	keywords map[SalesIntent][]string
}

// SalesIntent represents detected sales intentions from customer messages
type SalesIntent string

const (
	InquiryIntent      SalesIntent = "inquiry"      // General inquiry about product/service
	PricingIntent      SalesIntent = "pricing"      // Asking about pricing
	DemoIntent         SalesIntent = "demo"         // Requesting demo/trial
	CompetitorIntent   SalesIntent = "competitor"   // Comparing with competitors
	ObjectionIntent    SalesIntent = "objection"    // Raising objections
	BudgetIntent       SalesIntent = "budget"       // Discussing budget constraints
	AuthorityIntent    SalesIntent = "authority"    // Decision maker identification
	TimelineIntent     SalesIntent = "timeline"     // Asking about timelines
	FeatureIntent      SalesIntent = "feature"      // Inquiring about specific features
	IntegrationIntent  SalesIntent = "integration"  // Integration requirements
	ContractIntent     SalesIntent = "contract"     // Contract/terms discussion
	ChurnRiskIntent    SalesIntent = "churn_risk"   // Signs of potential churn
	ReferralIntent     SalesIntent = "referral"     // Asking for references/case studies
)

// NewIntentRecognizer creates a new intent recognizer
func NewIntentRecognizer() *IntentRecognizer {
	return &IntentRecognizer{
		keywords: map[SalesIntent][]string{
			PricingIntent: {
				"price", "cost", "pricing", "expensive", "cheap", "worth", "investment",
				"dollars", "cents", "$", "budget", "afford", "payment", "fee", "rate",
				"monthly", "yearly", "annual", "per month", "per year", "quote", "quotation",
			},
			DemoIntent: {
				"demo", "trial", "try", "show me", "see", "watch", "how it works",
				"live", "hands-on", "experience", "play", "test", "preview", "sample",
			},
			CompetitorIntent: {
				"compared", "compare", "vs", "versus", "alternative", "other",
				"better than", "instead of", "like", "same as", "similar to", "competitor",
			},
			ObjectionIntent: {
				"don't", "not", "no", "never", "can't", "won't", "too expensive",
				"doesn't work", "not sure", "maybe later", "not interested", "no thanks",
				"problem", "issue", "concern", "worried", "hesitant", "difficult",
			},
			BudgetIntent: {
				"budget", "funding", "money", "finance", "capital", "expense", "ROI",
				"return on investment", "cost effective", "value", "pay", "spend", "afford",
				"investment", "financial", "economic", "economical",
			},
			AuthorityIntent: {
				"decision", "decide", "approval", "approver", "authorize", "authority",
				"boss", "manager", "director", "CEO", "CTO", "VP", "executive", "lead",
				"I need to talk to", "who decides", "who approves", "authorized",
			},
			TimelineIntent: {
				"when", "time", "date", "schedule", "soon", "later", "now", "today",
				"tomorrow", "next week", "next month", "asap", "urgent", "deadline",
				"implementation", "start", "begin", "delivery", "shipping", "deployment",
			},
			FeatureIntent: {
				"feature", "capability", "function", "does it", "can it", "ability",
				"support", "work with", "integrates", "available", "possibility", "option",
				"include", "has", "with", "built-in", "native", "built for",
			},
			IntegrationIntent: {
				"integrate", "integration", "connect", "API", "interface", "link",
				"work with", "compatible", "plug-in", "extension", "system", "platform",
				"data transfer", "sync", "exchange", "import", "export", "migration",
			},
			ContractIntent: {
				"contract", "agreement", "terms", "conditions", "legal", "SLA",
				"license", "subscription", "sign", "signed", "signature", "binding",
				"paperwork", "documentation", "document", "clause", "obligation",
			},
			ChurnRiskIntent: {
				"cancel", "stop", "quit", "unsubscribe", "leave", "switch",
				"not happy", "disappointed", "problems", "issues", "regret", "wrong",
				"need something else", "looking elsewhere", "unsatisfied", "poor service",
			},
			ReferralIntent: {
				"case study", "example", "reference", "testimonial", "review",
				"success story", "customer", "client", "who else", "others", "proof",
				"users", "clients", "references", "recommendations", "social proof",
			},
		},
	}
}

// RecognizeIntent identifies the primary sales intent from a message
func (ir *IntentRecognizer) RecognizeIntent(message string) SalesIntent {
	message = strings.ToLower(message)

	// Score intents based on keyword matches
	intentScores := make(map[SalesIntent]int)

	for intent, keywords := range ir.keywords {
		for _, keyword := range keywords {
			if strings.Contains(message, keyword) {
				intentScores[intent]++
			}
		}
	}

	// Find the highest scoring intent
	var bestIntent SalesIntent
	bestScore := 0

	for intent, score := range intentScores {
		if score > bestScore {
			bestScore = score
			bestIntent = intent
		}
	}

	// Special handling for inquiry intent (if no other intent matches well)
	if bestScore == 0 {
		inquiryKeywords := []string{"what", "how", "is it", "does", "tell me", "about", "information"}
		inquiryCount := 0

		for _, keyword := range inquiryKeywords {
			if strings.Contains(message, keyword) {
				inquiryCount++
			}
		}

		if inquiryCount > 0 {
			return InquiryIntent
		}

		// Default to inquiry if we can't identify anything specific
		return ""
	}

	return bestIntent
}

// GetIntentKeywords returns all keywords associated with an intent
func (ir *IntentRecognizer) GetIntentKeywords(intent SalesIntent) []string {
	if keywords, exists := ir.keywords[intent]; exists {
		return keywords
	}
	return []string{}
}