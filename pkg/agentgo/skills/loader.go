package skills

// SkillLoader is the interface for loading skills from different sources
type SkillLoader interface {
	// Load loads skills from the source
	Load() ([]*Skill, error)

	// GetType returns the loader type ("local", "database", etc)
	GetType() string
}
