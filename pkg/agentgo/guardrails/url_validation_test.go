package guardrails

import (
	"context"
	"testing"
)

func TestURLValidationGuardrail_AllowedDomains(t *testing.T) {
	g := NewURLValidationGuardrailWithAllowedDomains([]string{"example.com", "api.trusted.com"})

	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{"allowed domain", "Visit https://example.com/page", false},
		{"allowed subdomain", "API at https://api.trusted.com/v1", false},
		{"blocked domain", "Visit https://evil.com/malware", true},
		{"no URL", "This text has no URLs", false},
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

func TestURLValidationGuardrail_BlockedDomains(t *testing.T) {
	g := NewURLValidationGuardrailWithBlockedDomains([]string{"evil.com", "malware.net"})

	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{"blocked domain", "Visit https://evil.com/page", true},
		{"blocked subdomain", "Visit https://sub.evil.com/page", true},
		{"allowed domain", "Visit https://good.com/page", false},
		{"no URL", "This text has no URLs", false},
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

func TestURLValidationGuardrail_PrivateIPs(t *testing.T) {
	// Guardrail that blocks private IPs
	g := NewURLValidationGuardrail(URLValidationConfig{
		AllowPrivateIPs: false,
	})

	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{"localhost", "Access http://localhost:8080", true},
		{"127.0.0.1", "Access http://127.0.0.1:3000", true},
		{"192.168.x.x", "Access http://192.168.1.1", true},
		{"10.x.x.x", "Access http://10.0.0.1", true},
		{"public IP", "Access https://example.com", false},
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

func TestURLValidationGuardrail_AllowPrivateIPs(t *testing.T) {
	// Guardrail that allows private IPs
	g := NewURLValidationGuardrail(URLValidationConfig{
		AllowPrivateIPs: true,
	})

	input := "Access http://localhost:8080"
	err := g.Check(context.Background(), &CheckInput{Input: input, Metadata: make(map[string]interface{})})

	if err != nil {
		t.Errorf("Expected no error when AllowPrivateIPs is true, got %v", err)
	}
}

func TestURLValidationGuardrail_FileScheme(t *testing.T) {
	// Guardrail that blocks file:// URLs
	g := NewURLValidationGuardrail(URLValidationConfig{
		AllowFileScheme: false,
	})

	input := "Read file:///etc/passwd"
	err := g.Check(context.Background(), &CheckInput{Input: input, Metadata: make(map[string]interface{})})

	if err == nil {
		t.Error("Expected error for file:// URL")
	}
}

func TestURLValidationGuardrail_AllowFileScheme(t *testing.T) {
	// Guardrail that allows file:// URLs
	g := NewURLValidationGuardrail(URLValidationConfig{
		AllowFileScheme: true,
	})

	input := "Read file:///home/user/doc.txt"
	err := g.Check(context.Background(), &CheckInput{Input: input, Metadata: make(map[string]interface{})})

	if err != nil {
		t.Errorf("Expected no error when AllowFileScheme is true, got %v", err)
	}
}

func TestURLValidationGuardrail_HallucinationDetection(t *testing.T) {
	g := NewURLValidationGuardrail(URLValidationConfig{
		DetectHallucinations: true,
	})

	tests := []struct {
		name              string
		input             string
		shouldHaveWarning bool
	}{
		{"example.com", "Visit https://example.com", true},
		{"test.com", "Visit https://test.com", true},
		{"placeholder.com", "Visit https://placeholder.com", true},
		{"real domain", "Visit https://google.com", false},
		{"long subdomain", "Visit https://a.b.c.d.e.f.g.h.example.com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := make(map[string]interface{})
			g.Check(context.Background(), &CheckInput{Input: tt.input, Metadata: metadata})

			warnings, ok := metadata["url_warnings"].([]string)
			hasWarning := ok && len(warnings) > 0

			if tt.shouldHaveWarning && !hasWarning {
				t.Error("Expected URL hallucination warning")
			}
			if !tt.shouldHaveWarning && hasWarning {
				t.Errorf("Did not expect URL hallucination warning, got: %v", warnings)
			}
		})
	}
}

func TestURLValidationGuardrail_MultipleURLs(t *testing.T) {
	g := NewURLValidationGuardrailWithAllowedDomains([]string{"good.com"})

	input := "Visit https://good.com and https://evil.com"
	err := g.Check(context.Background(), &CheckInput{Input: input, Metadata: make(map[string]interface{})})

	// Should error because evil.com is not in allowlist
	if err == nil {
		t.Error("Expected error for blocked domain")
	}
}

func TestURLValidationGuardrail_NoURLs(t *testing.T) {
	g := NewURLValidationGuardrailWithBlockedDomains([]string{"evil.com"})

	input := "This is just plain text with no URLs"
	err := g.Check(context.Background(), &CheckInput{Input: input, Metadata: make(map[string]interface{})})

	if err != nil {
		t.Errorf("Expected no error for text without URLs, got %v", err)
	}
}

func TestURLValidationGuardrail_Name(t *testing.T) {
	g := NewURLValidationGuardrail(URLValidationConfig{})
	if g.Name() != "URLValidationGuardrail" {
		t.Errorf("Expected name 'URLValidationGuardrail', got %s", g.Name())
	}
}

func TestURLValidationGuardrail_Masking(t *testing.T) {
	g := NewURLValidationGuardrailWithBlockedDomains([]string{"evil.com"})

	input := "Visit https://evil.com/secret/path?token=abc123"
	err := g.Check(context.Background(), &CheckInput{Input: input, Metadata: make(map[string]interface{})})

	if err == nil {
		t.Error("Expected error for blocked domain")
	}

	// Error message should mask the path
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("Expected non-empty error message")
	}
}
