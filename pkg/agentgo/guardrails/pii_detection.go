package guardrails

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/jholhewres/agent-go/pkg/agentgo/types"
)

// PIIType represents the kind of PII detected
// PIIType 表示检测到的 PII 类型
type PIIType string

const (
	// PIITypeEmail represents an email address
	// PIITypeEmail 表示电子邮件地址
	PIITypeEmail PIIType = "email"
	// PIITypePhone represents a phone number
	// PIITypePhone 表示电话号码
	PIITypePhone PIIType = "phone"
	// PIITypeSSN represents a US Social Security Number
	// PIITypeSSN 表示美国社会安全号码
	PIITypeSSN PIIType = "ssn"
	// PIITypeCreditCard represents a credit card number
	// PIITypeCreditCard 表示信用卡号码
	PIITypeCreditCard PIIType = "credit_card"
	// PIITypeCPF represents a Brazilian CPF
	// PIITypeCPF 表示巴西 CPF
	PIITypeCPF PIIType = "cpf"
	// PIITypeCNPJ represents a Brazilian CNPJ
	// PIITypeCNPJ 表示巴西 CNPJ
	PIITypeCNPJ PIIType = "cnpj"
)

// PIIDetection contains details about detected PII
// PIIDetection 包含检测到的 PII 的详细信息
type PIIDetection struct {
	// Type is the kind of PII detected
	// Type 是检测到的 PII 类型
	Type PIIType `json:"type"`
	// Value is the masked/redacted value
	// Value 是掩码/编辑后的值
	Value string `json:"value"`
	// StartIndex is the start position in the original text
	// StartIndex 是原始文本中的起始位置
	StartIndex int `json:"start_index"`
	// EndIndex is the end position in the original text
	// EndIndex 是原始文本中的结束位置
	EndIndex int `json:"end_index"`
	// Confidence is the detection confidence (0.0 - 1.0)
	// Confidence 是检测置信度 (0.0 - 1.0)
	Confidence float64 `json:"confidence"`
}

// PIIDetectionGuardrail detects personally identifiable information
// PIIDetectionGuardrail 检测个人身份信息
type PIIDetectionGuardrail struct {
	// EnabledTypes specifies which PII types to detect (empty = all)
	// EnabledTypes 指定要检测的 PII 类型（空 = 全部）
	EnabledTypes []PIIType
	// OnDetection specifies the action: "block" or "warn"
	// OnDetection 指定操作："block" 或 "warn"
	OnDetection string
	// MaskInOutput determines if PII should be masked in error messages
	// MaskInOutput 确定是否在错误消息中掩码 PII
	MaskInOutput bool
	// patterns is the compiled regex patterns for each PII type
	// patterns 是每种 PII 类型的编译正则表达式模式
	patterns map[PIIType]*regexp.Regexp
}

// NewPIIDetectionGuardrail creates a new PII detection guardrail with default settings.
// NewPIIDetectionGuardrail 使用默认设置创建新的 PII 检测防护栏。
func NewPIIDetectionGuardrail() *PIIDetectionGuardrail {
	g := &PIIDetectionGuardrail{
		EnabledTypes: []PIIType{
			PIITypeEmail, PIITypePhone, PIITypeSSN,
			PIITypeCreditCard, PIITypeCPF, PIITypeCNPJ,
		},
		OnDetection:  "block",
		MaskInOutput: true,
	}
	g.compilePatterns()
	return g
}

// NewPIIDetectionGuardrailWithTypes creates a guardrail for specific PII types.
// NewPIIDetectionGuardrailWithTypes 为特定 PII 类型创建防护栏。
func NewPIIDetectionGuardrailWithTypes(piiTypes []PIIType, action string) *PIIDetectionGuardrail {
	g := &PIIDetectionGuardrail{
		EnabledTypes: piiTypes,
		OnDetection:  action,
		MaskInOutput: true,
	}
	g.compilePatterns()
	return g
}

