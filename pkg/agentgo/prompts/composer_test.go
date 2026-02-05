package prompts

import (
	"strings"
	"testing"
)

// TestNewPromptComposer tests creating a new prompt composer
func TestNewPromptComposer(t *testing.T) {
	composer := NewPromptComposer()

	if composer == nil {
		t.Fatal("expected non-nil composer")
	}

	if composer.SectionCount() != 0 {
		t.Errorf("expected empty composer, got %d sections", composer.SectionCount())
	}
}

// TestPromptComposerWithSections tests creating composer with initial sections
func TestPromptComposerWithSections(t *testing.T) {
	sections := []PromptSection{
		{Name: "section1", Content: "content1", Priority: 1, Enabled: true},
		{Name: "section2", Content: "content2", Priority: 2, Enabled: true},
	}

	composer := NewPromptComposer(sections...)

	if composer.SectionCount() != 2 {
		t.Errorf("expected 2 sections, got %d", composer.SectionCount())
	}
}

// TestAddSection tests adding sections to the composer
func TestAddSection(t *testing.T) {
	composer := NewPromptComposer()

	section := PromptSection{
		Name:     "test",
		Content:  "test content",
		Priority: 1,
		Enabled:  true,
	}

	composer.AddSection(section)

	if composer.SectionCount() != 1 {
		t.Errorf("expected 1 section, got %d", composer.SectionCount())
	}

	retrieved, ok := composer.GetSection("test")
	if !ok {
		t.Error("section not found")
	}

	if retrieved.Name != "test" {
		t.Errorf("expected name 'test', got '%s'", retrieved.Name)
	}
}

// TestRemoveSection tests removing sections
func TestRemoveSection(t *testing.T) {
	sections := []PromptSection{
		{Name: "section1", Content: "content1", Priority: 1, Enabled: true},
		{Name: "section2", Content: "content2", Priority: 2, Enabled: true},
	}

	composer := NewPromptComposer(sections...)
	composer.RemoveSection("section1")

	if composer.SectionCount() != 1 {
		t.Errorf("expected 1 section after removal, got %d", composer.SectionCount())
	}

	_, ok := composer.GetSection("section1")
	if ok {
		t.Error("removed section still found")
	}
}

// TestEnableDisableSection tests enabling/disabling sections
func TestEnableDisableSection(t *testing.T) {
	sections := []PromptSection{
		{Name: "section1", Content: "content1", Priority: 1, Enabled: true},
		{Name: "section2", Content: "content2", Priority: 2, Enabled: true},
	}

	composer := NewPromptComposer(sections...)

	// Disable section1
	if !composer.DisableSection("section1") {
		t.Error("failed to disable section")
	}

	section, _ := composer.GetSection("section1")
	if section.Enabled {
		t.Error("section should be disabled")
	}

	// Enable section1
	if !composer.EnableSection("section1") {
		t.Error("failed to enable section")
	}

	section, _ = composer.GetSection("section1")
	if !section.Enabled {
		t.Error("section should be enabled")
	}
}

// TestCompose tests basic composition
func TestCompose(t *testing.T) {
	sections := []PromptSection{
		{Name: "section1", Content: "content1", Priority: 1, Enabled: true},
		{Name: "section2", Content: "content2", Priority: 2, Enabled: true},
		{Name: "section3", Content: "content3", Priority: 3, Enabled: false},
	}

	composer := NewPromptComposer(sections...)

	result, err := composer.Compose()
	if err != nil {
		t.Fatalf("compose failed: %v", err)
	}

	// Should only contain enabled sections
	if !strings.Contains(result, "content1") {
		t.Error("result should contain content1")
	}
	if !strings.Contains(result, "content2") {
		t.Error("result should contain content2")
	}
	if strings.Contains(result, "content3") {
		t.Error("result should not contain disabled content3")
	}

	// Check separator
	if !strings.Contains(result, "\n\n") {
		t.Error("result should have double newline separator")
	}
}

// TestComposePriority tests that sections are ordered by priority
func TestComposePriority(t *testing.T) {
	sections := []PromptSection{
		{Name: "low", Content: "low priority", Priority: 100, Enabled: true},
		{Name: "high", Content: "high priority", Priority: 1, Enabled: true},
		{Name: "mid", Content: "mid priority", Priority: 50, Enabled: true},
	}

	composer := NewPromptComposer(sections...)

	result, err := composer.Compose()
	if err != nil {
		t.Fatalf("compose failed: %v", err)
	}

	// Find positions
	highPos := strings.Index(result, "high priority")
	midPos := strings.Index(result, "mid priority")
	lowPos := strings.Index(result, "low priority")

	if highPos >= midPos || midPos >= lowPos {
		t.Error("sections not ordered by priority")
	}
}

