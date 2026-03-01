package sales_agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	agentcontext "salesmate/agent/context"
	"salesmate/agent/memory"
	"salesmate/agent/skills"
	"salesmate/agent/subagent"
	"salesmate/agent/tools"
	"salesmate/config"
	"salesmate/cron"
	"salesmate/providers"
	"salesmate/session"
	"salesmate/sales_agent/sales_intelligence"
)

// SalesLoop extends the base AgentLoop with sales-specific functionality
type SalesLoop struct {
	config            *config.Config
	provider          providers.LLMProvider
	workspace         string
	model             string
	maxTokens         int
	temperature       float64
	maxIterations     int
	memoryWindow      int
	toolRegistry      *tools.ToolRegistry
	sessionManager    *session.SessionManager
	cronService       *cron.CronService
	skillsLoader      *skills.SkillsLoader
	contextBuilder    *agentcontext.ContextBuilder
	memoryStore       *memory.MemoryStore
	subagentManager   *subagent.SubagentManager

	// Sales-specific components
	salesKB           *SalesKnowledgeBase
	intentRecognizer  *sales_intelligence.IntentRecognizer
	emotionAnalyzer   *EmotionAnalyzer
	pipelineManager   *SalesPipelineManager
	currentStage      SalesStage
	confidenceScore   float64
	lastInteraction   time.Time
}

