# YankRun - AI Assistant Reference

This document provides structured information for AI assistants working with the YankRun codebase.

## Quick Reference

| Item | Value |
|------|-------|
| **Purpose** | CLI, TUI, and local web workbench for template value replacement in files and git repositories |
| **Language** | Go 1.25+ |
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
│   ├── serve.go            # `serve` command wrapper
│   └── template.go         # `template` command
├── internal/
│   ├── tui/                # Terminal preview workflow
│   ├── web/                # Embedded web UI, handlers, static assets
│   └── workflow/           # Shared scan/apply/clone/generate workflow layer
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
User Command → main.go → actions/*.go → internal/workflow → services/*.go → File System
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
| `services/replacer.go` | Placeholder scanning, evaluated previews, and replacement | `ReplaceInDir()`, `AnalyzeDirDetails()`, `EvaluatePlaceholder()`, `ProcessTemplateFiles()` |
| `services/parser.go` | Parse JSON/YAML input files | `Parse()` |
| `services/cloner.go` | Git clone operations | `CloneRepository()`, `CloneRepositoryBranch()` |
| `services/configio.go` | Config file management | `Load()`, `Save()`, `Reset()` |
| `internal/workflow/workflow.go` | Shared workflow used by CLI/TUI/web | `ScanDir()`, `ApplyDir()`, `CloneAndApply()` |
| `internal/web/server.go` | Embedded local workbench API | `Scan()`, `Apply()`, `Clone()`, `Generate()`, `SetDelimiters()`, `ValidateDelimiters()` |
| `internal/tui/tui.go` | Preview-first terminal workflow | `Run()` |
| `actions/clone.go` | Clone command handler | `Execute()` |
| `actions/template.go` | Template command handler | `Execute()` |

### Interactive Surfaces

- `serve` embeds `internal/web/templates` and `internal/web/static` into the single binary.
- The web UI supports local scan/apply, direct clone, and generate from configured templates.
- Preview responses include file-level placeholder trees and evaluated transform previews.
- `POST /api/delimiters` lets the browser change the active start/end delimiter pair at runtime (`Server.SetDelimiters`); it validates with `ValidateDelimiters` (rejects empty, equal, or mutually-containing pairs — an empty pair would otherwise hang the literal scan in `services/replacer.go`), updates `Server.startDelim`/`endDelim` under `Server.mu`, and returns a fresh scan. The new pair applies to Local, Clone, and Generate alike since they all read it from the same `Server.settings()`.
- Browser IndexedDB stores saved presets locally; JSON import/export is client-side only.
- `tui` uses the same workflow engine for local directory scan/apply and remains preview-first.

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

| Package | Purpose |
|---------|---------|
| `github.com/urfave/cli/v3` | CLI framework (context-aware; migrated from v1) |
| `github.com/go-git/go-git/v5` | Pure Go git implementation |
| `github.com/rs/zerolog` | Structured logging (stderr) |
| `github.com/charmbracelet/{bubbletea,bubbles,lipgloss,huh}` | TUI + interactive prompts + terminal theme |
| `github.com/modelcontextprotocol/go-sdk` | MCP server (`yankrun mcp`) |
| `github.com/pmezard/go-difflib` | Unified dry-run diffs |
| `gopkg.in/yaml.v3` | YAML parsing |
| `github.com/mitchellh/go-homedir` | Home directory resolution |

Pin exact versions from `go.mod`; do not hardcode them here.

---

## Agent-facing contract

The machine-readable surface is a stable, versioned contract — treat it as public API.

- **JSON envelope** (`internal/schema`): every `--json` command prints one
  `{schemaVersion, command, ok, data|error}` object to stdout. `data` wraps the
  same `workflow.Summary` / `workflow.ApplyResult` structs the human path uses.
- **Exit codes** (`helpers/exit.go`): 0 ok · 1 internal · 2 usage · 3 validation ·
  4 not-found · 5 git · 130 cancelled. Constructors: `UsageErr`, `ValidationErr`,
  `NotFoundErr`, `GitErr`, `CancelledErr`; mapped once in `main`'s `ExitErrHandler`.
- **Value precedence** (`workflow.ResolveValues`): manifest defaults < `--input`
  file (or stdin `-`) < `YANKRUN_VAR_*` env < interactive answers.
- **Manifest** (`domain.Manifest`, `services.LoadManifest`/`ValidateValues`):
  optional `yankrun.yaml`; drives prompts, validation, and `scan --json`.
- **MCP** (`internal/mcp`): thin wrappers over `workflow.Engine`; outputs are the
  same JSON-tagged types as `--json`.
- **Non-interactive guarantee**: prompts (huh forms, the TUI) are gated behind
  `helpers.IsInteractive()` so agents/CI never hang.

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
