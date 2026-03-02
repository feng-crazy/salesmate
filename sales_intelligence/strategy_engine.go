package sales_intelligence

// StrategyEngine implements sales strategies like SPIN, FAB, and BANT
type StrategyEngine struct {
	// Standalone - no external dependencies
}

// NewStrategyEngine creates a new strategy engine (no dependencies needed)
func NewStrategyEngine() *StrategyEngine {
	return &StrategyEngine{}
}

// SPINFramework applies the SPIN selling framework (Situation, Problem, Implication, Need-Payoff)
type SPINFramework struct {
	SituationQuestions   []string // Questions about current situation
	ProblemQuestions     []string // Questions about problems/pain points
	ImplicationQuestions []string // Questions about consequences of problems
	NeedPayoffQuestions  []string // Questions about benefits of solutions
}

// ApplySPIN generates SPIN-based questions based on customer context
func (se *StrategyEngine) ApplySPIN(customerContext map[string]interface{}) *SPINFramework {
	Spin := &SPINFramework{}

	// Generate situation questions based on context
	Spin.SituationQuestions = se.generateSituationQuestions(customerContext)

	// Generate problem questions based on identified situation
	Spin.ProblemQuestions = se.generateProblemQuestions(customerContext)

	// Generate implication questions based on problems
	Spin.ImplicationQuestions = se.generateImplicationQuestions(customerContext)

	// Generate need-payoff questions based on implications
	Spin.NeedPayoffQuestions = se.generateNeedPayoffQuestions(customerContext)

	return Spin
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

	// Generate generic features based on customer needs
	fab.Features = []string{
		"Advanced automation capabilities",
		"Real-time analytics dashboard",
		"Seamless integration with existing tools",
		"Dedicated support team",
	}

	// Generate advantages
	fab.Advantages = se.generateAdvantages(customerNeeds)

	// Generate benefits
	fab.Benefits = se.generateBenefits(customerNeeds)

	return fab
}

// generateAdvantages creates advantage statements
func (se *StrategyEngine) generateAdvantages(customerNeeds []string) []string {
	var advantages []string

	advantages = append(advantages,
		"Our solution offers superior performance compared to standard alternatives",
		"We've designed our solution with additional capabilities not found in basic versions",
		"Our platform provides unmatched flexibility and scalability",
	)

	return advantages
}

// generateBenefits creates benefit statements for the customer
func (se *StrategyEngine) generateBenefits(customerNeeds []string) []string {
	var benefits []string

	benefits = append(benefits,
		"You'll save significant time with our automated solution",
		"Your team will be more efficient with our streamlined approach",
		"The ROI from our solution typically pays for itself in a few months",
	)

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
	if hasBudget {
		qualificationsMet++
	}
	if hasAuthority {
		qualificationsMet++
	}
	if hasNeed {
		qualificationsMet++
	}
	if hasTimeline {
		qualificationsMet++
	}

	return qualificationsMet >= 3
}

// parseFloatFromStr extracts a float value from a string
func parseFloatFromStr(str string) float64 {
	// Simplified implementation
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
			continue // Skip currency symbols and commas
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
