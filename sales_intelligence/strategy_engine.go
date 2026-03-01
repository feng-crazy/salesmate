package sales_intelligence

import (
	"salesmate/sales_agent"
)

// StrategyEngine implements sales strategies like SPIN, FAB, and BANT
type StrategyEngine struct {
	knowledgeBase *sales_agent.SalesKnowledgeBase
}

// NewStrategyEngine creates a new strategy engine
func NewStrategyEngine(kb *sales_agent.SalesKnowledgeBase) *StrategyEngine {
	return &StrategyEngine{
		knowledgeBase: kb,
	}
}

// SPINFramework applies the SPIN selling framework (Situation, Problem, Implication, Need-Payoff)
type SPINFramework struct {
	SituationQuestions    []string // Questions about current situation
	ProblemQuestions     []string // Questions about problems/pain points
	ImplicationQuestions []string // Questions about consequences of problems
	NeedPayoffQuestions  []string // Questions about benefits of solutions
}

// ApplySPIN generates SPIN-based questions based on customer context
func (se *StrategyEngine) ApplySPIN(customerContext map[string]interface{}) *SPINFramework {
	spin := &SPINFramework{}

	// Generate situation questions based on context
	spin.SituationQuestions = se.generateSituationQuestions(customerContext)

	// Generate problem questions based on identified situation
	spin.ProblemQuestions = se.generateProblemQuestions(customerContext)

	// Generate implication questions based on problems
	spin.ImplicationQuestions = se.generateImplicationQuestions(customerContext)

	// Generate need-payoff questions based on implications
	spin.NeedPayoffQuestions = se.generateNeedPayoffQuestions(customerContext)

	return spin
}

// generateSituationQuestions creates questions about the customer's current situation
func (se *StrategyEngine) generateSituationQuestions(context map[string]interface{}) []string {
	var questions []string

	// Industry-specific situation questions
	if industry, ok := context["industry"].(string); ok {
		questions = append(questions,
			"What's your current approach to addressing challenges in the "+industry+" industry?",
			"How long have you been operating in the "+industry+" space?",
			"What are the biggest trends you're seeing in "+industry+" right now?",
		)
	}

	// Company size-based situation questions
	if companySize, ok := context["company_size"].(string); ok {
		questions = append(questions,
			"How many employees do you currently have in your "+companySize+" organization?",
			"What are your main operational challenges at the "+companySize+" level?",
		)
	}

	// Generic situation questions if no specific context
	if len(questions) == 0 {
		questions = append(questions,
			"Tell me about your current process for...",
			"How are you currently handling...",
			"What does your typical day/week look like regarding...",
		)
	}

	return questions
}

// generateProblemQuestions creates questions about customer problems and pain points
func (se *StrategyEngine) generateProblemQuestions(context map[string]interface{}) []string {
	var questions []string

	// Problems related to efficiency
	questions = append(questions,
		"Are you experiencing any bottlenecks in your current process?",
		"What frustrates you most about your current approach?",
		"Where do you spend the most time in your current workflow?",
		"What challenges keep you up at night?",
	)

	// Industry-specific problems
	if industry, ok := context["industry"].(string); ok {
		switch industry {
		case "healthcare":
			questions = append(questions,
				"Are you struggling with patient data management?",
				"What compliance challenges do you face?",
			)
		case "finance":
			questions = append(questions,
				"Are you meeting all regulatory requirements efficiently?",
				"How do you manage risk assessment?",
			)
		case "retail":
			questions = append(questions,
				"Are you optimizing your inventory management?",
				"What challenges do you have with customer retention?",
			)
		}
	}

	return questions
}

// generateImplicationQuestions creates questions about the consequences of problems
func (se *StrategyEngine) generateImplicationQuestions(context map[string]interface{}) []string {
	var questions []string

	questions = append(questions,
		"What impact is this having on your bottom line?",
		"How is this affecting your team's productivity?",
		"What would happen if this problem persisted?",
		"If you don't solve this, what are the long-term consequences?",
	)

	return questions
}