// TestComposeTemplate tests template rendering in sections
func TestComposeTemplate(t *testing.T) {
	sections := []PromptSection{
		{
			Name:       "template",
			Content:    "Hello {{.Name}}!",
			Priority:   1,
			Enabled:    true,
			IsTemplate: true,
			Variables:  map[string]interface{}{"Name": "World"},
		},
	}

	composer := NewPromptComposer(sections...)

	result, err := composer.Compose()
	if err != nil {
		t.Fatalf("compose failed: %v", err)
	}

	expected := "Hello World!"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

// TestComposeWithVars tests global variable injection
func TestComposeWithVars(t *testing.T) {
	sections := []PromptSection{
		{
			Name:       "template",
			Content:    "Hello {{.Name}}, you are {{.Age}} years old",
			Priority:   1,
			Enabled:    true,
			IsTemplate: true,
			Variables:  map[string]interface{}{"Name": "Alice"},
		},
	}

	composer := NewPromptComposer(sections...)

	globalVars := map[string]interface{}{"Age": 30}
	result, err := composer.ComposeWithVars(globalVars)
	if err != nil {
		t.Fatalf("compose failed: %v", err)
	}

	expected := "Hello Alice, you are 30 years old"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}

// TestSetSectionVariables tests updating section variables
func TestSetSectionVariables(t *testing.T) {
	section := PromptSection{
		Name:       "template",
		Content:    "Value: {{.X}}",
		Priority:   1,
		Enabled:    true,
		IsTemplate: true,
		Variables:  map[string]interface{}{"X": 1},
	}

	composer := NewPromptComposer(section)

	// Update variables
	if !composer.SetSectionVariables("template", map[string]interface{}{"X": 42}) {
		t.Error("failed to set variables")
	}

	result, err := composer.Compose()
	if err != nil {
		t.Fatalf("compose failed: %v", err)
	}

	if !strings.Contains(result, "42") {
		t.Error("result should contain updated value 42")
	}
}

// TestPredefinedSections tests predefined section builders
func TestPredefinedSections(t *testing.T) {
	t.Run("IdentitySection", func(t *testing.T) {
		section := IdentitySection("TestBot", "A helpful testing assistant")
		composer := NewPromptComposer(section)

		result, err := composer.Compose()
		if err != nil {
			t.Fatalf("compose failed: %v", err)
		}

		if !strings.Contains(result, "TestBot") {
			t.Error("identity should contain name")
		}
		if !strings.Contains(result, "A helpful testing assistant") {
			t.Error("identity should contain description")
		}
	})

	t.Run("SkillsSection", func(t *testing.T) {
		tools := []string{"calculator", "web_search", "file_reader"}
		section := SkillsSection(tools)
		composer := NewPromptComposer(section)

		result, err := composer.Compose()
		if err != nil {
			t.Fatalf("compose failed: %v", err)
		}

		for _, tool := range tools {
			if !strings.Contains(result, tool) {
				t.Errorf("skills should contain %s", tool)
			}
		}
	})

	t.Run("SkillsSectionEmpty", func(t *testing.T) {
		section := SkillsSection([]string{})
		if section.Enabled {
			t.Error("empty skills section should be disabled")
		}
	})

	t.Run("MemorySection", func(t *testing.T) {
		memory := "Previous conversation: User asked about Go programming"
		section := MemorySection(memory)
		composer := NewPromptComposer(section)

		result, err := composer.Compose()
		if err != nil {
			t.Fatalf("compose failed: %v", err)
		}

		if !strings.Contains(result, memory) {
			t.Error("memory section should contain memory content")
		}
	})

	t.Run("MemorySectionEmpty", func(t *testing.T) {
		section := MemorySection("")
		if section.Enabled {
			t.Error("empty memory section should be disabled")
		}
	})

	t.Run("InstructionsSection", func(t *testing.T) {
		instructions := "Always be polite and concise"
		section := InstructionsSection(instructions)
		composer := NewPromptComposer(section)

		result, err := composer.Compose()
		if err != nil {
			t.Fatalf("compose failed: %v", err)
		}

		if !strings.Contains(result, instructions) {
			t.Error("instructions should contain provided text")
		}
	})

	t.Run("ConstraintsSection", func(t *testing.T) {
		constraints := []string{"No code execution", "Respect privacy"}
		section := ConstraintsSection(constraints)
		composer := NewPromptComposer(section)

		result, err := composer.Compose()
		if err != nil {
			t.Fatalf("compose failed: %v", err)
		}

		for _, constraint := range constraints {
			if !strings.Contains(result, constraint) {
				t.Errorf("constraints should contain %s", constraint)
			}
		}
	})
}

// TestComplexCompose tests a complex multi-section prompt
func TestComplexCompose(t *testing.T) {
	composer := NewPromptComposer()

	// Add multiple sections
	composer.AddSection(IdentitySection("GoHelper", "An expert Go programming assistant"))
	composer.AddSection(InstructionsSection("Help users write better Go code"))
	composer.AddSection(SkillsSection([]string{"code_analyzer", "doc_generator"}))
	composer.AddSection(ConstraintsSection([]string{"Always use Go 1.21+", "Follow Go best practices"}))

	result, err := composer.Compose()
	if err != nil {
		t.Fatalf("compose failed: %v", err)
	}

	// Verify all sections are present
	checks := []struct {
		name     string
		expected string
	}{
		{"identity", "GoHelper"},
		{"instructions", "Help users write better Go code"},
		{"skills", "code_analyzer"},
		{"constraints", "Go 1.21+"},
	}

	for _, check := range checks {
		if !strings.Contains(result, check.expected) {
			t.Errorf("%s section not found in result (expected: %s)", check.name, check.expected)
		}
	}

	// Verify order (identity should come before skills)
	identityPos := strings.Index(result, "GoHelper")
	skillsPos := strings.Index(result, "code_analyzer")
	if identityPos > skillsPos {
		t.Error("sections not in correct priority order")
	}
}

// TestListSections tests listing section names
func TestListSections(t *testing.T) {
	sections := []PromptSection{
		{Name: "section1", Content: "c1", Priority: 1, Enabled: true},
		{Name: "section2", Content: "c2", Priority: 2, Enabled: true},
		{Name: "section3", Content: "c3", Priority: 3, Enabled: true},
	}

	composer := NewPromptComposer(sections...)
	names := composer.ListSections()

	if len(names) != 3 {
		t.Errorf("expected 3 names, got %d", len(names))
	}

	for _, expected := range []string{"section1", "section2", "section3"} {
		found := false
		for _, name := range names {
			if name == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected name %s not found in list", expected)
		}
	}
}

// TestClear tests clearing all sections
func TestClear(t *testing.T) {
	sections := []PromptSection{
		{Name: "section1", Content: "c1", Priority: 1, Enabled: true},
		{Name: "section2", Content: "c2", Priority: 2, Enabled: true},
	}

	composer := NewPromptComposer(sections...)
	composer.Clear()

	if composer.SectionCount() != 0 {
		t.Errorf("expected 0 sections after clear, got %d", composer.SectionCount())
	}

	result, err := composer.Compose()
	if err != nil {
		t.Fatalf("compose failed: %v", err)
	}

	if result != "" {
		t.Errorf("expected empty result, got '%s'", result)
	}
}

// TestEmptyContentSkipping tests that empty content is skipped
func TestEmptyContentSkipping(t *testing.T) {
	sections := []PromptSection{
		{Name: "empty", Content: "", Priority: 1, Enabled: true},
		{Name: "whitespace", Content: "   \n\t  ", Priority: 2, Enabled: true},
		{Name: "content", Content: "actual content", Priority: 3, Enabled: true},
	}

	composer := NewPromptComposer(sections...)
	result, err := composer.Compose()
	if err != nil {
		t.Fatalf("compose failed: %v", err)
	}

	// Should only have actual content
	lines := strings.Split(strings.TrimSpace(result), "\n")
	if len(lines) != 1 {
		t.Errorf("expected 1 line (empty content skipped), got %d", len(lines))
	}

	if !strings.Contains(result, "actual content") {
		t.Error("result should contain actual content")
	}
}

// TestTemplateErrorHandling tests template error handling
func TestTemplateErrorHandling(t *testing.T) {
	// Test with invalid template syntax
	section := PromptSection{
		Name:       "invalid",
		Content:    "Hello {{.Name", // Missing closing brace
		Priority:   1,
		Enabled:    true,
		IsTemplate: true,
		Variables:  map[string]interface{}{},
	}

	composer := NewPromptComposer(section)

	_, err := composer.Compose()
	if err == nil {
		t.Error("expected error for invalid template syntax")
	}

	// Test with undefined variable (should not error, just render as empty)
	section2 := PromptSection{
		Name:       "undefined-var",
		Content:    "Hello {{.UndefinedVar}}",
		Priority:   2,
		Enabled:    true,
		IsTemplate: true,
		Variables:  map[string]interface{}{},
	}

	composer2 := NewPromptComposer(section2)

	result, err := composer2.Compose()
	if err != nil {
		t.Errorf("unexpected error for undefined variable: %v", err)
	}

	// Go templates render undefined variables as "<no value>"
	expected := "Hello <no value>"
	if result != expected {
		t.Errorf("expected '%s', got '%s'", expected, result)
	}
}
