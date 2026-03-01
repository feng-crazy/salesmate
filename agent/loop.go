package agent

import (
	"context"

	"fmt"

	agentcontext "salesmate/agent/context"
	"salesmate/agent/memory"
	"salesmate/agent/skills"
	"salesmate/agent/subagent"
	"salesmate/agent/tools"
	"salesmate/config"
	"salesmate/cron"
	"salesmate/providers"
	"salesmate/session"
)

// AgentLoop represents the core processing engine for the AI agent
type AgentLoop struct {
	config          *config.Config
	provider        providers.LLMProvider
	workspace       string
	model           string
	maxTokens       int
	temperature     float64
	maxIterations   int
	memoryWindow    int
	toolRegistry    *tools.ToolRegistry
	sessionManager  *session.SessionManager
	cronService     *cron.CronService
	skillsLoader    *skills.SkillsLoader
	contextBuilder  *agentcontext.ContextBuilder
	memoryStore     *memory.MemoryStore
	subagentManager *subagent.SubagentManager
}

// NewAgentLoop creates a new agent loop with the given configuration
func NewAgentLoop(cfg *config.Config) (*AgentLoop, error) {
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

	return &AgentLoop{
		config:          cfg,
		provider:        provider,
		workspace:       workspace,
		model:           cfg.Agents.Defaults.Model,
		maxTokens:       cfg.Agents.Defaults.MaxTokens,
		temperature:     cfg.Agents.Defaults.Temperature,
		maxIterations:   cfg.Agents.Defaults.MaxToolIterations,
		memoryWindow:    cfg.Agents.Defaults.MemoryWindow,
		toolRegistry:    toolRegistry,
		sessionManager:  sessionManager,
		skillsLoader:    skillsLoader,
		contextBuilder:  contextBuilder,
		memoryStore:     memoryStore,
		subagentManager: subagentManager,
	}, nil
}

// SetCronService sets the cron service for the agent
func (al *AgentLoop) SetCronService(service *cron.CronService) {
	al.cronService = service

	// Register cron tool if cron service is available
	if service != nil {
		cronTool := tools.NewCronTool(service)
		al.toolRegistry.Register(cronTool)
	}
}

// ProcessDirect processes a single message directly without going through message bus
func (al *AgentLoop) ProcessDirect(message, sessionID string) (string, error) {
	// Add message to session history
	if err := al.sessionManager.SaveMessage(sessionID, "user", message); err != nil {
		// Just log the error, don't fail the whole operation
		fmt.Printf("Warning: could not save message to session: %v\n", err)
	}

	// Get recent message history
	history, err := al.sessionManager.GetMessageHistory(sessionID, al.memoryWindow)
	if err != nil {
		// Just log the error, continue without history
		fmt.Printf("Warning: could not get message history: %v\n", err)
		history = []session.Message{}
	}

	// Build the context with history
	var messages []providers.Message

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
		Model:       al.model,
		Temperature: al.temperature,
		MaxTokens:   al.maxTokens,
	}

	ctx := context.Background()
	response, err := al.provider.Chat(ctx, chatReq)
	if err != nil {
		return "", fmt.Errorf("error calling LLM: %w", err)
	}

	// Add assistant response to session history
	if err := al.sessionManager.SaveMessage(sessionID, "assistant", response.Content); err != nil {
		fmt.Printf("Warning: could not save assistant message to session: %v\n", err)
	}

	return response.Content, nil
}

// Run starts the agent loop for processing messages from the message bus
func (al *AgentLoop) Run(ctx context.Context) error {
	// For now, just return (implementation would connect to message bus)
	// In a complete implementation, this would listen for incoming messages
	return nil
}

// Stop stops the agent loop
func (al *AgentLoop) Stop() {
	// Cleanup resources
}