// generateNeedPayoffQuestions creates questions about the benefits of solutions
func (se *StrategyEngine) generateNeedPayoffQuestions(context map[string]interface{}) []string {
	var questions []string

	questions = append(questions,
		"How valuable would it be to solve this?",
		"What would it mean for your business if you could resolve this?",
		"If you had a solution that addressed this, how would you use it?",
		"What would achieving this mean for your team/company?",
	)

	return questions
}

// FABFramework applies the FAB selling framework (Features, Advantages, Benefits)
type FABFramework struct {
	Features   []string // Product/service features
	Advantages []string // Advantages over alternatives
	Benefits   []string // Benefits to the customer
}

// ApplyFAB generates FAB-based messaging based on customer needs
func (se *StrategyEngine) ApplyFAB(productID string, customerNeeds []string) *FABFramework {
	fab := &FABFramework{}

	product, exists := se.knowledgeBase.GetProductByID(productID)
	if !exists {
		return fab // Return empty FAB if product doesn't exist
	}

	// Extract features from the product
	fab.Features = product.Features

	// Generate advantages based on features and customer needs
	fab.Advantages = se.generateAdvantages(*product, customerNeeds)

	// Generate benefits based on features, advantages, and customer needs
	fab.Benefits = se.generateBenefits(*product, customerNeeds)

	return fab
}

// generateAdvantages creates advantage statements for the product
func (se *StrategyEngine) generateAdvantages(product sales_agent.Product, customerNeeds []string) []string {
	var advantages []string

	// Look for features that directly address customer needs
	for _, need := range customerNeeds {
		for _, feature := range product.Features {
			if containsIgnoreCase(feature, need) {
				advantages = append(advantages,
					"Unlike other solutions, our "+product.Name+" specifically addresses "+need,
					"Our "+feature+" gives you an edge over alternatives that lack this capability",
				)
				break
			}
		}
	}

	// Add generic advantages if none found for specific needs
	if len(advantages) == 0 {
		advantages = append(advantages,
			"Our "+product.Name+" offers superior performance compared to standard alternatives",
			"We've designed "+product.Name+" with additional capabilities not found in basic versions",
		)
	}

	return advantages
}

// generateBenefits creates benefit statements for the customer
func (se *StrategyEngine) generateBenefits(product sales_agent.Product, customerNeeds []string) []string {
	var benefits []string

	// Map features to customer benefits
	for _, need := range customerNeeds {
		for _, feature := range product.Features {
			if containsIgnoreCase(feature, need) {
				benefits = append(benefits,
					"With "+feature+", you'll be able to "+deriveBenefitFromFeature(feature, need),
					"This "+feature+" translates to "+quantifyBenefit(feature, need),
				)
				break
			}
		}
	}

	// Add generic benefits if none found for specific needs
	if len(benefits) == 0 {
		benefits = append(benefits,
			"You'll save time with our automated "+product.Name,
			"Your team will be more efficient with "+product.Name+"'s streamlined approach",
			"The ROI from "+product.Name+" typically pays for itself in a few months",
		)
	}

	return benefits
}

// BANTModel implements the BANT qualification framework (Budget, Authority, Need, Timeline)
type BANTModel struct {
	Budget    float64  // Identified budget range
	Authority string   // Decision maker status
	Need      []string // Identified needs
	Timeline  string   // Purchase timeline
	Qualified bool     // Whether the lead is qualified
}

// ApplyBANT qualifies a lead based on BANT criteria
func (se *StrategyEngine) ApplyBANT(information map[string]interface{}) *BANTModel {
	bant := &BANTModel{}

	// Extract budget information
	if budget, ok := information["budget"].(float64); ok {
		bant.Budget = budget
	} else if budgetStr, ok := information["budget"].(string); ok {
		// Parse budget from string if needed
		bant.Budget = parseFloatFromStr(budgetStr)
	}

	// Extract authority information
	if authority, ok := information["authority"].(string); ok {
		bant.Authority = authority
	} else if decisionMaker, ok := information["decision_maker"].(bool); ok {
		if decisionMaker {
			bant.Authority = "primary_decision_maker"
		} else {
			bant.Authority = "influencer"
		}
	}

	// Extract need information
	if needs, ok := information["needs"].([]string); ok {
		bant.Need = needs
	} else if needStr, ok := information["need"].(string); ok {
		bant.Need = []string{needStr}
	}

	// Extract timeline information
	if timeline, ok := information["timeline"].(string); ok {
		bant.Timeline = timeline
	} else if timeframe, ok := information["timeframe"].(string); ok {
		bant.Timeline = timeframe
	}

	// Determine if lead is qualified based on BANT
	bant.Qualified = se.isQualifiedBANT(bant)

	return bant
}

