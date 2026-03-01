package skills

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Plugin represents a skill plugin with executable functionality
type Plugin struct {
	Name        string            `json:"name"`
	Path        string            `json:"path"`
	Source      string            `json:"source"`
	Description string            `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
	Executable  bool              `json:"executable"` // Whether this skill is a script that can be executed
}

// PluginManager manages plugin-based skills
type PluginManager struct {
	skillsLoader *SkillsLoader
	pluginsDir   string
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(loader *SkillsLoader, pluginsDir string) *PluginManager {
	return &PluginManager{
		skillsLoader: loader,
		pluginsDir:   pluginsDir,
	}
}

// LoadPlugin loads a skill plugin by name
func (pm *PluginManager) LoadPlugin(name string) (*Plugin, error) {
	_, err := pm.skillsLoader.LoadSkill(name)
	if err != nil {
		return nil, fmt.Errorf("plugin %s not found: %w", name, err)
	}

	// Check if the skill file is executable
	skillFile := pm.getSkillFilePath(name)
	isExecutable := pm.isExecutableSkill(skillFile)

	// Create plugin object
	plugin := &Plugin{
		Name:        name,
		Path:        skillFile,
		Source:      pm.getSkillSource(name),
		Description: pm.skillsLoader.getSkillDescription(name),
		Metadata:    pm.skillsLoader.getFullSkillMetadata(name),
		Executable:  isExecutable,
	}

	return plugin, nil
}

// ExecutePlugin executes a plugin with the given arguments
func (pm *PluginManager) ExecutePlugin(name string, args map[string]interface{}) (string, error) {
	plugin, err := pm.LoadPlugin(name)
	if err != nil {
		return "", err
	}

	if !plugin.Executable {
		return "", fmt.Errorf("plugin %s is not executable", name)
	}

	// For this implementation, we'll simulate execution by returning a message
	// In a real implementation, this would execute the skill file with the given arguments

	argsJSON, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("failed to marshal arguments: %w", err)
	}

	return fmt.Sprintf("Executing plugin %s with arguments: %s", name, string(argsJSON)), nil
}

// isExecutableSkill determines if a skill file is executable
func (pm *PluginManager) isExecutableSkill(skillPath string) bool {
	// In a real implementation, this would check if the file is executable
	// For now, we'll assume any skill file with a script-like extension is executable
	ext := filepath.Ext(skillPath)
	executableExts := map[string]bool{
		".sh": true,  // shell script
		".py": true,  // python script
		".js": true,  // javascript file
		".ts": true,  // typescript file
		".rb": true,  // ruby script
		".pl": true,  // perl script
		".php": true, // php script
	}

	return executableExts[ext]
}

// getSkillFilePath gets the path to a skill file
func (pm *PluginManager) getSkillFilePath(name string) string {
	// Try workspace first, then builtin
	workspaceSkill := filepath.Join(pm.skillsLoader.workspaceSkills, name, "SKILL.md")
	if _, err := os.Stat(workspaceSkill); err == nil {
		return workspaceSkill
	}

	builtinSkill := filepath.Join(pm.skillsLoader.builtinSkills, name, "SKILL.md")
	return builtinSkill
}

// getSkillSource gets the source of a skill (workspace or builtin)
func (pm *PluginManager) getSkillSource(name string) string {
	workspaceSkill := filepath.Join(pm.skillsLoader.workspaceSkills, name, "SKILL.md")
	if _, err := os.Stat(workspaceSkill); err == nil {
		return "workspace"
	}

	return "builtin"
}

// ListPlugins lists all available plugins
func (pm *PluginManager) ListPlugins(filterUnavailable bool) ([]Plugin, error) {
	skills, err := pm.skillsLoader.ListSkills(filterUnavailable)
	if err != nil {
		return nil, err
	}

	var plugins []Plugin
	for _, skill := range skills {
		plugin := Plugin{
			Name:        skill.Name,
			Path:        skill.Path,
			Source:      skill.Source,
			Description: skill.Description,
			Metadata:    skill.Metadata,
			Executable:  pm.isExecutableSkill(skill.Path),
		}

		// Only add if it's available (not filtered out)
		if !filterUnavailable || pm.skillsLoader.checkRequirements(skill.Metadata) {
			plugins = append(plugins, plugin)
		}
	}

	return plugins, nil
}

// InstallPluginFromURL installs a skill/plugin from a URL
func (pm *PluginManager) InstallPluginFromURL(url string, name string) error {
	// In a real implementation, this would download and install a skill from a URL
	// For this placeholder, we'll just create a basic skill file

	workspaceSkillsDir := filepath.Join(pm.skillsLoader.workspace, "skills")
	pluginDir := filepath.Join(workspaceSkillsDir, name)

	// Create the skill directory
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// Create a basic SKILL.md file
	skillContent := fmt.Sprintf(`---
name: %s
description: A skill installed from %s
---

# %s Skill

This skill was installed from: %s
`, name, url, name, url)

	skillFile := filepath.Join(pluginDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte(skillContent), 0644); err != nil {
		return fmt.Errorf("failed to create skill file: %w", err)
	}

	return nil
}

// AddPlugin adds a new plugin from content
func (pm *PluginManager) AddPlugin(name, content string) error {
	workspaceSkillsDir := filepath.Join(pm.skillsLoader.workspace, "skills")
	pluginDir := filepath.Join(workspaceSkillsDir, name)

	// Create the skill directory
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// Create the SKILL.md file with the provided content
	skillFile := filepath.Join(pluginDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to create skill file: %w", err)
	}

	return nil
}

// RemovePlugin removes a plugin
func (pm *PluginManager) RemovePlugin(name string) error {
	workspaceSkillsDir := filepath.Join(pm.skillsLoader.workspace, "skills")
	pluginDir := filepath.Join(workspaceSkillsDir, name)

	// Remove the plugin directory
	if err := os.RemoveAll(pluginDir); err != nil {
		return fmt.Errorf("failed to remove plugin: %w", err)
	}

	return nil
}

// ReloadPlugin reloads a plugin after changes
func (pm *PluginManager) ReloadPlugin(name string) error {
	// For this implementation, reloading is handled automatically by the skills loader
	// since it reads files fresh each time
	return nil
}

// UpdatePlugin updates a plugin with new content
func (pm *PluginManager) UpdatePlugin(name, newContent string) error {
	workspaceSkillsDir := filepath.Join(pm.skillsLoader.workspace, "skills")
	pluginDir := filepath.Join(workspaceSkillsDir, name)

	// Check if plugin exists
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		return fmt.Errorf("plugin %s does not exist", name)
	}

	// Update the SKILL.md file with the new content
	skillFile := filepath.Join(pluginDir, "SKILL.md")
	if err := os.WriteFile(skillFile, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to update skill file: %w", err)
	}

	return nil
}