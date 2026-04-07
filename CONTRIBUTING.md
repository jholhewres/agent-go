# Contributing to AgentGo

Thank you for your interest in contributing to AgentGo! We welcome contributions of all kinds: code, documentation, bug reports, feature requests, and testing feedback.

## Ways to Contribute

- **Code**: Bug fixes, new features, performance improvements, and refactoring
- **Documentation**: README improvements, API docs, examples, and guides
- **Bug Reports**: Report issues you find with clear reproduction steps
- **Ideas**: Suggest features or improvements via GitHub Discussions
- **Testing**: Test against different Go versions and environments

## Code of Conduct

We are committed to providing a welcoming and inclusive environment. Please read our [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) before participating.

## Development Setup

### Prerequisites
- Go 1.24.1 or higher
- Git

### Installation

```bash
# Fork the repository (GitHub UI)
# Clone your fork
git clone https://github.com/YOUR_USERNAME/agent-go.git
cd agent-go

# Add upstream remote
git remote add upstream https://github.com/jholhewres/agent-go.git

# Download dependencies
go mod download

# Install development tools
make install-tools
```

## Project Structure

```
pkg/agentgo/
├── agent/          # Core Agent type and Run method
├── team/           # Multi-agent coordination
├── workflow/       # Step-based workflow engine
├── models/         # LLM providers (OpenAI, Anthropic, Ollama, etc.)
├── tools/          # Tool integrations (calculator, http, file, etc.)
├── memory/         # Conversation history management
└── types/          # Core types and error definitions

pkg/agentos/        # OS integration layer (future)

cmd/examples/       # Example programs
```

## Making Changes

### 1. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
# or for bug fixes
git checkout -b fix/issue-description
```

### 2. Follow Code Style

```bash
# Format code (gofmt + goimports)
make fmt

# Run linter
make lint

# Run go vet
make vet
```

### 3. Write Tests

All changes must include tests:

```bash
# Run tests with race detector
make test

# Check coverage
make coverage
```

Target >70% coverage for core packages. See CLAUDE.md for current coverage status.

### 4. Commit Messages

Use Conventional Commits format:

```
feat: add new feature description
fix: fix bug description
docs: update documentation
refactor: refactor code
test: add or improve tests
chore: update dependencies or tooling
```

Example:
```
feat: add Groq model provider support

- Integrate Groq API for ultra-fast LLM inference
- Add configuration for API key and model selection
- Include unit tests and example program
```

## Pull Request Process

### Before Submitting

1. Update your fork: `git pull upstream main`
2. All tests must pass: `make test`
3. No lint errors: `make lint`
4. Conventional commit messages

### Submitting a PR

1. Push your branch: `git push origin feature/your-feature-name`
2. Open PR on GitHub with clear description
3. Fill out the PR template completely
4. Link related issues: "Fixes #123" or "Related to #456"
5. CI checks must pass (automatically run)
6. Wait for maintainer review (at least 1 approval required)
7. Squash merge by default

### PR Guidelines

- One feature/fix per PR when possible
- Keep PRs focused and reasonably sized
- Update CHANGELOG.md if adding/changing user-facing features
- Include tests for all new code
- Update docs if API changes

## Adding a New Model Provider

See the detailed guide in [CLAUDE.md](CLAUDE.md#添加模型提供商).

Quick steps:
1. Create `pkg/agentgo/models/<your_model>/`
2. Implement `models.Model` interface
3. Add unit tests (target >70% coverage)
4. Run `make fmt && make test`

## Adding a New Tool

See the detailed guide in [CLAUDE.md](CLAUDE.md#添加工具).

Quick steps:
1. Create `pkg/agentgo/tools/<your_tool>/`
2. Embed `toolkit.BaseToolkit`
3. Register functions with `RegisterFunction`
4. Add unit tests
5. Run `make fmt && make test`

## Reporting Bugs

Use the [bug report template](.github/ISSUE_TEMPLATE/bug_report.yml).

Include:
- Clear description of the issue
- Steps to reproduce
- Expected vs. actual behavior
- Go version: `go version`
- AgentGo version (git commit or release tag)
- Operating system
- Relevant logs or error messages

## Requesting Features

Use the [feature request template](.github/ISSUE_TEMPLATE/feature_request.yml).

Describe:
- The problem you're trying to solve
- Your proposed solution
- Alternative approaches you've considered
- Your use case
- Whether you're willing to help implement it

## Community & Getting Help

- **GitHub Discussions**: Ask questions and discuss ideas
- **GitHub Issues**: Report bugs and request features
- **Documentation**: Check [website/](website/) and [docs/](docs/)

## License

By contributing, you agree that your contributions are under the same MIT license as the project. See [LICENSE](LICENSE).

---

Thank you for making AgentGo better!
