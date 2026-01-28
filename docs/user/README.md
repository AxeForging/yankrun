# YankRun User Guide

A CLI tool for smart template replacement in repositories and directories.

## Quick Start

### Installation

Download the latest release for your platform:

```sh
# Linux AMD64
curl -L https://github.com/AxeForging/yankrun/releases/latest/download/yankrun-linux-amd64.tar.gz | tar xz
sudo mv yankrun-linux-amd64 /usr/local/bin/yankrun

# macOS ARM64 (Apple Silicon)
curl -L https://github.com/AxeForging/yankrun/releases/latest/download/yankrun-darwin-arm64.tar.gz | tar xz
sudo mv yankrun-darwin-arm64 /usr/local/bin/yankrun
```

Or build from source:
```sh
go install github.com/AxeForging/yankrun@latest
```

### Basic Usage

**1. Clone a template repo and replace placeholders:**
```sh
yankrun clone \
  --repo https://github.com/your-org/template.git \
  --input values.yaml \
  --outputDir ./my-project \
  --verbose
```

**2. Template an existing directory:**
```sh
yankrun template \
  --dir ./my-project \
  --input values.yaml \
  --verbose
```

**3. Interactive template generation:**
```sh
yankrun generate --prompt --verbose
```

## Values File Format

### YAML (recommended)
```yaml
ignore_patterns: [node_modules, dist, .git]
variables:
  - key: APP_NAME
    value: MyApp
  - key: VERSION
    value: 1.0.0
  - key: AUTHOR
    value: Your Name
```

### JSON
```json
{
  "ignore_patterns": ["node_modules", "dist", ".git"],
  "variables": [
    { "key": "APP_NAME", "value": "MyApp" },
    { "key": "VERSION", "value": "1.0.0" }
  ]
}
```

## Placeholder Syntax

Default delimiters are `[[` and `]]`. Use them in your template files:

```
# [[APP_NAME]]

Welcome to [[APP_NAME]] version [[VERSION]]!
```

### Transformation Functions

Apply transformations to values:

| Function | Example | Input | Output |
|----------|---------|-------|--------|
| `toUpperCase` | `[[APP_NAME:toUpperCase]]` | `my-app` | `MY-APP` |
| `toLowerCase` | `[[APP_NAME:toLowerCase]]` | `My-App` | `my-app` |
| `gsub(old,new)` | `[[NAME:gsub( ,-)]]` | `My App` | `My-App` |

Chain multiple transformations:
```
[[NAME:gsub( ,-):toLowerCase]]  # "My App" -> "my-app"
```

## Commands Reference

### `clone`
Clone a git repository and apply template replacements.

| Flag | Description | Default |
|------|-------------|---------|
| `--repo` | Git URL to clone | required |
| `--input`, `-i` | Path to values file (JSON/YAML) | - |
| `--outputDir`, `-o` | Output directory | required |
| `--startDelim` | Start delimiter | `[[` |
| `--endDelim` | End delimiter | `]]` |
| `--fileSizeLimit` | Skip files larger than | `3 mb` |
| `--prompt`, `-p` | Interactive mode | `false` |
| `--processTemplates`, `--pt` | Process `.tpl` files | `false` |
| `--verbose`, `-v` | Verbose output | `false` |

### `template`
Apply replacements to an existing directory.

| Flag | Description | Default |
|------|-------------|---------|
| `--dir`, `-d` | Directory to process | required |
| `--input`, `-i` | Path to values file | - |
| (other flags same as clone) | | |

### `generate`
Interactive template selection and generation.

| Flag | Description | Default |
|------|-------------|---------|
| `--templateName` | Template name from config | - |
| `--branch` | Branch to clone | default branch |
| (other flags same as clone) | | |

### `setup`
Configure YankRun defaults.

```sh
yankrun setup           # Interactive configuration
yankrun setup --show    # Show current config
yankrun setup --reset   # Delete config file
```

## Configuration

YankRun stores configuration in `~/.yankrun/config.yaml`:

```yaml
start_delim: "[["
end_delim: "]]"
file_size_limit: "3 mb"

# Template sources for `generate` command
templates:
  - name: "Go Service"
    url: "git@github.com:your-org/go-template.git"
    description: "Go microservice template"
    default_branch: "main"

# GitHub discovery for templates
github:
  orgs: ["your-org"]
  topic: "template"
  prefix: "template-"
  include_private: true
```

## Template File Processing

Files ending in `.tpl` can be processed and renamed:

```sh
yankrun template --dir ./project --input values.yaml --processTemplates
```

This will:
1. Find all `.tpl` files
2. Replace placeholders in them
3. Create new files without the `.tpl` suffix
4. Delete the original `.tpl` files

Example: `README.md.tpl` becomes `README.md` with placeholders replaced.

## Tips

- Use `--verbose` to see detailed replacement reports
- Use `--fileSizeLimit` to skip large binary files
- Ignored directories by default: `.git`, `node_modules`, `vendor`, `dist`
- Test with a small directory first before processing large repos