// isQualifiedBANT determines if a lead meets BANT qualification criteria
func (se *StrategyEngine) isQualifiedBANT(bant *BANTModel) bool {
	// A qualified lead should have budget, authority, need, and timeline
	hasBudget := bant.Budget > 0
	hasAuthority := bant.Authority != "" && bant.Authority != "influencer_only"
	hasNeed := len(bant.Need) > 0
	hasTimeline := bant.Timeline != ""

	// Basic BANT qualification: at least 3 of 4 criteria should be met
	qualificationsMet := 0
	if hasBudget { qualificationsMet++ }
	if hasAuthority { qualificationsMet++ }
	if hasNeed { qualificationsMet++ }
	if hasTimeline { qualificationsMet++ }

	return qualificationsMet >= 3
}

// containsIgnoreCase checks if a string contains a substring (case insensitive)
func containsIgnoreCase(source, substr string) bool {
	return containsSubstringIgnoreCase(source, substr)
}

// containsSubstringIgnoreCase implements case-insensitive substring matching
func containsSubstringIgnoreCase(source, substr string) bool {
	sourceLower := toLowerCase(source)
	substrLower := toLowerCase(substr)

	for i := 0; i <= len(sourceLower)-len(substrLower); i++ {
		match := true
		for j := 0; j < len(substrLower); j++ {
			if sourceLower[i+j] != substrLower[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// toLowerCase converts a string to lowercase
func toLowerCase(s string) string {
	var result []byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		result = append(result, c)
	}
	return string(result)
}

// deriveBenefitFromFeature attempts to derive a benefit from a feature
func deriveBenefitFromFeature(feature, need string) string {
	// This is a simplified version - in a real implementation, you'd have a more sophisticated mapping
	return "address " + need + " more effectively"
}

// quantifyBenefit attempts to quantify the benefit of a feature
func quantifyBenefit(feature, need string) string {
	// This is a simplified version - in a real implementation, you'd have more detailed benefit calculations
	return "measurable improvements in " + need + " management"
}

// parseFloatFromStr extracts a float value from a string
func parseFloatFromStr(str string) float64 {
	// This is a simplified implementation - in a real implementation, you'd use strconv.ParseFloat
	var result float64
	var temp float64
	var divisor float64 = 1
	var decimalStarted bool

	for i := 0; i < len(str); i++ {
		c := str[i]
		if c >= '0' && c <= '9' {
			temp = temp*10 + float64(c-'0')
			if decimalStarted {
				divisor *= 10
			}
		} else if c == '.' && !decimalStarted {
			decimalStarted = true
		} else if c == '$' || c == ',' {
			// Skip currency symbols and commas
			continue
		} else {
			break // Stop at first non-numeric character after number starts
		}
	}

	result = temp / divisor
	return result
}

// CalculateWinProbability estimates the probability of winning the deal based on various factors
func (se *StrategyEngine) CalculateWinProbability(bant *BANTModel, engagement int, competition string) float64 {
	probability := 0.5 // Base probability

	// Increase probability if lead is qualified via BANT
	if bant.Qualified {
		probability += 0.3
	}

	// Adjust based on engagement level (scale of 1-5)
	engagementFactor := float64(engagement) / 5.0
	probability += engagementFactor * 0.1

	// Adjust based on competition
	switch competition {
	case "low":
		probability += 0.1
	case "high":
		probability -= 0.2
	}

	// Ensure probability stays within bounds
	if probability > 1.0 {
		probability = 1.0
	} else if probability < 0.0 {
		probability = 0.0
	}

	return probability
}