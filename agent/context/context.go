package context

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"salesmate/agent/memory"
	"salesmate/agent/skills"
)

// ContextBuilder builds context (system prompt + messages) for the agent
type ContextBuilder struct {
	workspace      string
	memory         *memory.MemoryStore
	skills         *skills.SkillsLoader
	bootstrapFiles []string
}

// NewContextBuilder creates a new context builder
func NewContextBuilder(workspace string) *ContextBuilder {
	return &ContextBuilder{
		workspace:      workspace,
		memory:         memory.NewMemoryStore(workspace),
		skills:         skills.NewSkillsLoader(workspace, ""),
		bootstrapFiles: []string{"AGENTS.md", "SOUL.md", "USER.md", "TOOLS.md", "IDENTITY.md"},
	}
}

// BuildSystemPrompt builds the system prompt from bootstrap files, memory, and skills
func (cb *ContextBuilder) BuildSystemPrompt(skillNames []string) (string, error) {
	var parts []string

	// Core identity
	parts = append(parts, cb.getIdentity())

	// Bootstrap files
	bootstrap, err := cb.loadBootstrapFiles()
	if err != nil {
		return "", err
	}
	if bootstrap != "" {
		parts = append(parts, bootstrap)
	}

	// Memory context
	memContext, err := cb.memory.GetMemoryContext()
	if err != nil {
		return "", err
	}
	if memContext != "" {
		parts = append(parts, fmt.Sprintf("# Memory\n\n%s", memContext))
	}

	// Skills - progressive loading
	// 1. Always-loaded skills: include full content
	alwaysSkills, err := cb.skills.GetAlwaysSkills()
	if err != nil {
		return "", err
	}
	if len(alwaysSkills) > 0 {
		alwaysContent, err := cb.skills.LoadSkillsForContext(alwaysSkills)
		if err != nil {
			return "", err
		}
		if alwaysContent != "" {
			parts = append(parts, fmt.Sprintf("# Active Skills\n\n%s", alwaysContent))
		}
	}

	// 2. Available skills: only show summary (agent uses read_file to load)
	skillsSummary, err := cb.skills.BuildSkillsSummary()
	if err != nil {
		return "", err
	}
	if skillsSummary != "" {
		parts = append(parts, fmt.Sprintf(`# Skills

The following skills extend your capabilities. To use a skill, read its SKILL.md file using the read_file tool.
Skills with available="false" need dependencies installed first - you can try installing them with apt/brew.

%s`, skillsSummary))
	}

	return strings.Join(parts, "\n\n---\n\n"), nil
}

// getIdentity gets the core identity section
func (cb *ContextBuilder) getIdentity() string {
	workspacePath := cb.workspace
	runtimeInfo := fmt.Sprintf("%s %s, Go", getOSName(), runtime.GOARCH)

	return fmt.Sprintf(`# nanobot 🐈

You are nanobot, a helpful AI assistant.

## Runtime
%s

## Workspace
Your workspace is at: %s
- Long-term memory: %s/memory/MEMORY.md
- History log: %s/memory/HISTORY.md (grep-searchable)
- Custom skills: %s/skills/{{skill-name}}/SKILL.md

Reply directly with text for conversations. Only use the 'message' tool to send to a specific chat channel.

## Tool Call Guidelines
- Before calling tools, you may briefly state your intent (e.g. "Let me check that"), but NEVER predict or describe the expected result before receiving it.
- Before modifying a file, read it first to confirm its current content.
- Do not assume a file or directory exists — use list_dir or read_file to verify.
- After writing or editing a file, re-read it if accuracy matters.
- If a tool call fails, analyze the error before retrying with a different approach.

## Memory
- Remember important facts: write to %s/memory/MEMORY.md
- Recall past events: grep %s/memory/HISTORY.md`,
		runtimeInfo,
		workspacePath,
		workspacePath,
		workspacePath,
		workspacePath,
		workspacePath,
		workspacePath)
}

// getOSName returns a human-readable OS name
func getOSName() string {
	os := runtime.GOOS
	switch os {
	case "darwin":
		return "macOS"
	default:
		return os
	}
}

// InjectRuntimeContext appends dynamic runtime context to the tail of the user message
func (cb *ContextBuilder) InjectRuntimeContext(userContent string, channel, chatID *string) string {
	now := time.Now().Format("2006-01-02 15:04 (Monday)")
	tz, _ := time.Now().Zone()

	var lines []string
	lines = append(lines, fmt.Sprintf("Current Time: %s (%s)", now, tz))

	if channel != nil && *channel != "" {
		lines = append(lines, fmt.Sprintf("Channel: %s", *channel))
	}
	if chatID != nil && *chatID != "" {
		lines = append(lines, fmt.Sprintf("Chat ID: %s", *chatID))
	}

	block := "[Runtime Context]\n" + strings.Join(lines, "\n")
	return fmt.Sprintf("%s\n\n%s", userContent, block)
}

// loadBootstrapFiles loads all bootstrap files from workspace
func (cb *ContextBuilder) loadBootstrapFiles() (string, error) {
	var parts []string

	for _, filename := range cb.bootstrapFiles {
		filePath := filepath.Join(cb.workspace, filename)

		if exists, _ := cb.fileExists(filePath); exists {
			content, err := os.ReadFile(filePath)
			if err != nil {
				return "", err
			}

			parts = append(parts, fmt.Sprintf("## %s\n\n%s", filename, string(content)))
		}
	}

	if len(parts) == 0 {
		return "", nil
	}

	return strings.Join(parts, "\n\n"), nil
}

// fileExists checks if a file exists
func (cb *ContextBuilder) fileExists(filename string) (bool, error) {
	_, err := os.Stat(filename)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
