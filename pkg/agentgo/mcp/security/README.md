# MCP Security - Custom Binary Support

## Overview

This package provides security validation for MCP (Model Context Protocol) server commands, with support for custom binary executables.

## Features

### 8-Layer Validation Strategy

The `PathValidator` implements a comprehensive 8-layer validation strategy:

1. **Shell Metacharacter Check**
   - Blocks dangerous characters like `;`, `|`, `&`, `` ` ``, `$`, etc.

2. **Command Splitting**
   - Parses command strings to extract the executable

3. **Relative Path Validation**
   - Validates scripts starting with `./` or `../`
   - Windows support: `.\\` and `..\\`

4. **Absolute Path Validation**
   - Validates full file system paths
   - Checks file existence and permissions

5. **Current Directory Check**
   - Looks for executables in the current working directory

6. **PATH Lookup**
   - Uses `exec.LookPath` to find commands in system PATH

7. **Whitelist Validation**
   - Checks against allowed commands list

8. **Argument Validation**
   - Validates all command arguments for dangerous characters

## Usage

### Basic Usage

```go
import "github.com/jholhewres/agent-go/pkg/agentgo/mcp/security"

// Create a path validator
validator := security.NewPathValidator(nil) // nil uses default whitelist

// Validate a command
err := validator.ValidateExecutable("python", []string{"-m", "server"})
if err != nil {
    log.Fatal(err)
}
```

### With Custom Whitelist

```go
// Create custom command validator
customValidator := security.NewCustomCommandValidator(
    []string{"python", "node", "my-custom-tool"},
    security.DefaultBlockedChars(),
)

// Create path validator with custom whitelist
pathValidator := security.NewPathValidator(customValidator)

// Validate
err := pathValidator.ValidateExecutable("my-custom-tool", []string{"arg1"})
```

### Relative Path Scripts

```go
// Validate relative path script
err := validator.ValidateExecutable("./my_script.sh", []string{})
```

### Absolute Path Executables

```go
// Add script name to whitelist first
validator.GetValidator().AddAllowedCommand("my_script.sh")

// Validate absolute path
err := validator.ValidateExecutable("/path/to/my_script.sh", []string{})
```

## Integration with MCP Client

The validation is automatically integrated into `StdioTransport`:

### Default Validation (Enabled)

```go
import "github.com/jholhewres/agent-go/pkg/agentgo/mcp/client"

// Validation enabled by default when AllowedCommands is set
config := client.StdioConfig{
    Command: "python",
    Args:    []string{"-m", "mcp_server"},
    ValidateCommand: true, // Explicit enable
}

transport, err := client.NewStdioTransport(config)
```

### Custom Whitelist

```go
config := client.StdioConfig{
    Command: "./my_custom_server.sh",
    Args:    []string{"--port", "8000"},
    AllowedCommands: []string{"my_custom_server.sh"},
}

transport, err := client.NewStdioTransport(config)
```

### Disable Validation (Not Recommended)

```go
config := client.StdioConfig{
    Command: "bash",
    Args:    []string{"-c", "some command"},
    ValidateCommand: false, // Explicitly disable
}

transport, err := client.NewStdioTransport(config)
```

## Security Considerations

### Default Whitelist

The default whitelist includes commonly used MCP server commands:

- `python`, `python3`
- `node`, `npm`, `npx`
- `uvx`
- `docker`

### Blocked Characters

The following shell metacharacters are blocked:

- Command separators: `;`, `&`, `|`, `\n`, `\r`
- Command substitution: `` ` ``, `$`
- Redirection: `<`, `>`
- Glob patterns: `*`, `?`, `[`, `]`
- Brace expansion: `{`, `}`
- Escape character: `\`
- Quotes: `'`, `"`

### Cross-Platform Support

The validator supports both Unix and Windows path formats:

- Unix: `./script`, `../script`, `/usr/bin/python`
- Windows: `.\\script`, `..\\script`, `C:\\Python\\python.exe`

**Note**: Backslashes (`\`) are blocked by default. Use forward slashes (`/`) for cross-platform compatibility.

## Performance

Benchmark results on Apple M3:

```
BenchmarkPathValidator_ValidateExecutable_Simple-8     59895    19898 ns/op    8104 B/op    94 allocs/op
BenchmarkPathValidator_ValidateExecutable_WithPath-8  312634     3857 ns/op     248 B/op     3 allocs/op
```

- Simple validation (PATH lookup): ~20 us
- Absolute path validation: ~4 us

## Testing

The package includes comprehensive tests:

- **Test Coverage**: 90.1%
- **Unit Tests**: All validation layers
- **Edge Cases**: Empty input, path traversal, null bytes
- **Cross-Platform**: Unix and Windows paths

Run tests:

```bash
go test -v ./pkg/agentgo/mcp/security/...
```

## Examples

See `/cmd/examples/mcp_custom_binary/main.go` for complete examples:

1. Whitelisted commands
2. Relative path scripts
3. Absolute path executables
4. Custom whitelist
5. Validation disabled (for testing)

## Migration from Python

This Go implementation provides equivalent functionality to Python's MCP validation with improvements:

| Feature | Python | Go |
|---------|--------|-----|
| Shell metacharacter check | Yes | Yes |
| Command splitting | Yes | Yes |
| Relative path validation | Yes | Yes |
| Absolute path validation | Yes | Yes |
| PATH lookup | Yes | Yes |
| Whitelist validation | Yes | Yes |
| Windows support | Partial | Yes |
| Performance | ~baseline | ~10x faster |

## API Reference

### Types

```go
type PathValidator struct {
    validator *CommandValidator
}
```

### Functions

```go
// NewPathValidator creates a new path validator
func NewPathValidator(validator *CommandValidator) *PathValidator

// ValidateExecutable validates an executable using 8-layer strategy
func (pv *PathValidator) ValidateExecutable(executable string, args []string) error

// GetValidator returns the underlying command validator
func (pv *PathValidator) GetValidator() *CommandValidator
```

## License

See repository LICENSE file.
