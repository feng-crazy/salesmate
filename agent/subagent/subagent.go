package subagent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"salesmate/agent/tools"
	"salesmate/bus"
	"salesmate/providers"
)

// SubagentManager manages background subagent execution
type SubagentManager struct {
	provider                providers.LLMProvider
	workspace               string
	bus                     *bus.MessageBus
	model                   string
	temperature             float64
	maxTokens               int
	braveAPIKey             string
	restrictToWorkspace     bool
	runningTasks            map[string]*SubagentTask
	runningTasksMu          sync.RWMutex
	onTaskCompletedCallback func(taskID, label, result string)
	taskDependencies        map[string][]string // Maps task ID to its dependencies
	dependencyWaiters       map[string][]string // Maps dependency ID to tasks waiting for it
}

// SubagentTask represents a running subagent task
type SubagentTask struct {
	ID           string
	Label        string
	Task         string
	Context      context.Context
	Cancel       context.CancelFunc
	Status       TaskStatus
	CreatedAt    time.Time
	Dependencies []string
}

// TaskStatus represents the status of a task
type TaskStatus string

const (
	TaskPending   TaskStatus = "pending"
	TaskRunning   TaskStatus = "running"
	TaskCompleted TaskStatus = "completed"
	TaskFailed    TaskStatus = "failed"
)

// NewSubagentManager creates a new subagent manager
func NewSubagentManager(
	provider providers.LLMProvider,
	workspace string,
	bus *bus.MessageBus,
	model string,
	temperature float64,
	maxTokens int,
	braveAPIKey string,
	restrictToWorkspace bool,
) *SubagentManager {
	return &SubagentManager{
		provider:            provider,
		workspace:           workspace,
		bus:                 bus,
		model:               model,
		temperature:         temperature,
		maxTokens:           maxTokens,
		braveAPIKey:         braveAPIKey,
		restrictToWorkspace: restrictToWorkspace,
		runningTasks:        make(map[string]*SubagentTask),
		taskDependencies:    make(map[string][]string),
		dependencyWaiters:   make(map[string][]string),
	}
}

// Spawn spawns a subagent to execute a task in the background
func (sm *SubagentManager) Spawn(
	task string,
	label *string,
	originChannel string,
	originChatID string,
	dependencies ...string,
) (string, error) {
	taskID := sm.generateTaskID()
	displayLabel := task[:min(len(task), 30)]
	if len(task) > 30 {
		displayLabel += "..."
	}
	if label != nil {
		displayLabel = *label
	}

	// Create context for the task
	ctx, cancel := context.WithCancel(context.Background())

	// Check if dependencies exist
	for _, depID := range dependencies {
		if !sm.taskExists(depID) {
			return "", fmt.Errorf("dependency task %s does not exist", depID)
		}
	}

	// Store task info
	subagentTask := &SubagentTask{
		ID:           taskID,
		Label:        displayLabel,
		Task:         task,
		Context:      ctx,
		Cancel:       cancel,
		Status:       TaskPending,
		CreatedAt:    time.Now(),
		Dependencies: dependencies,
	}

	// Check if dependencies are met
	if len(dependencies) > 0 {
		if !sm.areDependenciesMet(dependencies) {
			// Wait for dependencies to complete
			go sm.waitForDependencies(subagentTask)
			return fmt.Sprintf("Subagent [%s] scheduled (id: %s). Waiting for dependencies to complete before starting.", displayLabel, taskID), nil
		}
	}

	// Update status and run the task
	subagentTask.Status = TaskRunning
	sm.runningTasksMu.Lock()
	sm.runningTasks[taskID] = subagentTask
	sm.runningTasksMu.Unlock()

	// Run the subagent in a goroutine
	go func() {
		defer func() {
			// Remove from running tasks when done
			sm.runningTasksMu.Lock()
			delete(sm.runningTasks, taskID)
			sm.runningTasksMu.Unlock()

			// Notify tasks that were waiting for this task to complete
			sm.notifyWaiters(taskID)
		}()

		result, err := sm.runSubagent(taskID, task, displayLabel, originChannel, originChatID)
		if err != nil {
			log.Printf("Subagent [%s] failed: %v", taskID, err)
			subagentTask.Status = TaskFailed
			result = fmt.Sprintf("Error: %v", err)
		} else {
			subagentTask.Status = TaskCompleted
		}

		// Announce result
		sm.announceResult(taskID, displayLabel, task, result, originChannel, originChatID)
	}()

	log.Printf("Spawned subagent [%s]: %s", taskID, displayLabel)
	return fmt.Sprintf("Subagent [%s] started (id: %s). I'll notify you when it completes.", displayLabel, taskID), nil
}

