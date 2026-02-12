package guardrails

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/jholhewres/agent-go/pkg/agentgo/types"
)

// URLValidationConfig configures the URL validation guardrail
// URLValidationConfig 配置 URL 验证防护栏
type URLValidationConfig struct {
	// AllowedDomains is a whitelist of permitted domains (empty = allow all)
	// AllowedDomains 是允许的域名白名单（空 = 允许所有）
	AllowedDomains []string
	// BlockedDomains is a blacklist of forbidden domains
	// BlockedDomains 是禁止的域名黑名单
	BlockedDomains []string
	// AllowPrivateIPs determines if private/local IPs are allowed
	// AllowPrivateIPs 确定是否允许私有/本地 IP
	AllowPrivateIPs bool
	// AllowFileScheme determines if file:// URLs are allowed
	// AllowFileScheme 确定是否允许 file:// URL
	AllowFileScheme bool
	// DetectHallucinations enables detection of likely fake domains
	// DetectHallucinations 启用检测可能的虚假域名
	DetectHallucinations bool
}

// URLValidationGuardrail validates URLs in input/output
// URLValidationGuardrail 验证输入/输出中的 URL
type URLValidationGuardrail struct {
	config  URLValidationConfig
	urlPath *regexp.Regexp
}

// NewURLValidationGuardrail creates a new URL validation guardrail.
// NewURLValidationGuardrail 创建新的 URL 验证防护栏。
func NewURLValidationGuardrail(config URLValidationConfig) *URLValidationGuardrail {
	return &URLValidationGuardrail{
		config: config,
		urlPath: regexp.MustCompile(
			`[a-zA-Z][a-zA-Z0-9+.-]*://[^\s<>"{}\|]+`,
		),
	}
}

// NewURLValidationGuardrailWithAllowedDomains creates a simple allowlist guardrail.
// NewURLValidationGuardrailWithAllowedDomains 创建简单的白名单防护栏。
func NewURLValidationGuardrailWithAllowedDomains(domains []string) *URLValidationGuardrail {
	return NewURLValidationGuardrail(URLValidationConfig{
		AllowedDomains:       domains,
		AllowPrivateIPs:      false,
		AllowFileScheme:      false,
		DetectHallucinations: true,
	})
}

// NewURLValidationGuardrailWithBlockedDomains creates a simple blocklist guardrail.
// NewURLValidationGuardrailWithBlockedDomains 创建简单的黑名单防护栏。
func NewURLValidationGuardrailWithBlockedDomains(domains []string) *URLValidationGuardrail {
	return NewURLValidationGuardrail(URLValidationConfig{
		BlockedDomains:       domains,
		AllowPrivateIPs:      true,
		AllowFileScheme:      false,
		DetectHallucinations: true,
	})
}

// Check validates URLs in the input
// Check 验证输入中的 URL
func (g *URLValidationGuardrail) Check(ctx context.Context, input *CheckInput) error {
	urls := g.extractURLs(input.Input)

	for _, rawURL := range urls {
		parsed, err := url.Parse(rawURL)
		if err != nil {
			continue // Skip invalid URLs
		}

		// Check scheme
		// 检查协议
		if !g.config.AllowFileScheme && parsed.Scheme == "file" {
			return types.NewOutputCheckError(
				fmt.Sprintf("file:// URLs are not allowed: %s", g.maskURL(rawURL)),
				nil,
			)
		}

		// Check blocked domains
		// 检查黑名单域名
		for _, blocked := range g.config.BlockedDomains {
			if strings.HasSuffix(parsed.Host, blocked) || parsed.Host == blocked {
				return types.NewOutputCheckError(
					fmt.Sprintf("blocked domain detected: %s", g.maskURL(rawURL)),
					nil,
				)
			}
		}

		// Check allowed domains (if specified)
		// 检查白名单域名（如果指定）
		if len(g.config.AllowedDomains) > 0 {
			allowed := false
			for _, domain := range g.config.AllowedDomains {
				if strings.HasSuffix(parsed.Host, domain) || parsed.Host == domain {
					allowed = true
					break
				}
			}
			if !allowed {
				return types.NewOutputCheckError(
					fmt.Sprintf("domain not in allowlist: %s", g.maskURL(rawURL)),
					nil,
				)
			}
		}

		// Check private IPs
		// 检查私有 IP
		if !g.config.AllowPrivateIPs {
			if g.isPrivateIP(parsed.Host) {
				return types.NewOutputCheckError(
					fmt.Sprintf("private IP addresses are not allowed: %s", g.maskURL(rawURL)),
					nil,
				)
			}
		}

		// Check for hallucinated domains
		// 检查虚假域名
		if g.config.DetectHallucinations {
			if g.isLikelyHallucinated(parsed.Host) {
				if input.Metadata != nil {
					warnings, ok := input.Metadata["url_warnings"].([]string)
					if !ok {
						warnings = []string{}
					}
					input.Metadata["url_warnings"] = append(warnings, parsed.Host)
				}
			}
		}
	}

	return nil
}

func (g *URLValidationGuardrail) extractURLs(text string) []string {
	return g.urlPath.FindAllString(text, -1)
}

func (g *URLValidationGuardrail) maskURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "[invalid-url]"
	}
	return parsed.Scheme + "://" + parsed.Host + "/[redacted]"
}

func (g *URLValidationGuardrail) isPrivateIP(host string) bool {
	// Simple check for common private IP patterns
	// 简单检查常见的私有 IP 模式
	privatePatterns := []string{
		"10.", "172.16.", "172.17.", "172.18.", "172.19.",
		"172.20.", "172.21.", "172.22.", "172.23.", "172.24.",
		"172.25.", "172.26.", "172.27.", "172.28.", "172.29.",
		"172.30.", "172.31.",
		"192.168.", "127.", "localhost", "0.0.0.0",
		"::1", "[::1]",
	}

	lowerHost := strings.ToLower(host)
	for _, pattern := range privatePatterns {
		if strings.HasPrefix(lowerHost, strings.ToLower(pattern)) || lowerHost == strings.ToLower(pattern) {
			return true
		}
	}
	return false
}

func (g *URLValidationGuardrail) isLikelyHallucinated(domain string) bool {
	// Common hallucination patterns
	// 常见的虚假模式
	hallucinationPatterns := []string{
		"example.com", "test.com", "fake.com",
		"your-domain.com", "yourdomain.com",
		"placeholder.com", "sample.com",
		"demo.com", "dummy.com",
		"domain.com", "somedomain.com",
		"mysite.com", "mywebsite.com",
		"website.com", "site.com",
	}

	lowerDomain := strings.ToLower(domain)
	for _, pattern := range hallucinationPatterns {
		if strings.Contains(lowerDomain, pattern) {
			return true
		}
	}

	// Check for suspiciously long subdomains
	// 检查可疑的长子域名
	parts := strings.Split(domain, ".")
	if len(parts) > 4 {
		return true
	}

	// Check for placeholder patterns
	// 检查占位符模式
	placeholderPatterns := []string{
		"xxx", "yyy", "zzz",
		"abc", "123",
		"change-me", "todo",
	}
	for _, pattern := range placeholderPatterns {
		if strings.Contains(lowerDomain, pattern) {
			return true
		}
	}

	return false
}

// Name returns the guardrail name
// Name 返回防护栏名称
func (g *URLValidationGuardrail) Name() string {
	return "URLValidationGuardrail"
}
