package prompts

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"
)

// PromptSection represents a modular section of a system prompt
// PromptSection 表示系统提示的模块化部分
type PromptSection struct {
	// Name is the unique identifier for this section
	// Name 是此部分的唯一标识符
	Name string

	// Content is the prompt text (can be a template)
	// Content 是提示文本（可以是模板）
	Content string

	// Priority determines the order (lower number = higher priority)
	// Priority 决定顺序（较小的数字 = 较高的优先级）
	Priority int

	// Enabled indicates whether this section is active
	// Enabled 指示此部分是否处于活动状态
	Enabled bool

	// IsTemplate indicates if Content should be rendered as a template
	// IsTemplate 指示 Content 是否应作为模板渲染
	IsTemplate bool

	// Variables are template variables (used if IsTemplate is true)
	// Variables 是模板变量（如果 IsTemplate 为 true 则使用）
	Variables map[string]interface{}
}

// PromptComposer composes multiple prompt sections into a final prompt
// PromptComposer 将多个提示部分组合成最终提示
type PromptComposer struct {
	sections []PromptSection
	template *template.Template
}

// NewPromptComposer creates a new prompt composer
// NewPromptComposer 创建一个新的提示组合器
func NewPromptComposer(sections ...PromptSection) *PromptComposer {
	return &PromptComposer{
		sections: sections,
	}
}

// AddSection adds a new section to the composer
// AddSection 向组合器添加一个新部分
func (c *PromptComposer) AddSection(section PromptSection) {
	c.sections = append(c.sections, section)
}

// RemoveSection removes a section by name
// RemoveSection 按名称删除部分
func (c *PromptComposer) RemoveSection(name string) {
	for i, s := range c.sections {
		if s.Name == name {
			c.sections = append(c.sections[:i], c.sections[i+1:]...)
			return
		}
	}
}

// EnableSection enables a section by name
// EnableSection 按名称启用部分
func (c *PromptComposer) EnableSection(name string) bool {
	for i := range c.sections {
		if c.sections[i].Name == name {
			c.sections[i].Enabled = true
			return true
		}
	}
	return false
}

// DisableSection disables a section by name
// DisableSection 按名称禁用部分
func (c *PromptComposer) DisableSection(name string) bool {
	for i := range c.sections {
		if c.sections[i].Name == name {
			c.sections[i].Enabled = false
			return true
		}
	}
	return false
}

// SetSectionVariables updates variables for a template section
// SetSectionVariables 更新模板部分的变量
func (c *PromptComposer) SetSectionVariables(name string, vars map[string]interface{}) bool {
	for i := range c.sections {
		if c.sections[i].Name == name {
			if c.sections[i].Variables == nil {
				c.sections[i].Variables = make(map[string]interface{})
			}
			for k, v := range vars {
				c.sections[i].Variables[k] = v
			}
			return true
		}
	}
	return false
}

// Compose builds the final prompt from all enabled sections
// Compose 从所有启用的部分构建最终提示
func (c *PromptComposer) Compose() (string, error) {
	return c.ComposeWithVars(nil)
}

// ComposeWithVars builds the final prompt with additional global variables
// ComposeWithVars 使用额外的全局变量构建最终提示
func (c *PromptComposer) ComposeWithVars(globalVars map[string]interface{}) (string, error) {
	// Sort sections by priority (lower number = higher priority)
	// 按优先级排序部分（较小的数字 = 较高的优先级）
	sortedSections := make([]PromptSection, len(c.sections))
	copy(sortedSections, c.sections)

	sort.Slice(sortedSections, func(i, j int) bool {
		return sortedSections[i].Priority < sortedSections[j].Priority
	})

	var parts []string

	for _, section := range sortedSections {
		if !section.Enabled {
			continue
		}

		content := section.Content

		// Render template if needed
		// 如果需要，渲染模板
		if section.IsTemplate {
			// Merge section variables with global vars (global takes precedence)
			// 合并部分变量与全局变量（全局变量优先）
			vars := make(map[string]interface{})
			for k, v := range section.Variables {
				vars[k] = v
			}
			for k, v := range globalVars {
				vars[k] = v
			}

			rendered, err := renderTemplate(section.Name, content, vars)
			if err != nil {
				return "", fmt.Errorf("failed to render section '%s': %w", section.Name, err)
			}
			content = rendered
		}

		// Add section if not empty
		// 如果不为空则添加部分
		if strings.TrimSpace(content) != "" {
			parts = append(parts, content)
		}
	}

	// Join sections with double newline
	// 使用双换行符连接部分
	return strings.Join(parts, "\n\n"), nil
}

