# YankRun

<div align="center">
  <img src="doc/logo.png" alt="YankRun" width="200">
  <p>
    <img src="https://img.shields.io/badge/Go-1.24%2B-00ADD8?style=flat-square&logo=go" alt="Go Version">
    <img src="https://img.shields.io/badge/OS-Linux%20%7C%20macOS%20%7C%20Windows-darkblue?style=flat-square&logo=windows" alt="OS Support">
    <img src="https://img.shields.io/badge/License-MIT-green?style=flat-square" alt="License">
  </p>
</div>

**Template smarter**: Clone repos and replace tokens safely with size limits, custom delimiters, and JSON/YAML inputs.

## TL;DR

```sh
# Install
curl -L https://github.com/AxeForging/yankrun/releases/latest/download/yankrun-linux-amd64.tar.gz | tar xz
sudo mv yankrun-linux-amd64 /usr/local/bin/yankrun

# Clone a template and replace placeholders
yankrun clone --repo https://github.com/AxeForging/template-tester.git \
  --input values.yaml --outputDir ./my-project --verbose

# Or template an existing directory
yankrun template --dir ./my-project --input values.yaml --verbose
```

## Features

- **Template values replacement** across a directory tree
- **Git clone** with post-clone templating
- **Custom delimiters** with smart wrapping (default `[[` `]]`)
- **Size-based skipping** (default 3 MB)
- **Verbose reporting** with per-file replacement counts
- **JSON/YAML inputs** and ignore patterns
- **Transformation functions** (`toUpperCase`, `toLowerCase`, `gsub`)
- **Template file processing** (`.tpl` files processed and renamed)

## Documentation

| Audience | Link |
|----------|------|
| **Users** | [docs/user/README.md](docs/user/README.md) - Installation, usage, examples |
| **AI Assistants** | [docs/AI/README.md](docs/AI/README.md) - Architecture, testing, common tasks |
| **Transformations** | [doc/functions.md](doc/functions.md) - Function reference |

---

## Install

<details>
<summary><strong>Linux/macOS (AMD64)</strong></summary>

```sh
curl -L https://github.com/AxeForging/yankrun/releases/latest/download/yankrun-linux-amd64.tar.gz | tar xz
chmod +x yankrun-linux-amd64
sudo mv yankrun-linux-amd64 /usr/local/bin/yankrun
```

</details>

<details>
<summary><strong>Linux/macOS (ARM64 / Apple Silicon)</strong></summary>

```sh
# Linux ARM64
curl -L https://github.com/AxeForging/yankrun/releases/latest/download/yankrun-linux-arm64.tar.gz | tar xz
chmod +x yankrun-linux-arm64
sudo mv yankrun-linux-arm64 /usr/local/bin/yankrun

# macOS Apple Silicon
curl -L https://github.com/AxeForging/yankrun/releases/latest/download/yankrun-darwin-arm64.tar.gz | tar xz
chmod +x yankrun-darwin-arm64
sudo mv yankrun-darwin-arm64 /usr/local/bin/yankrun
```

</details>

<details>
<summary><strong>Windows (PowerShell)</strong></summary>

```powershell
Invoke-WebRequest -Uri https://github.com/AxeForging/yankrun/releases/latest/download/yankrun-windows-amd64.zip -OutFile yankrun.zip
Expand-Archive -Path yankrun.zip -DestinationPath .
Move-Item -Path yankrun-windows-amd64.exe -Destination yankrun.exe
```

</details>

<details>
<summary><strong>From Source (Go 1.24+)</strong></summary>

```sh
go install github.com/AxeForging/yankrun@latest
```

Or build locally:

```sh
git clone https://github.com/AxeForging/yankrun.git
cd yankrun
go build -o yankrun .
sudo mv yankrun /usr/local/bin/
```

</details>

---

## Quick Start

### 1. Create a values file

```yaml
# values.yaml
variables:
  - key: APP_NAME
    value: MyAwesomeApp
  - key: AUTHOR
    value: Jane Developer
  - key: VERSION
    value: 1.0.0
```

### 2. Clone and template

```sh
yankrun clone \
  --repo https://github.com/AxeForging/template-tester.git \
  --input values.yaml \
  --outputDir ./my-new-project \
  --verbose
```

### 3. Result

All `[[APP_NAME]]`, `[[AUTHOR]]`, `[[VERSION]]` placeholders are replaced with your values.

---

## Commands

<details>
<summary><strong>clone</strong> - Clone a repo and replace placeholders</summary>