// waitForDependencies waits for dependencies to complete before starting the task
func (sm *SubagentManager) waitForDependencies(task *SubagentTask) {
	for {
		select {
		case <-task.Context.Done():
			// Task was cancelled while waiting
			return
		default:
			if sm.areDependenciesMet(task.Dependencies) {
				// Dependencies are met, start the task
				task.Status = TaskRunning
				go func() {
					result, err := sm.runSubagent(task.ID, task.Task, task.Label, "cli", "direct") // Use default channel/chat for dependency tasks
					if err != nil {
						log.Printf("Subagent [%s] failed: %v", task.ID, err)
						task.Status = TaskFailed
						result = fmt.Sprintf("Error: %v", err)
					} else {
						task.Status = TaskCompleted
					}

					// Announce result
					sm.announceResult(task.ID, task.Label, task.Task, result, "cli", "direct")
				}()
				return
			}
			time.Sleep(1 * time.Second) // Wait before checking again
		}
	}
}

// areDependenciesMet checks if all dependencies for a task are completed
func (sm *SubagentManager) areDependenciesMet(dependencies []string) bool {
	sm.runningTasksMu.RLock()
	defer sm.runningTasksMu.RUnlock()

	for _, depID := range dependencies {
		task, exists := sm.runningTasks[depID]
		if !exists || task.Status != TaskCompleted {
			return false
		}
	}
	return true
}

// taskExists checks if a task exists
func (sm *SubagentManager) taskExists(taskID string) bool {
	sm.runningTasksMu.RLock()
	defer sm.runningTasksMu.RUnlock()

	_, exists := sm.runningTasks[taskID]
	return exists
}

// notifyWaiters notifies tasks that were waiting for a task to complete
func (sm *SubagentManager) notifyWaiters(taskID string) {
	// In a real implementation, this would check which tasks were waiting for this taskID
	// and update their states accordingly
}

// runSubagent executes the subagent task and returns the result
func (sm *SubagentManager) runSubagent(
	taskID string,
	task string,
	label string,
	originChannel string,
	originChatID string,
) (string, error) {
	log.Printf("Subagent [%s] starting task: %s", taskID, label)

	// Build subagent tools (no message tool, no spawn tool)
	toolRegistry := tools.NewToolRegistry()
	allowedDir := ""
	if sm.restrictToWorkspace {
		allowedDir = sm.workspace
	}

	toolRegistry.Register(tools.NewReadFileTool(sm.workspace, allowedDir))
	toolRegistry.Register(tools.NewWriteFileTool(sm.workspace, allowedDir))
	toolRegistry.Register(tools.NewEditFileTool(sm.workspace, allowedDir))
	toolRegistry.Register(tools.NewListDirTool(sm.workspace, allowedDir))
	toolRegistry.Register(tools.NewExecTool(sm.workspace, 60, sm.restrictToWorkspace)) // 60s timeout default
	toolRegistry.Register(tools.NewWebSearchTool(sm.braveAPIKey, 5))                   // 5 results max
	toolRegistry.Register(tools.NewWebFetchTool())

	// Build messages with subagent-specific prompt
	systemPrompt := sm.buildSubagentPrompt(task)
	messages := []providers.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: task},
	}

	// Run agent loop (limited iterations)
	maxIterations := 15
	iteration := 0
	var finalResult string

	for iteration < maxIterations {
		iteration++

		// Prepare tool definitions for the provider
		toolDefs := sm.getToolDefinitions(toolRegistry)

		response, err := sm.provider.Chat(context.Background(), providers.ChatRequest{
			Messages:    messages,
			Model:       sm.model,
			Temperature: sm.temperature,
			MaxTokens:   sm.maxTokens,
			Tools:       toolDefs,
		})
		if err != nil {
			return "", err
		}

		if len(response.ToolCalls) > 0 {
			// Add assistant message with tool calls
			// Note: The providers.Message type doesn't have ToolCalls field, so we'll handle it differently
			// In a real implementation, we may need to extend the Message type or handle differently
			for _, tc := range response.ToolCalls {
				argsBytes, _ := json.Marshal(tc.Args)

				messages = append(messages, providers.Message{
					Role:    "assistant",
					Content: fmt.Sprintf("Calling tool: %s", tc.Name),
				})

				log.Printf("Subagent [%s] executing: %s with arguments: %s", taskID, tc.Name, string(argsBytes))

				result, err := toolRegistry.Execute(tc.Name, tc.Args)
				if err != nil {
					return "", fmt.Errorf("tool execution failed: %w", err)
				}

				messages = append(messages, providers.Message{
					Role:    "tool",
					Content: result,
					Name:    tc.Name,
				})
			}
		} else {
			finalResult = response.Content
			break
		}
	}

	if finalResult == "" {
		finalResult = "Task completed but no final response was generated."
	}

	log.Printf("Subagent [%s] completed successfully", taskID)
	return finalResult, nil
}

