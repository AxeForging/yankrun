# YankRun - AI Assistant Reference

This document provides structured information for AI assistants working with the YankRun codebase.

## Quick Reference

| Item | Value |
|------|-------|
| **Purpose** | CLI tool for template value replacement in files and git repositories |
| **Language** | Go 1.24+ |
| **Repository** | https://github.com/AxeForging/yankrun |
| **License** | MIT |
| **Binary Size** | ~12-13MB (includes go-git library) |

---

## Architecture

```
yankrun/
├── main.go                 # Entry point, CLI app setup
├── flags.go                # CLI flag definitions
├── actions/                # Command handlers
│   ├── clone.go            # `clone` command
│   ├── generate.go         # `generate` command
│   ├── setup.go            # `setup` command
│   └── template.go         # `template` command
├── services/               # Business logic
│   ├── cloner.go           # Git clone operations
│   ├── configio.go         # Config file I/O (~/.yankrun/config.yaml)
│   ├── filesystem.go       # File system abstraction
│   ├── github.go           # GitHub API for template discovery
│   ├── parser.go           # YAML/JSON input parsing
│   ├── replacer.go         # Core replacement logic + transformations
│   └── replacer_test.go    # Unit tests for replacer
├── domain/                 # Data models
│   ├── config.go           # Config structs
│   └── replacements.go     # Replacement structs
├── helpers/                # Utilities
│   ├── error.go            # Error handling helpers
│   └── logger.go           # Zerolog setup
├── integration/            # Integration tests
│   ├── integration_test.go
│   ├── case_transformations_test.go
│   └── template_processing_test.go
├── docs/                   # Documentation
│   ├── user/README.md      # User guide
│   └── AI/README.md        # This file
└── doc/
    └── functions.md        # Transformation functions reference
```

---

## Key Components

### Command Flow

```
User Command → main.go → actions/*.go → services/*.go → File System
                 ↓
              flags.go (parse flags)
                 ↓
              services/configio.go (load defaults from ~/.yankrun/config.yaml)
                 ↓
              services/parser.go (parse input JSON/YAML)
                 ↓
              services/replacer.go (scan + replace placeholders)
```

### Core Files

| File | Purpose | Key Functions |
|------|---------|---------------|
| `services/replacer.go` | Placeholder scanning and replacement | `ReplaceInDir()`, `AnalyzeDir()`, `ProcessTemplateFiles()` |
| `services/parser.go` | Parse JSON/YAML input files | `Parse()` |
| `services/cloner.go` | Git clone operations | `CloneRepository()`, `CloneRepositoryBranch()` |
| `services/configio.go` | Config file management | `Load()`, `Save()`, `Reset()` |
| `actions/clone.go` | Clone command handler | `Execute()` |
| `actions/template.go` | Template command handler | `Execute()` |

---

## Testing Strategy

**Approach**: Build-first integration testing

The tests compile the actual binary and execute it, verifying real end-to-end behavior:

```go
func buildBinary(t *testing.T) string {
    bin := filepath.Join(t.TempDir(), "yankrun-test")
    if runtime.GOOS == "windows" {
        bin += ".exe"
    }
    cmd := exec.Command("go", "build", "-o", bin, ".")
    cmd.Dir = repoRoot(t)
    out, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("build failed: %v\n%s", err, string(out))
    }
    return bin
}
```

### Test Files

| File | Tests |
|------|-------|
| `integration/integration_test.go` | `TestCloneNonInteractive`, `TestTemplateNonInteractive` |
| `integration/case_transformations_test.go` | `TestCaseTransformations` (toUpperCase, toLowerCase, gsub) |
| `integration/template_processing_test.go` | `TestTemplateProcessingIntegration`, `TestCloneWithTemplateProcessing` |
| `services/replacer_test.go` | Unit tests for `.tpl` processing |

### Running Tests

```sh
# All tests
go test ./... -v

# Specific test
go test ./integration -run TestCloneNonInteractive -v

# With coverage
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

## Build System

### Local Development

```sh
# Build for current platform
go build -o yankrun .

# Build all platforms
make build

# Clean build artifacts
make clean