```sh
# Non-interactive with values file
yankrun clone \
  --repo https://github.com/AxeForging/template-tester.git \
  --input values.yaml \
  --outputDir ./my-project \
  --verbose

# Interactive mode - prompts for each placeholder
yankrun clone \
  --repo git@github.com:AxeForging/template-tester.git \
  --outputDir ./my-project \
  --prompt --verbose

# With custom delimiters
yankrun clone \
  --repo https://github.com/example/repo.git \
  --input values.json \
  --outputDir ./out \
  --startDelim "{{" --endDelim "}}"

# Process .tpl files (README.md.tpl → README.md)
yankrun clone \
  --repo https://github.com/example/repo.git \
  --input values.yaml \
  --outputDir ./out \
  --processTemplates
```

**Flags:**

| Flag | Alias | Description | Default |
|------|-------|-------------|---------|
| `--repo` | | Git URL (HTTPS or SSH) | required |
| `--outputDir` | `-o` | Output directory | required |
| `--input` | `-i` | Values file (JSON/YAML) | - |
| `--startDelim` | | Start delimiter | `[[` |
| `--endDelim` | | End delimiter | `]]` |
| `--fileSizeLimit` | | Skip files larger than | `3 mb` |
| `--prompt` | `-p` | Interactive mode | `false` |
| `--processTemplates` | `--pt` | Process `.tpl` files | `false` |
| `--onlyTemplates` | `--ot` | Only process `.tpl` files | `false` |
| `--verbose` | `-v` | Verbose output | `false` |
| `--branch` | `-b` | Branch to clone | default |

</details>

<details>
<summary><strong>template</strong> - Template an existing directory</summary>

```sh
# Basic usage
yankrun template --dir ./my-project --input values.yaml --verbose

# Interactive mode
yankrun template --dir ./my-project --prompt

# With custom delimiters and size limit
yankrun template \
  --dir ./my-project \
  --input values.yaml \
  --startDelim "{{" --endDelim "}}" \
  --fileSizeLimit "10 mb" \
  --verbose
```

**Flags:**

| Flag | Alias | Description | Default |
|------|-------|-------------|---------|
| `--dir` | `-d` | Directory to process | required |
| `--input` | `-i` | Values file (JSON/YAML) | - |
| `--startDelim` | | Start delimiter | `[[` |
| `--endDelim` | | End delimiter | `]]` |
| `--fileSizeLimit` | | Skip files larger than | `3 mb` |
| `--prompt` | `-p` | Interactive mode | `false` |
| `--processTemplates` | `--pt` | Process `.tpl` files | `false` |
| `--onlyTemplates` | `--ot` | Only process `.tpl` files | `false` |
| `--verbose` | `-v` | Verbose output | `false` |

</details>

<details>
<summary><strong>generate</strong> - Interactive template selection</summary>

```sh
# Interactive - choose template from config
yankrun generate --prompt --verbose

# Non-interactive with template filter
yankrun generate --template "go-service" --input values.yaml --outputDir ./new-project

# With branch selection
yankrun generate --template "api" --branch "feature/v2" --outputDir ./new-api
```

Requires templates configured in `~/.yankrun/config.yaml` or GitHub discovery enabled.

</details>

<details>
<summary><strong>setup</strong> - Configure defaults</summary>

```sh
# Interactive setup
yankrun setup

# Show current config
yankrun setup --show

# Reset config
yankrun setup --reset
```

Creates/updates `~/.yankrun/config.yaml`.

</details>

---

## Values File Format

<details>
<summary><strong>YAML (recommended)</strong></summary>

```yaml
# Optional: directories to skip
ignore_patterns:
  - node_modules
  - dist
  - .git
  - vendor

# Required: key-value pairs
variables:
  - key: APP_NAME
    value: MyApp
  - key: PROJECT_NAME
    value: my-awesome-project
  - key: AUTHOR_NAME
    value: Jane Developer
  - key: AUTHOR_EMAIL
    value: jane@example.com
  - key: VERSION
    value: "1.0.0"
  - key: YEAR
    value: "2024"
```

</details>

<details>
<summary><strong>JSON</strong></summary>

```json
{
  "ignore_patterns": ["node_modules", "dist", ".git"],
  "variables": [
    { "key": "APP_NAME", "value": "MyApp" },
    { "key": "PROJECT_NAME", "value": "my-awesome-project" },
    { "key": "AUTHOR_NAME", "value": "Jane Developer" },
    { "key": "AUTHOR_EMAIL", "value": "jane@example.com" },
    { "key": "VERSION", "value": "1.0.0" }
  ]
}
```

