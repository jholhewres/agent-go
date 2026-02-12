package guardrails

import (
	"context"
	"strings"
	"testing"
)

func TestPIIDetectionGuardrail_Email(t *testing.T) {
	g := NewPIIDetectionGuardrail()

	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{"valid email", "Contact me at user@example.com for more info", true},
		{"multiple emails", "Email foo@bar.com and baz@qux.com", true},
		{"no email", "This text has no email addresses", false},
		{"partial email", "user@ is not a complete email", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.Check(context.Background(), &CheckInput{Input: tt.input, Metadata: make(map[string]interface{})})
			if tt.shouldErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestPIIDetectionGuardrail_Phone(t *testing.T) {
	g := NewPIIDetectionGuardrail()

	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{"US phone", "Call me at 555-123-4567", true},
		{"Brazilian phone", "Meu telefone é 11 98765-4321", true},
		{"no phone", "This text has no phone numbers", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.Check(context.Background(), &CheckInput{Input: tt.input, Metadata: make(map[string]interface{})})
			if tt.shouldErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestPIIDetectionGuardrail_SSN(t *testing.T) {
	g := NewPIIDetectionGuardrail()

	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{"valid SSN", "My SSN is 123-45-6789", true},
		{"no SSN", "This text has no SSN", false},
		{"wrong format", "Random text without SSN format", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.Check(context.Background(), &CheckInput{Input: tt.input, Metadata: make(map[string]interface{})})
			if tt.shouldErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestPIIDetectionGuardrail_CreditCard(t *testing.T) {
	g := NewPIIDetectionGuardrail()

	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{"credit card with dashes", "Card: 1234-5678-9012-3456", true},
		{"credit card with spaces", "Card: 1234 5678 9012 3456", true},
		{"no credit card", "This text has no credit card", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.Check(context.Background(), &CheckInput{Input: tt.input, Metadata: make(map[string]interface{})})
			if tt.shouldErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestPIIDetectionGuardrail_CPF(t *testing.T) {
	g := NewPIIDetectionGuardrail()

	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{"valid CPF", "Meu CPF é 123.456.789-00", true},
		{"no CPF", "Este texto não tem CPF", false},
		{"wrong format", "CPF format is XXX.XXX.XXX-XX", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.Check(context.Background(), &CheckInput{Input: tt.input, Metadata: make(map[string]interface{})})
			if tt.shouldErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestPIIDetectionGuardrail_CNPJ(t *testing.T) {
	g := NewPIIDetectionGuardrail()

	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{"valid CNPJ", "O CNPJ da empresa é 12.345.678/0001-90", true},
		{"no CNPJ", "Este texto não tem CNPJ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.Check(context.Background(), &CheckInput{Input: tt.input, Metadata: make(map[string]interface{})})
			if tt.shouldErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestPIIDetectionGuardrail_MultipleTypes(t *testing.T) {
	g := NewPIIDetectionGuardrail()

	input := "Contact user@example.com or call 555-123-4567. My CPF is 123.456.789-00."
	err := g.Check(context.Background(), &CheckInput{Input: input, Metadata: make(map[string]interface{})})

	if err == nil {
		t.Error("Expected error for multiple PII types, got nil")
	}

	// Check that error message mentions multiple types
	if !strings.Contains(err.Error(), "email") {
		t.Error("Expected error to mention email")
	}
	if !strings.Contains(err.Error(), "phone") {
		t.Error("Expected error to mention phone")
	}
	if !strings.Contains(err.Error(), "cpf") {
		t.Error("Expected error to mention cpf")
	}
}

func TestPIIDetectionGuardrail_SpecificTypes(t *testing.T) {
	// Only detect email
	g := NewPIIDetectionGuardrailWithTypes([]PIIType{PIITypeEmail}, "block")

	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{"email should error", "Contact user@example.com", true},
		{"phone should pass", "Call 555-123-4567", false},
		{"CPF should pass", "CPF: 123.456.789-00", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := g.Check(context.Background(), &CheckInput{Input: tt.input, Metadata: make(map[string]interface{})})
			if tt.shouldErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}

func TestPIIDetectionGuardrail_WarnMode(t *testing.T) {
	g := NewPIIDetectionGuardrailWithTypes([]PIIType{PIITypeEmail}, "warn")

	metadata := make(map[string]interface{})
	err := g.Check(context.Background(), &CheckInput{
		Input:    "Contact user@example.com",
		Metadata: metadata,
	})

	// Should not error in warn mode
	if err != nil {
		t.Errorf("Expected no error in warn mode, got %v", err)
	}

	// Should have warnings in metadata
	warnings, ok := metadata["pii_warnings"]
	if !ok {
		t.Error("Expected pii_warnings in metadata")
	}
	if warnings == nil {
		t.Error("Expected warnings to be non-nil")
	}
}

func TestPIIDetectionGuardrail_Masking(t *testing.T) {
	g := NewPIIDetectionGuardrail()
	g.MaskInOutput = true

	err := g.Check(context.Background(), &CheckInput{
		Input:    "Contact verylongemail@domain.com",
		Metadata: make(map[string]interface{}),
	})

	if err == nil {
		t.Error("Expected error, got nil")
	}

	// Error should contain masked email, not full email
	if strings.Contains(err.Error(), "verylongemail@domain.com") {
		t.Error("Expected email to be masked in error message")
	}
}

func TestPIIDetectionGuardrail_Name(t *testing.T) {
	g := NewPIIDetectionGuardrail()
	if g.Name() != "PIIDetectionGuardrail" {
		t.Errorf("Expected name 'PIIDetectionGuardrail', got %s", g.Name())
	}
}