// announceResult announces the subagent result to the main agent via the message bus
func (sm *SubagentManager) announceResult(
	taskID string,
	label string,
	task string,
	result string,
	originChannel string,
	originChatID string,
) {
	_ = originChannel // use variables to avoid "not used" error
	_ = originChatID
	_ = task

	// For now, we'll just log the result. In a real implementation, we'd publish to the bus
	log.Printf("Subagent [%s] result announced: %s", taskID, result)

	// If there's a callback, call it
	if sm.onTaskCompletedCallback != nil {
		sm.onTaskCompletedCallback(taskID, label, result)
	}
}

// buildSubagentPrompt builds a focused system prompt for the subagent
func (sm *SubagentManager) buildSubagentPrompt(task string) string {
	now := fmt.Sprintf("%s (%s)", time.Now().Format("2006-01-02 15:04"), time.Now().Weekday().String())

	return fmt.Sprintf(`# Subagent

## Current Time
%s

You are a subagent spawned by the main agent to complete a specific task.

## Rules
1. Stay focused - complete only the assigned task, nothing else
2. Your final response will be reported back to the main agent
3. Do not initiate conversations or take on side tasks
4. Be concise but informative in your findings

## What You Can Do
- Read and write files in the workspace
- Execute shell commands
- Search the web and fetch web pages
- Complete the task thoroughly

## What You Cannot Do
- Send messages directly to users (no message tool available)
- Spawn other subagents
- Access the main agent's conversation history

## Workspace
Your workspace is at: %s
Skills are available at: %s/skills/ (read SKILL.md files as needed)

When you have completed the task, provide a clear summary of your findings or actions.`,
		now, sm.workspace, sm.workspace)
}

// getToolDefinitions converts the tool registry to provider tool definitions
func (sm *SubagentManager) getToolDefinitions(toolRegistry *tools.ToolRegistry) []providers.ToolDef {
	definitions := toolRegistry.GetDefinitions()

	var toolDefs []providers.ToolDef
	for _, def := range definitions {
		// Type assertion to convert to providers.ToolDef
		if defMap, ok := def.(map[string]interface{}); ok {
			// This is a simplified conversion - in a real implementation, you'd have
			// a more robust conversion
			if funcDef, exists := defMap["function"].(map[string]interface{}); exists {
				name, _ := funcDef["name"].(string)
				description, _ := funcDef["description"].(string)

				toolDef := providers.ToolDef{
					Type: "function",
					Function: providers.FunctionDef{
						Name:        name,
						Description: description,
						Parameters:  funcDef["parameters"].(map[string]interface{}),
					},
				}
				toolDefs = append(toolDefs, toolDef)
			}
		}
	}

	return toolDefs
}

// generateTaskID generates a simple task ID
func (sm *SubagentManager) generateTaskID() string {
	// In a real implementation, you'd want a proper UUID
	// For now, we'll use a timestamp-based ID
	return fmt.Sprintf("%d", time.Now().Unix())
}

// GetRunningCount returns the number of currently running subagents
func (sm *SubagentManager) GetRunningCount() int {
	sm.runningTasksMu.RLock()
	defer sm.runningTasksMu.RUnlock()
	return len(sm.runningTasks)
}

// SetOnTaskCompletedCallback sets a callback to be called when a task completes
func (sm *SubagentManager) SetOnTaskCompletedCallback(callback func(taskID, label, result string)) {
	sm.onTaskCompletedCallback = callback
}

// GetTaskStatus returns the status of a specific task
func (sm *SubagentManager) GetTaskStatus(taskID string) (TaskStatus, bool) {
	sm.runningTasksMu.RLock()
	defer sm.runningTasksMu.RUnlock()

	task, exists := sm.runningTasks[taskID]
	if !exists {
		return "", false
	}

	return task.Status, true
}

// GetRunningTasks returns a list of currently running task IDs
func (sm *SubagentManager) GetRunningTasks() []string {
	sm.runningTasksMu.RLock()
	defer sm.runningTasksMu.RUnlock()

	var tasks []string
	for id, task := range sm.runningTasks {
		if task.Status == TaskRunning || task.Status == TaskPending {
			tasks = append(tasks, id)
		}
	}

	return tasks
}

// CancelTask cancels a running task
func (sm *SubagentManager) CancelTask(taskID string) error {
	sm.runningTasksMu.Lock()
	defer sm.runningTasksMu.Unlock()

	task, exists := sm.runningTasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	task.Cancel()
	task.Status = TaskFailed
	return nil
}

// min is a helper function to get the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