</details>

---

## Transformation Functions

Apply transformations to placeholder values using the syntax `[[KEY:function]]`:

| Function | Syntax | Input | Output |
|----------|--------|-------|--------|
| Uppercase | `[[APP:toUpperCase]]` | `my-app` | `MY-APP` |
| Lowercase | `[[APP:toLowerCase]]` | `My-App` | `my-app` |
| Replace | `[[APP:gsub(-,_)]]` | `my-app` | `my_app` |
| Chain | `[[APP:gsub( ,-):toLowerCase]]` | `My App` | `my-app` |

<details>
<summary><strong>More examples</strong></summary>

```
# Template file content
Package: [[PACKAGE_NAME:toLowerCase]]
Constant: [[PACKAGE_NAME:toUpperCase]]
ClassName: [[PACKAGE_NAME:gsub(-,_)]]
URL-safe: [[PROJECT_NAME:gsub( ,-):toLowerCase]]

# With PACKAGE_NAME=My-Package and PROJECT_NAME=My Project
Package: my-package
Constant: MY-PACKAGE
ClassName: My_Package
URL-safe: my-project
```

</details>

See [doc/functions.md](doc/functions.md) for full documentation.

---

## Configuration

<details>
<summary><strong>Full config example (~/.yankrun/config.yaml)</strong></summary>

```yaml
# Default delimiters
start_delim: "[["
end_delim: "]]"

# Skip files larger than this
file_size_limit: "3 mb"

# Pre-configured templates for `generate` command
templates:
  - name: "Go Microservice"
    url: "git@github.com:your-org/go-template.git"
    description: "Production-ready Go service template"
    default_branch: "main"

  - name: "React App"
    url: "https://github.com/your-org/react-template.git"
    description: "React + TypeScript starter"
    default_branch: "main"

# GitHub auto-discovery for templates
github:
  user: "your-username"              # Your GitHub username
  orgs: ["your-org", "another-org"]  # Organizations to search
  topic: "template"                  # Filter by topic
  prefix: "template-"                # Filter by name prefix
  include_private: true              # Include private repos
  token: "ghp_xxxx"                  # Optional: for private repos
```

</details>

---

## Use Cases

<details>
<summary><strong>Bootstrap a new project from a template</strong></summary>

**Problem:** You maintain a template repo with CI, linting, and base code. You want to create a new project with your names filled in, without the template's git history.

**Solution:**

```sh
yankrun generate --prompt --verbose
```

Or non-interactively:

```sh
cat > values.yaml << 'EOF'
variables:
  - key: PROJECT_NAME
    value: my-new-service
  - key: AUTHOR
    value: Platform Team
EOF

yankrun clone \
  --repo git@github.com:your-org/service-template.git \
  --input values.yaml \
  --outputDir ./my-new-service \
  --verbose
```

**Result:** Fresh project with all placeholders replaced, no template git history.

</details>

<details>
<summary><strong>Batch update config across many files</strong></summary>

**Problem:** You need to update company name, version, or email across dozens of files.

**Solution:**

```sh
cat > updates.yaml << 'EOF'
variables:
  - key: COMPANY_NAME
    value: "New Company Inc"
  - key: SUPPORT_EMAIL
    value: "support@newcompany.com"
  - key: VERSION
    value: "2.0.0"
EOF

yankrun template --dir . --input updates.yaml --verbose
```

**Result:** Consistent updates across all files with a detailed report.

</details>

<details>
<summary><strong>CI/CD pipeline templating</strong></summary>

**Problem:** You need to stamp out projects in an automated pipeline with no user interaction.

**Solution:**

```sh
# In your CI script
yankrun clone \
  --repo "$TEMPLATE_REPO" \
  --input /config/values.json \
  --outputDir ./output \
  --startDelim "{{" --endDelim "}}" \
  --processTemplates \
  --verbose
```

**Result:** Fully templated project ready for deployment, no prompts.

</details>

<details>
<summary><strong>Process .tpl template files</strong></summary>

**Problem:** You have `Dockerfile.tpl`, `config.yaml.tpl` files that should be processed and renamed.

**Solution:**

```sh
yankrun template --dir ./project --input values.yaml --processTemplates --verbose
```

**Result:**
- `Dockerfile.tpl` → `Dockerfile` (with placeholders replaced)
- `config.yaml.tpl` → `config.yaml` (with placeholders replaced)
- Original `.tpl` files are removed

</details>

---

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Error (invalid flags, clone failed, etc.) |

---

## License

MIT - see [LICENSE](LICENSE)