// NewSalesLoop creates a new sales loop with the given configuration
func NewSalesLoop(cfg *config.Config) (*SalesLoop, error) {
	workspace := cfg.GetWorkspacePath()

	// Create the provider
	provider, err := providers.ProviderFactory(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Create tool registry with available tools
	toolRegistry := tools.NewToolRegistry()

	// Add file tools
	toolRegistry.Register(tools.NewReadFileTool(workspace, ""))
	toolRegistry.Register(tools.NewWriteFileTool(workspace, ""))
	toolRegistry.Register(tools.NewListDirTool(workspace, ""))
	toolRegistry.Register(tools.NewEditFileTool(workspace, ""))

	// Add exec tool
	execTool := tools.NewExecTool(workspace, cfg.Tools.Exec.Timeout, cfg.Tools.RestrictToWorkspace)
	toolRegistry.Register(execTool)

	// Add web tools
	webSearchTool := tools.NewWebSearchTool(cfg.Tools.Web.Search.APIKey, cfg.Tools.Web.Search.MaxResults)
	webFetchTool := tools.NewWebFetchTool()
	toolRegistry.Register(webSearchTool)
	toolRegistry.Register(webFetchTool)

	// Create session manager
	sessionManager := session.NewSessionManager(workspace)

	// Create skills loader
	skillsLoader := skills.NewSkillsLoader(workspace, "")

	// Create context builder
	contextBuilder := agentcontext.NewContextBuilder(workspace)

	// Create memory store
	memoryStore := memory.NewMemoryStore(workspace)

	// Create subagent manager
	subagentManager := subagent.NewSubagentManager(
		provider,
		workspace,
		nil, // message bus would be passed in a real implementation
		cfg.Agents.Defaults.Model,
		cfg.Agents.Defaults.Temperature,
		cfg.Agents.Defaults.MaxTokens,
		cfg.Tools.Web.Search.APIKey,
		cfg.Tools.RestrictToWorkspace,
	)

	// Create sales-specific components
	salesKB := NewSalesKnowledgeBase(workspace)
	intentRecognizer := sales_intelligence.NewIntentRecognizer()
	emotionAnalyzer := NewEmotionAnalyzer()
	pipelineManager := NewSalesPipelineManager()

	return &SalesLoop{
		config:            cfg,
		provider:          provider,
		workspace:         workspace,
		model:             cfg.Agents.Defaults.Model,
		maxTokens:         cfg.Agents.Defaults.MaxTokens,
		temperature:       cfg.Agents.Defaults.Temperature,
		maxIterations:     cfg.Agents.Defaults.MaxToolIterations,
		memoryWindow:      cfg.Agents.Defaults.MemoryWindow,
		toolRegistry:      toolRegistry,
		sessionManager:    sessionManager,
		skillsLoader:      skillsLoader,
		contextBuilder:    contextBuilder,
		memoryStore:       memoryStore,
		subagentManager:   subagentManager,

		// Sales-specific components
		salesKB:           salesKB,
		intentRecognizer:  intentRecognizer,
		emotionAnalyzer:   emotionAnalyzer,
		pipelineManager:   pipelineManager,
		currentStage:      NewContact,
		confidenceScore:   0.0,
		lastInteraction:   time.Now(),
	}, nil
}

// SetCronService sets the cron service for the agent
func (sl *SalesLoop) SetCronService(service *cron.CronService) {
	sl.cronService = service

	// Register cron tool if cron service is available
	if service != nil {
		cronTool := tools.NewCronTool(service)
		sl.toolRegistry.Register(cronTool)
	}
}

// ProcessSalesMessage processes a sales-specific message and updates sales pipeline state
func (sl *SalesLoop) ProcessSalesMessage(message, sessionID string) (string, error) {
	// Add message to session history
	if err := sl.sessionManager.SaveMessage(sessionID, "user", message); err != nil {
		fmt.Printf("Warning: could not save message to session: %v\n", err)
	}

	// Analyze sales intent from the message
	intent := sl.intentRecognizer.RecognizeIntent(message)

	// Analyze emotional signals
	emotions := sl.emotionAnalyzer.AnalyzeEmotion(message)

	// Update pipeline based on intent and emotions
	sl.updateSalesPipeline(intent, emotions, sessionID)

	// Get recent message history
	history, err := sl.sessionManager.GetMessageHistory(sessionID, sl.memoryWindow)
	if err != nil {
		fmt.Printf("Warning: could not get message history: %v\n", err)
		history = []session.Message{}
	}

	// Build the context with history and sales-specific information
	var messages []providers.Message

	// Add system message with sales context
	systemPrompt, err := sl.buildSalesSystemPrompt(intent, emotions)
	if err != nil {
		fmt.Printf("Warning: could not build sales system prompt: %v\n", err)
		// Fallback to basic system message
		systemPrompt = fmt.Sprintf("You are a professional sales representative. Current sales stage: %s. Respond to the customer accordingly.", sl.currentStage)
	}

	messages = append(messages, providers.Message{
		Role:    "system",
		Content: systemPrompt,
	})

	// Add history if available
	for _, msg := range history {
		messages = append(messages, providers.Message{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Add the current user message
	messages = append(messages, providers.Message{
		Role:    "user",
		Content: message,
	})

	// Create the chat request
	chatReq := providers.ChatRequest{
		Messages:    messages,
		Model:       sl.model,
		Temperature: sl.temperature,
		MaxTokens:   sl.maxTokens,
	}

	ctx := context.Background()
	response, err := sl.provider.Chat(ctx, chatReq)
	if err != nil {
		return "", fmt.Errorf("error calling LLM: %w", err)
	}

	// Add assistant response to session history
	if err := sl.sessionManager.SaveMessage(sessionID, "assistant", response.Content); err != nil {
		fmt.Printf("Warning: could not save assistant message to session: %v\n", err)
	}

	// Update last interaction time
	sl.lastInteraction = time.Now()

	return response.Content, nil
}

// buildSalesSystemPrompt creates a system prompt with sales-specific context
func (sl *SalesLoop) buildSalesSystemPrompt(intent sales_intelligence.SalesIntent, emotions []EmotionSignal) (string, error) {
	var promptParts []string

	// Core sales identity
	promptParts = append(promptParts, `# SalesMate AI - Professional Sales Representative

You are SalesMate AI, an autonomous sales partner designed to help close deals. You have a Top Sales mindset and follow proven sales methodologies.`)

	// Current sales stage
	promptParts = append(promptParts, fmt.Sprintf("## Current Sales Stage: %s", formatSalesStageDescription(sl.currentStage)))

	// Sales methodology guidance
	promptParts = append(promptParts, getSalesMethodologyGuidance(sl.currentStage))

	// Detected intent
	if intent != "" {
		promptParts = append(promptParts, fmt.Sprintf("## Customer Intent Detected: %s", formatSalesIntentDescription(intent)))
	}

	// Emotional signals
	if len(emotions) > 0 {
		emotionDesc := ""
		for _, emotion := range emotions {
			emotionDesc += fmt.Sprintf("- %s (intensity: %.1f)\n", emotion.Type, emotion.Intensity)
		}
		promptParts = append(promptParts, fmt.Sprintf("## Emotional Signals:\n%s", emotionDesc))
	}

	// Confidence guidelines
	promptParts = append(promptParts, getConfidenceGuidelines(sl.confidenceScore))

	// Current date and time
	now := time.Now().Format("2006-01-02 15:04 (Monday)")
	promptParts = append(promptParts, fmt.Sprintf("## Current Time: %s", now))

	return strings.Join(promptParts, "\n\n---\n\n"), nil
}

// updateSalesPipeline updates the sales pipeline based on intent and emotions
func (sl *SalesLoop) updateSalesPipeline(intent sales_intelligence.SalesIntent, emotions []EmotionSignal, sessionID string) {
	// Update based on intent
	switch intent {
	case sales_intelligence.PricingIntent:
		if sl.currentStage == Discovery || sl.currentStage == NewContact {
			sl.currentStage = Presentation
			sl.confidenceScore += 0.15
		}
	case sales_intelligence.DemoIntent:
		if sl.currentStage == Discovery || sl.currentStage == NewContact {
			sl.currentStage = Presentation
			sl.confidenceScore += 0.2
		}
	case sales_intelligence.ObjectionIntent:
		if sl.currentStage == Negotiation || sl.currentStage == Presentation {
			// Handle objections in negotiation phase
			sl.confidenceScore -= 0.1
		}
	case sales_intelligence.BudgetIntent:
		if sl.currentStage == Negotiation || sl.currentStage == Presentation {
			sl.confidenceScore += 0.1
		}
	case sales_intelligence.AuthorityIntent:
		// Determine if the person is a decision maker
		if sl.currentStage != QualifiedLead {
			sl.currentStage = QualifiedLead
			sl.confidenceScore += 0.1
		}
	case sales_intelligence.TimelineIntent:
		if sl.currentStage == Negotiation || sl.currentStage == Close {
			sl.confidenceScore += 0.05
		}
	}

	// Update based on positive emotions
	for _, emotion := range emotions {
		switch emotion.Type {
		case "interest", "excitement":
			sl.confidenceScore += emotion.Intensity * 0.1
		case "hesitation":
			sl.confidenceScore -= emotion.Intensity * 0.05
		case "frustration":
			sl.confidenceScore -= emotion.Intensity * 0.1
		}
	}

	// Cap confidence score between 0 and 1
	if sl.confidenceScore > 1.0 {
		sl.confidenceScore = 1.0
	} else if sl.confidenceScore < 0.0 {
		sl.confidenceScore = 0.0
	}

	// Transition to next stage based on confidence and progression
	sl.transitionSalesStage(intent)
}

// transitionSalesStage handles stage transitions based on various factors
func (sl *SalesLoop) transitionSalesStage(intent sales_intelligence.SalesIntent) {
	// Define stage transition rules based on intent and confidence
	switch sl.currentStage {
	case NewContact:
		// Move to discovery if customer shows engagement
		if intent == sales_intelligence.InquiryIntent || intent == sales_intelligence.FeatureIntent {
			sl.currentStage = Discovery
		}
	case Discovery:
		// Move to presentation if customer shows interest in features or pricing
		if intent == sales_intelligence.PricingIntent || intent == sales_intelligence.DemoIntent || intent == sales_intelligence.FeatureIntent {
			sl.currentStage = Presentation
		}
	case Presentation:
		// Move to negotiation if customer is asking for specific terms or comparing
		if intent == sales_intelligence.ObjectionIntent || intent == sales_intelligence.CompetitorIntent {
			sl.currentStage = Negotiation
		} else if intent == sales_intelligence.ContractIntent {
			sl.currentStage = Close
		}
	case Negotiation:
		// Move to close if customer is ready to discuss contract terms
		if intent == sales_intelligence.ContractIntent && sl.confidenceScore > 0.7 {
			sl.currentStage = Close
		}
		// If confidence drops significantly, may need to move back or qualify differently
		if sl.confidenceScore < 0.3 {
			sl.currentStage = Discovery
		}
	case Close:
		// Stay in close until deal is finalized or lost
		if intent == sales_intelligence.ChurnRiskIntent {
			sl.currentStage = Lost
		}
	}
}

// GetCurrentStage returns the current sales stage for the session
func (sl *SalesLoop) GetCurrentStage(sessionID string) SalesStage {
	return sl.currentStage
}

// GetConfidenceScore returns the current confidence score for the sales interaction
func (sl *SalesLoop) GetConfidenceScore(sessionID string) float64 {
	return sl.confidenceScore
}

// ProcessDirect processes a single message directly without going through message bus
func (sl *SalesLoop) ProcessDirect(message, sessionID string) (string, error) {
	// For backward compatibility, route to sales processing
	return sl.ProcessSalesMessage(message, sessionID)
}

// Run starts the sales agent loop for processing messages from the message bus
func (sl *SalesLoop) Run(ctx context.Context) error {
	// For now, just return (implementation would connect to message bus)
	// In a complete implementation, this would listen for incoming messages
	return nil
}

// GetSalesKnowledgeBase returns the sales knowledge base for external use
func (sl *SalesLoop) GetSalesKnowledgeBase() *SalesKnowledgeBase {
	return sl.salesKB
}

// formatSalesStageDescription provides human-readable descriptions for sales stages
func formatSalesStageDescription(stage SalesStage) string {
	descriptions := map[SalesStage]string{
		NewContact:    "New Contact - Just made initial contact with potential customer",
		Discovery:     "Discovery - Understanding customer needs, pain points, and requirements",
		Presentation:  "Presentation - Demonstrating how our solution addresses their needs",
		Negotiation:   "Negotiation - Addressing objections, discussing terms, and finalizing details",
		Close:         "Close - Finalizing the deal and moving toward contract/signature",
		Lost:          "Lost - Deal was not successful",
		QualifiedLead: "Qualified Lead - Validated as a BANT qualified lead (Budget, Authority, Need, Timeline)",
	}

	if desc, exists := descriptions[stage]; exists {
		return desc
	}
	return string(stage) // fallback to raw value
}

// formatSalesIntentDescription provides human-readable descriptions for sales intents
func formatSalesIntentDescription(intent sales_intelligence.SalesIntent) string {
	descriptions := map[sales_intelligence.SalesIntent]string{
		sales_intelligence.InquiryIntent:     "General Inquiry - Customer is learning about our product/service",
		sales_intelligence.PricingIntent:     "Pricing Inquiry - Customer is interested in pricing information",
		sales_intelligence.DemoIntent:        "Demo/Trial Request - Customer wants to see the product in action",
		sales_intelligence.CompetitorIntent:  "Competitor Comparison - Customer is comparing with other solutions",
		sales_intelligence.ObjectionIntent:   "Objection Raised - Customer has concerns about the solution",
		sales_intelligence.BudgetIntent:      "Budget Discussion - Customer is evaluating financial aspects",
		sales_intelligence.AuthorityIntent:   "Decision Maker - Identifying who has authority to make purchasing decisions",
		sales_intelligence.TimelineIntent:    "Timeline Inquiry - Customer is asking about implementation timeline",
		sales_intelligence.FeatureIntent:     "Feature Inquiry - Customer wants details about specific capabilities",
		sales_intelligence.IntegrationIntent: "Integration Requirements - Customer needs to understand integration possibilities",
		sales_intelligence.ContractIntent:    "Contract Terms - Customer is ready to discuss final terms",
		sales_intelligence.ChurnRiskIntent:   "Churn Risk - Signs that customer may be considering leaving",
		sales_intelligence.ReferralIntent:    "Reference Request - Customer wants to see case studies or references",
	}

	if desc, exists := descriptions[intent]; exists {
		return desc
	}
	return string(intent) // fallback to raw value
}

// getSalesMethodologyGuidance provides stage-specific sales methodology guidance
func getSalesMethodologyGuidance(stage SalesStage) string {
	guidances := map[SalesStage]string{
		NewContact: `## Approach for New Contact Stage:
- Focus on building rapport and establishing trust
- Ask open-ended questions to understand their situation
- Listen more than you speak
- Don't jump to solutions too quickly`,

		Discovery: `## Approach for Discovery Stage:
- Use SPIN selling technique (Situation, Problem, Implication, Need-payoff)
- Ask about their goals, challenges, and current solutions
- Identify pain points and quantify their impact
- Understand their decision-making process`,

		Presentation: `## Approach for Presentation Stage:
- Tailor your presentation to the specific needs identified
- Use examples relevant to their industry/size
- Demonstrate ROI and business value
- Address potential concerns proactively`,

		Negotiation: `## Approach for Negotiation Stage:
- Acknowledge and address objections with empathy
- Focus on value delivered rather than just price
- Use social proof and case studies
- Guide toward next steps`,

		Close: `## Approach for Close Stage:
- Confirm mutual understanding of benefits
- Ask for commitment with assumptive language
- Handle final objections confidently
- Set expectations for next steps`,

		QualifiedLead: `## Approach for Qualified Lead Stage:
- Recognize this as a high-quality opportunity
- Validate the BANT criteria (Budget, Authority, Need, Timeline)
- Accelerate the sales process appropriately
- Minimize unnecessary discovery`,
	}

	if guidance, exists := guidances[stage]; exists {
		return guidance
	}

	return fmt.Sprintf("## Sales Stage Guidance:\nFocus on advancing the sales process to the next stage.")
}

// getConfidenceGuidelines provides guidance based on confidence score
func getConfidenceGuidelines(confidence float64) string {
	level := "Low"
	action := "Consider escalating to human assistance or asking more qualifying questions"

	if confidence >= 0.7 {
		level = "High"
		action = "Continue with sales process confidently"
	} else if confidence >= 0.5 {
		level = "Medium"
		action = "Proceed with caution, validate key assumptions"
	}

	return fmt.Sprintf("## Confidence Level: %s (%.1f)\n%s", level, confidence, action)
}