// renderTemplate renders a template string with variables
func renderTemplate(name, content string, vars map[string]interface{}) (string, error) {
	tmpl, err := template.New(name).Parse(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// GetSection returns a section by name (copy to prevent modification)
// GetSection 按名称返回部分（复制以防止修改）
func (c *PromptComposer) GetSection(name string) (PromptSection, bool) {
	for _, s := range c.sections {
		if s.Name == name {
			return s, true
		}
	}
	return PromptSection{}, false
}

// ListSections returns all section names
// ListSections 返回所有部分名称
func (c *PromptComposer) ListSections() []string {
	names := make([]string, 0, len(c.sections))
	for _, s := range c.sections {
		names = append(names, s.Name)
	}
	return names
}

// Clear removes all sections
// Clear 删除所有部分
func (c *PromptComposer) Clear() {
	c.sections = nil
}

// SectionCount returns the number of sections
// SectionCount 返回部分的数量
func (c *PromptComposer) SectionCount() int {
	return len(c.sections)
}

// Common section builders
// 常用部分构建器

// NewSection creates a new prompt section
// NewSection 创建一个新的提示部分
func NewSection(name, content string, priority int) PromptSection {
	return PromptSection{
		Name:     name,
		Content:  content,
		Priority: priority,
		Enabled:  true,
	}
}

// NewTemplateSection creates a new template-based prompt section
// NewTemplateSection 创建一个新的基于模板的提示部分
func NewTemplateSection(name, content string, priority int, vars map[string]interface{}) PromptSection {
	return PromptSection{
		Name:       name,
		Content:    content,
		Priority:   priority,
		Enabled:    true,
		IsTemplate: true,
		Variables:  vars,
	}
}

// Predefined sections
// 预定义的部分

// IdentitySection creates an identity section for the agent
// IdentitySection 为代理创建身份部分
func IdentitySection(name, description string) PromptSection {
	content := fmt.Sprintf(`You are %s.
%s`, name, description)
	return NewSection("identity", content, 1)
}

// SkillsSection creates a skills/tools section
// SkillsSection 创建技能/工具部分
func SkillsSection(tools []string) PromptSection {
	if len(tools) == 0 {
		return PromptSection{
			Name:     "skills",
			Content:  "",
			Priority: 20,
			Enabled:  false,
		}
	}

	content := fmt.Sprintf(`## Available Tools

You have access to the following tools:
%s

Use these tools when helpful to complete the user's request.`,
		formatToolList(tools))
	return NewSection("skills", content, 20)
}

// MemorySection creates a memory search section
// MemorySection 创建内存搜索部分
func MemorySection(memoryResults string) PromptSection {
	if memoryResults == "" {
		return PromptSection{
			Name:     "memory",
			Content:  "",
			Priority: 30,
			Enabled:  false,
		}
	}

	content := fmt.Sprintf(`## Relevant Context from Memory

%s

Use this context when responding to the user's request.`, memoryResults)
	return NewSection("memory", content, 30)
}

// InstructionsSection creates a custom instructions section
// InstructionsSection 创建自定义指令部分
func InstructionsSection(instructions string) PromptSection {
	return NewSection("instructions", instructions, 10)
}

// ConstraintsSection creates a constraints/behavior section
// ConstraintsSection 创建约束/行为部分
func ConstraintsSection(constraints []string) PromptSection {
	if len(constraints) == 0 {
		return PromptSection{
			Name:     "constraints",
			Content:  "",
			Priority: 15,
			Enabled:  false,
		}
	}

	content := fmt.Sprintf(`## Constraints

%s`, strings.Join(constraints, "\n- "))
	return NewSection("constraints", content, 15)
}

// formatToolList formats a list of tools for the prompt
func formatToolList(tools []string) string {
	var buf strings.Builder
	for _, tool := range tools {
		buf.WriteString(fmt.Sprintf("- %s\n", tool))
	}
	return buf.String()
}
