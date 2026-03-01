package skills

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Skill represents a skill with its metadata
type Skill struct {
	Name        string            `json:"name"`
	Path        string            `json:"path"`
	Source      string            `json:"source"`
	Description string            `json:"description"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// SkillsLoader manages agent skills
type SkillsLoader struct {
	workspace       string
	workspaceSkills string
	builtinSkills   string
}

// NewSkillsLoader creates a new skills loader
func NewSkillsLoader(workspace string, builtinSkillsDir string) *SkillsLoader {
	workspaceSkills := filepath.Join(workspace, "skills")

	// If builtinSkillsDir is empty, use default location
	if builtinSkillsDir == "" {
		// For now, we'll use a relative path from the binary location
		// In a real implementation, this would be determined differently
		builtinSkillsDir = filepath.Join(filepath.Dir(os.Args[0]), "..", "skills")
	}

	return &SkillsLoader{
		workspace:       workspace,
		workspaceSkills: workspaceSkills,
		builtinSkills:   builtinSkillsDir,
	}
}

// ListSkills lists all available skills
func (sl *SkillsLoader) ListSkills(filterUnavailable bool) ([]Skill, error) {
	var skills []Skill

	// Workspace skills (highest priority)
	if exists, _ := sl.dirExists(sl.workspaceSkills); exists {
		workspaceSkillDirs, err := ioutil.ReadDir(sl.workspaceSkills)
		if err == nil {
			for _, skillDir := range workspaceSkillDirs {
				if skillDir.IsDir() {
					skillFile := filepath.Join(sl.workspaceSkills, skillDir.Name(), "SKILL.md")
					if exists, _ := sl.fileExists(skillFile); exists {
						meta := sl.getSkillMetadata(skillDir.Name())
						if !filterUnavailable || sl.checkRequirements(meta) {
							skills = append(skills, Skill{
								Name:        skillDir.Name(),
								Path:        skillFile,
								Source:      "workspace",
								Description: sl.getSkillDescription(skillDir.Name()),
								Metadata:    meta,
							})
						}
					}
				}
			}
		}
	}

	// Built-in skills
	if exists, _ := sl.dirExists(sl.builtinSkills); exists {
		builtinSkillDirs, err := ioutil.ReadDir(sl.builtinSkills)
		if err == nil {
			for _, skillDir := range builtinSkillDirs {
				if skillDir.IsDir() {
					skillFile := filepath.Join(sl.builtinSkills, skillDir.Name(), "SKILL.md")
					if exists, _ := sl.fileExists(skillFile); exists {
						// Check if this skill is already in the list (from workspace)
						existsInList := false
						for _, existingSkill := range skills {
							if existingSkill.Name == skillDir.Name() {
								existsInList = true
								break
							}
						}

						if !existsInList {
							meta := sl.getSkillMetadata(skillDir.Name())
							if !filterUnavailable || sl.checkRequirements(meta) {
								skills = append(skills, Skill{
									Name:        skillDir.Name(),
									Path:        skillFile,
									Source:      "builtin",
									Description: sl.getSkillDescription(skillDir.Name()),
									Metadata:    meta,
								})
							}
						}
					}
				}
			}
		}
	}

	return skills, nil
}

// LoadSkill loads a skill by name
func (sl *SkillsLoader) LoadSkill(name string) (string, error) {
	// Check workspace first
	workspaceSkill := filepath.Join(sl.workspaceSkills, name, "SKILL.md")
	if exists, _ := sl.fileExists(workspaceSkill); exists {
		content, err := ioutil.ReadFile(workspaceSkill)
		if err != nil {
			return "", err
		}
		return sl.stripFrontmatter(string(content)), nil
	}

	// Check built-in
	builtinSkill := filepath.Join(sl.builtinSkills, name, "SKILL.md")
	if exists, _ := sl.fileExists(builtinSkill); exists {
		content, err := ioutil.ReadFile(builtinSkill)
		if err != nil {
			return "", err
		}
		return sl.stripFrontmatter(string(content)), nil
	}

	return "", fmt.Errorf("skill '%s' not found", name)
}

// LoadSkillsForContext loads specific skills for inclusion in agent context
func (sl *SkillsLoader) LoadSkillsForContext(skillNames []string) (string, error) {
	var parts []string

	for _, name := range skillNames {
		content, err := sl.LoadSkill(name)
		if err != nil {
			continue // Skip if not found
		}

		parts = append(parts, fmt.Sprintf("### Skill: %s\n\n%s", name, content))
	}

	if len(parts) == 0 {
		return "", nil
	}

	return strings.Join(parts, "\n\n---\n\n"), nil
}

// BuildSkillsSummary builds a summary of all skills in XML format
func (sl *SkillsLoader) BuildSkillsSummary() (string, error) {
	allSkills, err := sl.ListSkills(false) // Don't filter unavailable for summary
	if err != nil {
		return "", err
	}

	if len(allSkills) == 0 {
		return "", nil
	}

	var lines []string
	lines = append(lines, "<skills>")

	for _, s := range allSkills {
		name := sl.escapeXML(s.Name)
		path := s.Path
		desc := sl.escapeXML(s.Description)
		available := sl.checkRequirements(s.Metadata)

		lines = append(lines, fmt.Sprintf("  <skill available=\"%s\">", fmt.Sprintf("%t", available)))
		lines = append(lines, fmt.Sprintf("    <name>%s</name>", name))
		lines = append(lines, fmt.Sprintf("    <description>%s</description>", desc))
		lines = append(lines, fmt.Sprintf("    <location>%s</location>", path))

		// Show missing requirements for unavailable skills
		if !available {
			missing := sl.getMissingRequirements(s.Metadata)
			if missing != "" {
				lines = append(lines, fmt.Sprintf("    <requires>%s</requires>", sl.escapeXML(missing)))
			}
		}

		lines = append(lines, "  </skill>")
	}

	lines = append(lines, "</skills>")

	return strings.Join(lines, "\n"), nil
}

// fileExists checks if a file exists
func (sl *SkillsLoader) fileExists(filename string) (bool, error) {
	_, err := os.Stat(filename)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// dirExists checks if a directory exists
func (sl *SkillsLoader) dirExists(dirname string) (bool, error) {
	info, err := os.Stat(dirname)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}

// getMissingRequirements gets a description of missing requirements
func (sl *SkillsLoader) getMissingRequirements(skillMeta map[string]interface{}) string {
	var missing []string

	if requires, ok := skillMeta["requires"].(map[string]interface{}); ok {
		// Check bins
		if bins, ok := requires["bins"].([]interface{}); ok {
			for _, bin := range bins {
				binStr := bin.(string)
				if _, err := exec.LookPath(binStr); err != nil {
					missing = append(missing, fmt.Sprintf("CLI: %s", binStr))
				}
			}
		}

		// Check environment variables
		if envVars, ok := requires["env"].([]interface{}); ok {
			for _, env := range envVars {
				envStr := env.(string)
				if os.Getenv(envStr) == "" {
					missing = append(missing, fmt.Sprintf("ENV: %s", envStr))
				}
			}
		}
	}

	return strings.Join(missing, ", ")
}

// getSkillDescription gets the description of a skill from its metadata
func (sl *SkillsLoader) getSkillDescription(name string) string {
	meta := sl.getSkillMetadata(name)
	if desc, ok := meta["description"].(string); ok && desc != "" {
		return desc
	}
	return name // Fallback to skill name
}

// stripFrontmatter removes YAML frontmatter from markdown content
func (sl *SkillsLoader) stripFrontmatter(content string) string {
	if strings.HasPrefix(content, "---") {
		re := regexp.MustCompile(`(?s)^---\n.*?\n---\n`)
		return strings.TrimSpace(re.ReplaceAllString(content, ""))
	}
	return content
}

// parseNanobotMetadata parses skill metadata JSON from frontmatter
func (sl *SkillsLoader) parseNanobotMetadata(raw string) map[string]interface{} {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return make(map[string]interface{})
	}

	// Extract nanobot or openclaw metadata
	if nanobotData, ok := data["nanobot"]; ok {
		if nanobotMap, ok := nanobotData.(map[string]interface{}); ok {
			return nanobotMap
		}
	}

	if openclawData, ok := data["openclaw"]; ok {
		if openclawMap, ok := openclawData.(map[string]interface{}); ok {
			return openclawMap
		}
	}

	return data
}

// checkRequirements checks if skill requirements are met
func (sl *SkillsLoader) checkRequirements(skillMeta map[string]interface{}) bool {
	if requires, ok := skillMeta["requires"].(map[string]interface{}); ok {
		// Check bins
		if bins, ok := requires["bins"].([]interface{}); ok {
			for _, bin := range bins {
				binStr := bin.(string)
				if _, err := exec.LookPath(binStr); err != nil {
					return false
				}
			}
		}

		// Check environment variables
		if envVars, ok := requires["env"].([]interface{}); ok {
			for _, env := range envVars {
				envStr := env.(string)
				if os.Getenv(envStr) == "" {
					return false
				}
			}
		}
	}

	return true
}

// getSkillMetadata gets nanobot metadata for a skill
func (sl *SkillsLoader) getSkillMetadata(name string) map[string]interface{} {
	meta := sl.getFullSkillMetadata(name)
	if metadataStr, ok := meta["metadata"].(string); ok {
		return sl.parseNanobotMetadata(metadataStr)
	}
	return make(map[string]interface{})
}

// getFullSkillMetadata gets the full metadata from a skill's frontmatter
func (sl *SkillsLoader) getFullSkillMetadata(name string) map[string]interface{} {
	content, err := sl.LoadSkill(name)
	if err != nil {
		// Try loading the raw skill file to get metadata
		workspaceSkill := filepath.Join(sl.workspaceSkills, name, "SKILL.md")
		var contentBytes []byte

		if exists, _ := sl.fileExists(workspaceSkill); exists {
			contentBytes, err = ioutil.ReadFile(workspaceSkill)
		} else {
			builtinSkill := filepath.Join(sl.builtinSkills, name, "SKILL.md")
			if exists, _ := sl.fileExists(builtinSkill); exists {
				contentBytes, err = ioutil.ReadFile(builtinSkill)
			}
		}

		if err != nil {
			return make(map[string]interface{})
		}

		content = string(contentBytes)
	}

	// Extract frontmatter
	if strings.HasPrefix(content, "---") {
		re := regexp.MustCompile(`(?s)^---\n(.*?)\n---`)
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			// Parse the YAML-like frontmatter
			metadata := make(map[string]interface{})
			frontmatter := matches[1]

			lines := strings.Split(frontmatter, "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if idx := strings.Index(line, ":"); idx > 0 {
					key := strings.TrimSpace(line[:idx])
					value := strings.TrimSpace(line[idx+1:])
					// Remove quotes if present
					if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
					   (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
						value = value[1 : len(value)-1]
					}
					metadata[key] = value
				}
			}

			return metadata
		}
	}

	return make(map[string]interface{})
}

// escapeXML escapes XML special characters
func (sl *SkillsLoader) escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	return s
}

// GetAlwaysSkills gets skills marked as always=true that meet requirements
func (sl *SkillsLoader) GetAlwaysSkills() ([]string, error) {
	var result []string

	allSkills, err := sl.ListSkills(true) // Filter unavailable
	if err != nil {
		return nil, err
	}

	for _, s := range allSkills {
		fullMeta := sl.getFullSkillMetadata(s.Name)

		// Check if "always" is set in the metadata
		if always, ok := fullMeta["always"].(bool); ok && always {
			result = append(result, s.Name)
		} else if alwaysStr, ok := fullMeta["always"].(string); ok && alwaysStr == "true" {
			result = append(result, s.Name)
		} else {
			// Also check in nanobot metadata section
			nanobotMeta := sl.parseNanobotMetadata(fmt.Sprintf("{\"nanobot\": %v}", s.Metadata))
			if always, ok := nanobotMeta["always"].(bool); ok && always {
				result = append(result, s.Name)
			} else if alwaysStr, ok := nanobotMeta["always"].(string); ok && alwaysStr == "true" {
				result = append(result, s.Name)
			}
		}
	}

	return result, nil
}