func (g *PIIDetectionGuardrail) compilePatterns() {
	g.patterns = make(map[PIIType]*regexp.Regexp)

	// Email pattern
	// 电子邮件模式
	g.patterns[PIITypeEmail] = regexp.MustCompile(
		`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
	)

	// Phone pattern (international format, including Brazilian)
	// 电话号码模式（国际格式，包括巴西）
	g.patterns[PIITypePhone] = regexp.MustCompile(
		`(?:\+?1[-.\s]?)?(?:\(?[0-9]{3}\)?[-.\s]?)?[0-9]{3}[-.\s]?[0-9]{4}|(?:\+?55[-.\s]?)?(?:\(?[0-9]{2}\)?[-.\s]?)?[0-9]{4,5}[-.\s]?[0-9]{4}`,
	)

	// US SSN pattern (XXX-XX-XXXX)
	// 美国社会安全号码模式 (XXX-XX-XXXX)
	g.patterns[PIITypeSSN] = regexp.MustCompile(
		`\b[0-9]{3}-[0-9]{2}-[0-9]{4}\b`,
	)

	// Credit card pattern (basic format)
	// 信用卡模式（基本格式）
	g.patterns[PIITypeCreditCard] = regexp.MustCompile(
		`\b(?:[0-9]{4}[-\s]?){3}[0-9]{4}\b`,
	)

	// Brazilian CPF (XXX.XXX.XXX-XX)
	// 巴西 CPF (XXX.XXX.XXX-XX)
	g.patterns[PIITypeCPF] = regexp.MustCompile(
		`\b[0-9]{3}\.[0-9]{3}\.[0-9]{3}-[0-9]{2}\b`,
	)

	// Brazilian CNPJ (XX.XXX.XXX/XXXX-XX)
	// 巴西 CNPJ (XX.XXX.XXX/XXXX-XX)
	g.patterns[PIITypeCNPJ] = regexp.MustCompile(
		`\b[0-9]{2}\.[0-9]{3}\.[0-9]{3}/[0-9]{4}-[0-9]{2}\b`,
	)
}

// Check validates the input for PII
// Check 验证输入中的 PII
func (g *PIIDetectionGuardrail) Check(ctx context.Context, input *CheckInput) error {
	var detections []PIIDetection

	for _, piiType := range g.EnabledTypes {
		pattern, ok := g.patterns[piiType]
		if !ok {
			continue
		}

		matches := pattern.FindAllStringIndex(input.Input, -1)
		for _, match := range matches {
			value := input.Input[match[0]:match[1]]
			detections = append(detections, PIIDetection{
				Type:       piiType,
				Value:      g.maskValue(value),
				StartIndex: match[0],
				EndIndex:   match[1],
				Confidence: 0.9, // Default confidence for pattern match
			})
		}
	}

	if len(detections) > 0 {
		msg := g.buildDetectionMessage(detections)

		if g.OnDetection == "block" {
			return types.NewPIIDetectedError(msg, nil)
		}
		// For "warn", add to metadata but don't block
		// 对于 "warn"，添加到元数据但不阻止
		if input.Metadata != nil {
			input.Metadata["pii_warnings"] = detections
		}
	}

	return nil
}

func (g *PIIDetectionGuardrail) maskValue(value string) string {
	if !g.MaskInOutput {
		return value
	}
	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}

func (g *PIIDetectionGuardrail) buildDetectionMessage(detections []PIIDetection) string {
	typeCounts := make(map[PIIType]int)
	for _, d := range detections {
		typeCounts[d.Type]++
	}

	var parts []string
	for t, count := range typeCounts {
		parts = append(parts, fmt.Sprintf("%s: %d", t, count))
	}

	return fmt.Sprintf("PII detected in input: %s", strings.Join(parts, ", "))
}

// Name returns the guardrail name
// Name 返回防护栏名称
func (g *PIIDetectionGuardrail) Name() string {
	return "PIIDetectionGuardrail"
}