# Show version info
make version
```

### Version Injection

Version info is injected via ldflags:

```sh
go build -ldflags="-s -w -X main.Version=v1.0.0 -X main.BuildTime=$(date -u '+%Y-%m-%d_%H:%M:%S') -X main.GitCommit=$(git rev-parse --short HEAD)" -o yankrun .
```

Variables in `main.go`:

```go
var (
    Version   = "dev"
    BuildTime = "unknown"
    GitCommit = "unknown"
)
```

### Platforms Built

| OS | Architectures |
|----|---------------|
| Linux | amd64, arm64, 386, arm |
| macOS | amd64, arm64 |
| Windows | amd64, arm64, 386 |

---

## CI/CD

### Test Workflow (`.github/workflows/test.yml`)

- **Triggers**: All pushes, all PRs
- **Action**: `go test ./... -v`

### Release Workflow (`.github/workflows/release.yml`)

- **Triggers**: Manual dispatch with `tag` input
- **Action**: Build all platforms, create GitHub Release with archives

```sh
# Trigger release
gh workflow run release.yml -f tag=v1.0.0
```

---

## Common Tasks for AI

### Adding a New Command

1. Define flags in `flags.go`
2. Create action handler in `actions/newcommand.go`
3. Register command in `main.go`
4. Add integration tests in `integration/`

### Adding a Transformation Function

1. Edit `services/replacer.go` - find the `applyTransformations()` function
2. Add new case in the switch statement
3. Add test in `integration/case_transformations_test.go`
4. Document in `doc/functions.md`

Example:

```go
// In services/replacer.go
case "capitalize":
    if len(value) > 0 {
        value = strings.ToUpper(string(value[0])) + strings.ToLower(value[1:])
    }
```

### Modifying Replacement Logic

Core logic is in `services/replacer.go`:

- `AnalyzeDir()` - Scans directory for placeholders
- `ReplaceInDir()` - Performs replacements
- `ProcessTemplateFiles()` - Handles `.tpl` files

### Debugging a Test Failure

```sh
# Run single test with verbose output
go test ./integration -run TestCloneNonInteractive -v

# Build with debug symbols
go build -gcflags="all=-N -l" -o yankrun .

# Check verbose command output
yankrun clone --repo <url> --outputDir /tmp/test --input values.yaml --verbose
```

---

## Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/urfave/cli` | v1.22.17 | CLI framework |
| `github.com/go-git/go-git/v5` | v5.16.4 | Pure Go git implementation |
| `github.com/rs/zerolog` | v1.34.0 | Structured logging |
| `gopkg.in/yaml.v3` | v3.0.1 | YAML parsing |
| `github.com/mitchellh/go-homedir` | v1.1.0 | Home directory resolution |
| `golang.org/x/crypto` | v0.47.0 | Cryptographic operations |

All dependencies are kept up-to-date for security.

---

## Error Handling Patterns

```go
// Fatal errors - log and exit
if err != nil {
    helpers.LogAndExit("operation failed", err)
}

// Non-fatal logging
log.Info().Str("file", path).Msg("processing file")
log.Warn().Err(err).Msg("skipping file")
log.Debug().Int("count", n).Msg("replacements made")
```

---

## Configuration Structure

```go
// domain/config.go
type Config struct {
    StartDelim    string         `yaml:"start_delim"`
    EndDelim      string         `yaml:"end_delim"`
    FileSizeLimit string         `yaml:"file_size_limit"`
    Templates     []TemplateRepo `yaml:"templates"`
    GitHub        GitHubConfig   `yaml:"github"`
}

type TemplateRepo struct {
    Name          string `yaml:"name"`
    URL           string `yaml:"url"`
    Description   string `yaml:"description"`
    DefaultBranch string `yaml:"default_branch"`
}

type GitHubConfig struct {
    User           string   `yaml:"user"`
    Orgs           []string `yaml:"orgs"`
    Topic          string   `yaml:"topic"`
    Prefix         string   `yaml:"prefix"`
    IncludePrivate bool     `yaml:"include_private"`
    Token          string   `yaml:"token"`
}
```

---

## Common AI Prompts

### "Add a new transformation function"

1. Edit `services/replacer.go`
2. Find `applyTransformations()` function
3. Add case in switch statement
4. Add test in `integration/case_transformations_test.go`
5. Update `doc/functions.md`

### "Fix a bug in placeholder replacement"

1. Check `services/replacer.go` - `ReplaceInDir()` and `replaceInFile()`
2. Run existing tests: `go test ./... -v`
3. Add regression test if needed

### "Add a new CLI flag"

1. Add flag definition in `flags.go`
2. Use flag in appropriate action handler in `actions/`
3. Update documentation in README and docs/user/

### "Update dependencies"

```sh
go get -u ./...
go mod tidy
go test ./... -v  # Verify nothing broke
```

---

## Useful Commands

```sh
# Check for outdated dependencies
go list -m -u all

# Run linter
golangci-lint run

# Format code
gofmt -w .

# Check for vulnerabilities
govulncheck ./...

# Generate test coverage report
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

---

## Links

- **Repository**: https://github.com/AxeForging/yankrun
- **Releases**: https://github.com/AxeForging/yankrun/releases
- **Issues**: https://github.com/AxeForging/yankrun/issues
- **Test Fixtures**: https://github.com/AxeForging/template-tester
