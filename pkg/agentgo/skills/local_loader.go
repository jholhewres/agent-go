package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LocalSkills loads skills from the local filesystem
type LocalSkills struct {
	path string
}

// NewLocalSkills creates a new local skills loader
func NewLocalSkills(path string) *LocalSkills {
	// Expand ~ to home directory
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			path = filepath.Join(home, path[2:])
		}
	}

	return &LocalSkills{
		path: path,
	}
}

// Load loads skills from the filesystem
func (l *LocalSkills) Load() ([]*Skill, error) {
	// Check if path exists
	info, err := os.Stat(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			// Path doesn't exist, return empty list
			return []*Skill{}, nil
		}
		return nil, fmt.Errorf("failed to stat path: %w", err)
	}

	// If it's a single skill directory
	if l.isSingleSkill() {
		skill, err := l.loadSkill(l.path)
		if err != nil {
			return nil, err
		}
		return []*Skill{skill}, nil
	}

	// If it's a directory containing multiple skills
	if !info.IsDir() {
		return nil, fmt.Errorf("path must be a directory")
	}

	return l.loadSkillsFromDirectory()
}

// GetType returns the loader type
func (l *LocalSkills) GetType() string {
	return "local"
}

// isSingleSkill checks if the path is a single skill directory
func (l *LocalSkills) isSingleSkill() bool {
	skillFile := filepath.Join(l.path, "SKILL.md")
	_, err := os.Stat(skillFile)
	return err == nil
}

// loadSkill loads a single skill from a directory
func (l *LocalSkills) loadSkill(skillPath string) (*Skill, error) {
	// Read SKILL.md
	skillFile := filepath.Join(skillPath, "SKILL.md")
	content, err := os.ReadFile(skillFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read SKILL.md: %w", err)
	}

	// Parse SKILL.md
	skill, err := ParseSkillMD(content)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SKILL.md: %w", err)
	}

	skill.Path = skillPath

	// Load scripts if they exist
	scriptsDir := filepath.Join(skillPath, "scripts")
	if info, err := os.Stat(scriptsDir); err == nil && info.IsDir() {
		scripts, err := l.loadScripts(scriptsDir)
		if err != nil {
			return nil, fmt.Errorf("failed to load scripts: %w", err)
		}
		skill.Scripts = scripts
	}

	// Load references if they exist
	referencesDir := filepath.Join(skillPath, "references")
	if info, err := os.Stat(referencesDir); err == nil && info.IsDir() {
		references, err := l.loadReferences(referencesDir)
		if err != nil {
			return nil, fmt.Errorf("failed to load references: %w", err)
		}
		skill.References = references
	}

	return skill, nil
}

// loadSkillsFromDirectory loads multiple skills from a directory
func (l *LocalSkills) loadSkillsFromDirectory() ([]*Skill, error) {
	entries, err := os.ReadDir(l.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var skills []*Skill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(l.path, entry.Name())
		skillFile := filepath.Join(skillPath, "SKILL.md")

		// Check if SKILL.md exists
		if _, err := os.Stat(skillFile); err != nil {
			continue // Skip directories without SKILL.md
		}

		skill, err := l.loadSkill(skillPath)
		if err != nil {
			// Log warning but continue loading other skills
			fmt.Printf("Warning: failed to load skill from %s: %v\n", skillPath, err)
			continue
		}

		skills = append(skills, skill)
	}

	return skills, nil
}

// loadScripts loads all scripts from a directory
func (l *LocalSkills) loadScripts(scriptsDir string) (map[string]*Script, error) {
	scripts := make(map[string]*Script)

	err := filepath.Walk(scriptsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Read script content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read script %s: %w", path, err)
		}

		// Get relative path from scripts directory
		relPath, err := filepath.Rel(scriptsDir, path)
		if err != nil {
			relPath = filepath.Base(path)
		}

		scripts[relPath] = &Script{
			Name:    filepath.Base(path),
			Path:    relPath,
			Content: content,
			Shebang: ExtractShebang(content),
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return scripts, nil
}

// loadReferences loads all reference documents from a directory
func (l *LocalSkills) loadReferences(referencesDir string) (map[string]*Reference, error) {
	references := make(map[string]*Reference)

	err := filepath.Walk(referencesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Read reference content
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read reference %s: %w", path, err)
		}

		// Get relative path from references directory
		relPath, err := filepath.Rel(referencesDir, path)
		if err != nil {
			relPath = filepath.Base(path)
		}

		references[relPath] = &Reference{
			Name:    filepath.Base(path),
			Path:    relPath,
			Content: content,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return references, nil
}
