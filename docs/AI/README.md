# YankRun - AI Assistant Reference

This document provides structured information for AI assistants working with the YankRun codebase.

## Project Overview

**Purpose**: CLI tool for template value replacement in files and git repositories.

**Language**: Go 1.23+

**Repository**: https://github.com/AxeForging/yankrun

## Architecture

```
yankrun/
├── main.go              # Entry point, CLI app setup
├── flags.go             # CLI flag definitions
├── actions/             # Command handlers
│   ├── clone.go         # `clone` command implementation
│   ├── generate.go      # `generate` command implementation
│   ├── setup.go         # `setup` command implementation
│   └── template.go      # `template` command implementation
├── services/            # Business logic
│   ├── cloner.go        # Git clone operations
│   ├── configio.go      # Config file I/O
│   ├── filesystem.go    # File system abstraction
│   ├── github.go        # GitHub API interactions
│   ├── parser.go        # YAML/JSON parsing
│   └── replacer.go      # Template replacement logic
├── domain/              # Data models
│   ├── config.go        # Configuration structures
│   └── replacements.go  # Replacement data structures
├── helpers/             # Utilities
│   ├── error.go         # Error handling
│   └── logger.go        # Logging setup
└── integration/         # Integration tests
    ├── integration_test.go
    ├── case_transformations_test.go
    └── template_processing_test.go
```

## Key Components

### Command Flow
1. User invokes command (`clone`, `template`, `generate`, `setup`)
2. Action handler in `actions/` processes flags
3. Services in `services/` execute business logic
4. Results are logged via `helpers/logger.go`

### Template Replacement (`services/replacer.go`)
- Scans files for placeholders between configurable delimiters
- Applies transformation functions (`toUpperCase`, `toLowerCase`, `gsub`)
- Respects file size limits and ignore patterns
- Processes `.tpl` files when flag is set

### Configuration (`services/configio.go`)
- Stores defaults in `~/.yankrun/config.yaml`
- Supports template sources and GitHub discovery
- CLI flags override config values

## Testing Strategy

**Approach**: Build-first integration testing

Tests compile the actual binary and execute it, verifying real behavior:

```go
func buildBinary(t *testing.T) string {
    bin := filepath.Join(t.TempDir(), "yankrun-test")
    cmd := exec.Command("go", "build", "-o", bin, ".")
    cmd.Dir = repoRoot(t)
    out, err := cmd.CombinedOutput()
    if err != nil {
        t.Fatalf("build failed: %v\n%s", err, string(out))
    }
    return bin
}
```

Test files:
- `integration/integration_test.go` - Clone and template commands
- `integration/case_transformations_test.go` - Transformation functions
- `integration/template_processing_test.go` - `.tpl` file handling

Run tests:
```sh
go test ./... -v
```

## Build System

### Local Build
```sh
make build              # Build all platforms
make clean              # Remove build artifacts
make version            # Show version info
make release-check      # Build and test before release
```

### Version Injection
```sh
VERSION=v1.0.0 make build
```

Builds inject version info via ldflags:
```go
var (
    Version   = "dev"
    BuildTime = "unknown"
    GitCommit = "unknown"
)
```

### Platforms Built
- linux/amd64, linux/arm64, linux/386, linux/arm
- darwin/amd64, darwin/arm64
- windows/amd64, windows/arm64, windows/386

## CI/CD

### Test Workflow (`.github/workflows/test.yml`)
- Triggers: All pushes, all PRs
- Runs: `go test ./... -v`

### Release Workflow (`.github/workflows/release.yml`)
- Triggers: Manual dispatch with tag input
- Creates: Archives for all platforms
- Publishes: GitHub Release with auto-generated notes

## Common Tasks

### Adding a New Command
1. Add flag definitions in `flags.go`
2. Create action handler in `actions/`
3. Add command to `main.go`
4. Write integration tests

### Adding a Transformation Function
1. Edit `services/replacer.go`
2. Add case in transformation switch
3. Add tests in `integration/case_transformations_test.go`
4. Document in `doc/functions.md`

### Modifying Replacement Logic
1. Core logic: `services/replacer.go`
2. Test: `integration/integration_test.go`
3. Ensure all existing tests pass

## Dependencies

| Package | Purpose |
|---------|---------|
| `github.com/urfave/cli` | CLI framework |
| `github.com/go-git/go-git/v5` | Git operations |
| `github.com/rs/zerolog` | Structured logging |
| `gopkg.in/yaml.v3` | YAML parsing |
| `github.com/mitchellh/go-homedir` | Home directory resolution |

## Error Handling Patterns

```go
// Use helpers.LogAndExit for fatal errors
if err != nil {
    helpers.LogAndExit("operation failed", err)
}

// Use zerolog for non-fatal logging
log.Info().Str("file", path).Msg("processing file")
log.Warn().Err(err).Msg("skipping file")
```

## Debugging Tips

- Use `--verbose` flag to see detailed replacement reports
- Check `~/.yankrun/config.yaml` for stored configuration
- Run single test: `go test ./integration -run TestCloneNonInteractive -v`
- Build with debug info: `go build -gcflags="all=-N -l" -o yankrun .`
