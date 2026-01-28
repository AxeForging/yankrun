# YankRun User Guide

A CLI tool for smart template replacement in repositories and directories.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Commands](#commands)
- [Values File Format](#values-file-format)
- [Transformation Functions](#transformation-functions)
- [Configuration](#configuration)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)

---

## Installation

<details open>
<summary><strong>Linux (AMD64)</strong></summary>

```sh
curl -L https://github.com/AxeForging/yankrun/releases/latest/download/yankrun-linux-amd64.tar.gz | tar xz
sudo mv yankrun-linux-amd64 /usr/local/bin/yankrun
yankrun version
```

</details>

<details>
<summary><strong>Linux (ARM64)</strong></summary>

```sh
curl -L https://github.com/AxeForging/yankrun/releases/latest/download/yankrun-linux-arm64.tar.gz | tar xz
sudo mv yankrun-linux-arm64 /usr/local/bin/yankrun
```

</details>

<details>
<summary><strong>macOS (Intel)</strong></summary>

```sh
curl -L https://github.com/AxeForging/yankrun/releases/latest/download/yankrun-darwin-amd64.tar.gz | tar xz
sudo mv yankrun-darwin-amd64 /usr/local/bin/yankrun
```

</details>

<details>
<summary><strong>macOS (Apple Silicon)</strong></summary>

```sh
curl -L https://github.com/AxeForging/yankrun/releases/latest/download/yankrun-darwin-arm64.tar.gz | tar xz
sudo mv yankrun-darwin-arm64 /usr/local/bin/yankrun
```

</details>

<details>
<summary><strong>Windows (PowerShell)</strong></summary>

```powershell
# Download
Invoke-WebRequest -Uri https://github.com/AxeForging/yankrun/releases/latest/download/yankrun-windows-amd64.zip -OutFile yankrun.zip

# Extract
Expand-Archive -Path yankrun.zip -DestinationPath .

# Move to PATH (adjust as needed)
Move-Item -Path yankrun-windows-amd64.exe -Destination C:\Windows\yankrun.exe

# Verify
yankrun version
```

</details>

<details>
<summary><strong>From Source (Go 1.24+)</strong></summary>

```sh
go install github.com/AxeForging/yankrun@latest
```

Or build manually:

```sh
git clone https://github.com/AxeForging/yankrun.git
cd yankrun
go build -o yankrun .
sudo mv yankrun /usr/local/bin/
```

</details>

---

## Quick Start

### Step 1: Create a values file

```yaml
# values.yaml
variables:
  - key: APP_NAME
    value: MyApp
  - key: AUTHOR
    value: Your Name
  - key: VERSION
    value: 1.0.0
```

### Step 2: Clone and template a repository

```sh
yankrun clone \
  --repo https://github.com/AxeForging/template-tester.git \
  --input values.yaml \
  --outputDir ./my-new-project \
  --verbose
```

### Step 3: Check the output

```sh
cat ./my-new-project/README.md
# All [[APP_NAME]], [[AUTHOR]], [[VERSION]] placeholders are replaced!
```

---

## Commands

### `clone`

Clone a git repository and replace placeholders.

```sh
yankrun clone --repo <URL> --outputDir <DIR> [options]
```

<details>
<summary><strong>All flags</strong></summary>

| Flag | Alias | Description | Default |
|------|-------|-------------|---------|
| `--repo` | | Git URL (HTTPS or SSH) | **required** |
| `--outputDir` | `-o` | Output directory | **required** |
| `--input` | `-i` | Values file path (JSON/YAML) | - |
| `--startDelim` | | Start delimiter | `[[` |
| `--endDelim` | | End delimiter | `]]` |
| `--fileSizeLimit` | | Skip files larger than | `3 mb` |
| `--prompt` | `-p` | Interactive mode (prompt for values) | `false` |
| `--processTemplates` | `--pt` | Process `.tpl` files | `false` |
| `--onlyTemplates` | `--ot` | Only process `.tpl` files | `false` |
| `--branch` | `-b` | Branch to clone | default branch |
| `--verbose` | `-v` | Show detailed output | `false` |

</details>

<details>
<summary><strong>Examples</strong></summary>

```sh
# Basic clone with values
yankrun clone \
  --repo https://github.com/org/template.git \
  --input values.yaml \
  --outputDir ./new-project

# Interactive mode - will prompt for each placeholder
yankrun clone \
  --repo git@github.com:org/template.git \
  --outputDir ./new-project \
  --prompt

# Clone specific branch with custom delimiters
yankrun clone \
  --repo https://github.com/org/template.git \
  --branch develop \
  --input values.json \
  --outputDir ./new-project \
  --startDelim "{{" \
  --endDelim "}}"

# Clone and process .tpl files
yankrun clone \
  --repo https://github.com/org/template.git \
  --input values.yaml \
  --outputDir ./new-project \
  --processTemplates \
  --verbose
```

</details>

---

### `template`

Apply replacements to an existing directory.

```sh
yankrun template --dir <DIR> [options]
```

<details>
<summary><strong>All flags</strong></summary>

| Flag | Alias | Description | Default |
|------|-------|-------------|---------|
| `--dir` | `-d` | Directory to process | **required** |
| `--input` | `-i` | Values file path (JSON/YAML) | - |
| `--startDelim` | | Start delimiter | `[[` |
| `--endDelim` | | End delimiter | `]]` |
| `--fileSizeLimit` | | Skip files larger than | `3 mb` |
| `--prompt` | `-p` | Interactive mode | `false` |
| `--processTemplates` | `--pt` | Process `.tpl` files | `false` |
| `--onlyTemplates` | `--ot` | Only process `.tpl` files | `false` |
| `--verbose` | `-v` | Show detailed output | `false` |

</details>

<details>
<summary><strong>Examples</strong></summary>

```sh
# Basic templating
yankrun template --dir ./my-project --input values.yaml

# Interactive mode
yankrun template --dir ./my-project --prompt --verbose

# Skip large files
yankrun template --dir ./my-project --input values.yaml --fileSizeLimit "10 mb"

# Only process .tpl files
yankrun template --dir ./my-project --input values.yaml --processTemplates --onlyTemplates
```

</details>

---

### `generate`

Interactively choose a template from your config and generate a new project.

```sh
yankrun generate [options]
```

<details>
<summary><strong>All flags</strong></summary>

| Flag | Alias | Description | Default |
|------|-------|-------------|---------|
| `--template` | `-t` | Filter templates by name | - |
| `--branch` | `-b` | Branch to clone | default branch |
| `--outputDir` | `-o` | Output directory | prompted |
| `--input` | `-i` | Values file path | - |
| `--prompt` | `-p` | Interactive mode | `false` |
| `--verbose` | `-v` | Show detailed output | `false` |

</details>

<details>
<summary><strong>Examples</strong></summary>

```sh
# Fully interactive
yankrun generate --prompt --verbose

# Filter templates and auto-select
yankrun generate --template "go-service" --outputDir ./new-service

# Non-interactive with values
yankrun generate \
  --template "api-template" \
  --branch main \
  --input values.yaml \
  --outputDir ./new-api
```

</details>

---

### `setup`

Configure YankRun defaults.

```sh
# Interactive setup
yankrun setup

# Show current config
yankrun setup --show

# Reset/delete config
yankrun setup --reset
```

---

## Values File Format

### YAML (Recommended)

```yaml
# Directories to skip (optional)
ignore_patterns:
  - node_modules
  - dist
  - .git
  - vendor
  - __pycache__

# Variables to replace (required)
variables:
  - key: APP_NAME
    value: MyApplication

  - key: PACKAGE_NAME
    value: com.example.myapp

  - key: AUTHOR_NAME
    value: Jane Developer

  - key: AUTHOR_EMAIL
    value: jane@example.com

  - key: VERSION
    value: "1.0.0"

  - key: DESCRIPTION
    value: "A sample application"

  - key: LICENSE
    value: MIT

  - key: YEAR
    value: "2024"
```

### JSON

```json
{
  "ignore_patterns": ["node_modules", "dist", ".git"],
  "variables": [
    { "key": "APP_NAME", "value": "MyApplication" },
    { "key": "PACKAGE_NAME", "value": "com.example.myapp" },
    { "key": "AUTHOR_NAME", "value": "Jane Developer" },
    { "key": "AUTHOR_EMAIL", "value": "jane@example.com" },
    { "key": "VERSION", "value": "1.0.0" }
  ]
}
```

### Key Behavior

- Keys **without** delimiters are automatically wrapped: `APP_NAME` → `[[APP_NAME]]`
- Keys **with** delimiters are used as-is: `{{APP_NAME}}` stays `{{APP_NAME}}`

---

## Transformation Functions

Apply transformations to values using the colon syntax:

```
[[KEY:function]]
[[KEY:function1:function2]]
```

### Available Functions

| Function | Description | Example | Input → Output |
|----------|-------------|---------|----------------|
| `toUpperCase` | Convert to UPPERCASE | `[[NAME:toUpperCase]]` | `hello` → `HELLO` |
| `toLowerCase` | Convert to lowercase | `[[NAME:toLowerCase]]` | `HELLO` → `hello` |
| `toDownCase` | Alias for toLowerCase | `[[NAME:toDownCase]]` | `HELLO` → `hello` |
| `gsub(a,b)` | Replace `a` with `b` | `[[NAME:gsub(-,_)]]` | `my-app` → `my_app` |

### Chaining Functions

Functions are applied left-to-right:

```
[[PROJECT_NAME:gsub( ,-):toLowerCase]]

Input:  "My Cool Project"
Step 1: "My-Cool-Project"  (gsub replaces spaces with dashes)
Step 2: "my-cool-project"  (toLowerCase)
Output: "my-cool-project"
```

### Examples in Templates

```markdown
# [[PROJECT_NAME]]

Package: [[PACKAGE_NAME:toLowerCase]]
Constant: [[PACKAGE_NAME:toUpperCase]]
URL Slug: [[PROJECT_NAME:gsub( ,-):toLowerCase]]
Database: [[APP_NAME:gsub(-,_):toLowerCase]]_db
```

---

## Configuration

YankRun stores configuration in `~/.yankrun/config.yaml`.

### Full Configuration Example

```yaml
# Default delimiters for all commands
start_delim: "[["
end_delim: "]]"

# Default file size limit
file_size_limit: "3 mb"

# Pre-configured templates for `generate` command
templates:
  - name: "Go Microservice"
    url: "git@github.com:your-org/go-service-template.git"
    description: "Production-ready Go microservice"
    default_branch: "main"

  - name: "React Application"
    url: "https://github.com/your-org/react-template.git"
    description: "React + TypeScript + Vite starter"
    default_branch: "main"

  - name: "Python CLI"
    url: "git@github.com:your-org/python-cli-template.git"
    description: "Click-based Python CLI tool"
    default_branch: "main"

# GitHub auto-discovery (optional)
github:
  user: "your-username"
  orgs:
    - "your-org"
    - "another-org"
  topic: "template"           # Only repos with this topic
  prefix: "template-"         # Only repos starting with this
  include_private: true       # Include private repositories
  token: "ghp_xxxxxxxxxxxx"   # Required for private repos
```

---

## Examples

<details>
<summary><strong>Create a Go project from template</strong></summary>

```yaml
# go-values.yaml
variables:
  - key: MODULE_NAME
    value: github.com/myorg/myservice
  - key: SERVICE_NAME
    value: user-service
  - key: AUTHOR
    value: Platform Team
  - key: GO_VERSION
    value: "1.22"
```

```sh
yankrun clone \
  --repo git@github.com:myorg/go-template.git \
  --input go-values.yaml \
  --outputDir ./user-service \
  --verbose
```

</details>

<details>
<summary><strong>Create a React app from template</strong></summary>

```yaml
# react-values.yaml
variables:
  - key: APP_NAME
    value: my-dashboard
  - key: APP_TITLE
    value: My Dashboard
  - key: DESCRIPTION
    value: Admin dashboard for managing users
```

```sh
yankrun clone \
  --repo https://github.com/myorg/react-template.git \
  --input react-values.yaml \
  --outputDir ./my-dashboard \
  --processTemplates \
  --verbose
```

</details>

<details>
<summary><strong>Update version across a monorepo</strong></summary>

```yaml
# version-bump.yaml
variables:
  - key: OLD_VERSION
    value: "1.2.3"
  - key: VERSION
    value: "1.3.0"
```

```sh
# First, manually replace OLD_VERSION with [[VERSION]] placeholders
# Then run:
yankrun template \
  --dir ./packages \
  --input version-bump.yaml \
  --verbose
```

</details>

<details>
<summary><strong>CI/CD pipeline usage</strong></summary>

```yaml
# .github/workflows/create-service.yml
name: Create Service
on:
  workflow_dispatch:
    inputs:
      service_name:
        description: 'Service name'
        required: true

jobs:
  create:
    runs-on: ubuntu-latest
    steps:
      - name: Create values file
        run: |
          cat > values.yaml << EOF
          variables:
            - key: SERVICE_NAME
              value: ${{ github.event.inputs.service_name }}
            - key: CREATED_BY
              value: ${{ github.actor }}
          EOF

      - name: Generate service
        run: |
          yankrun clone \
            --repo ${{ secrets.TEMPLATE_REPO }} \
            --input values.yaml \
            --outputDir ./output \
            --verbose
```

</details>

---

## Troubleshooting

<details>
<summary><strong>Placeholders not being replaced</strong></summary>

1. **Check delimiters match**: Ensure your template uses `[[KEY]]` (or your custom delimiters)
2. **Check key names match exactly**: `APP_NAME` ≠ `app_name`
3. **Use verbose mode**: `--verbose` shows which files are processed
4. **Check file size**: Files over 3MB are skipped by default

```sh
yankrun template --dir ./project --input values.yaml --verbose
```

</details>

<details>
<summary><strong>File being skipped</strong></summary>

Files are skipped if:
- They exceed `--fileSizeLimit` (default 3MB)
- They're in ignored directories (`.git`, `node_modules`, `vendor`, etc.)
- They're binary files

Increase the limit if needed:

```sh
yankrun template --dir ./project --input values.yaml --fileSizeLimit "50 mb"
```

</details>

<details>
<summary><strong>Git clone fails</strong></summary>

1. **Check URL**: Ensure the repo URL is correct
2. **Check authentication**: For SSH, ensure your key is loaded (`ssh-add -l`)
3. **Check network**: Ensure you can reach GitHub/GitLab

```sh
# Test git access directly
git clone https://github.com/org/repo.git /tmp/test-clone
```

</details>

<details>
<summary><strong>Permission denied on install</strong></summary>

Use `sudo` or install to a user directory:

```sh
# Option 1: Use sudo
sudo mv yankrun /usr/local/bin/

# Option 2: Install to user bin (add to PATH)
mkdir -p ~/.local/bin
mv yankrun ~/.local/bin/
export PATH="$HOME/.local/bin:$PATH"
```

</details>

---

## Getting Help

- **Documentation**: [GitHub Repository](https://github.com/AxeForging/yankrun)
- **Issues**: [GitHub Issues](https://github.com/AxeForging/yankrun/issues)
- **AI Reference**: See [docs/AI/README.md](../AI/README.md) for technical details
