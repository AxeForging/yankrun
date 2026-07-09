# YankRun Commands

Quick reference for the commands available in the single `yankrun` binary.

## `template`

Template an existing directory in place.

```sh
yankrun template --dir ./project --input values.yaml
yankrun template --dir ./project --input values.yaml --dryRun
yankrun template --dir ./project --input values.yaml --processTemplates
```

Useful when you already have a working tree and want to replace placeholders safely.

Key flags:

| Flag | Description |
|------|-------------|
| `--dir`, `-d` | Directory to process |
| `--input`, `-i` | YAML/JSON values file |
| `--dryRun`, `--dr` | Preview without writing |
| `--startDelim`, `--sd` | Start delimiter, default `[[` |
| `--endDelim`, `--ed` | End delimiter, default `]]` |
| `--processTemplates`, `--pt` | Process `.tpl` files and remove `.tpl` suffix |
| `--onlyTemplates`, `--ot` | Only process `.tpl` files |
| `--ignore` | Glob pattern to skip |

## `clone`

Clone a repository and apply replacements.

```sh
yankrun clone \
  --repo https://github.com/AxeForging/template-tester.git \
  --input values.yaml \
  --outputDir ./my-project
```

SSH works when your local SSH key or agent can access the repo:

```sh
yankrun clone \
  --repo git@github.com:AxeForging/template-tester.git \
  --branch main \
  --outputDir ./my-project
```

Key flags:

| Flag | Description |
|------|-------------|
| `--repo` | HTTPS or SSH Git URL |
| `--branch`, `-b` | Branch to clone |
| `--outputDir`, `-o` | Output directory |
| `--input`, `-i` | YAML/JSON values file |
| `--dryRun`, `--dr` | Preview without leaving output behind |

## `generate`

Choose from configured template repos and generate a project.

```sh
yankrun generate --prompt
yankrun generate --template go-service --input values.yaml --outputDir ./new-service
```

Templates come from `~/.yankrun/config.yaml` and optional GitHub discovery.

Key flags:

| Flag | Description |
|------|-------------|
| `--template`, `-t` | Template name/filter |
| `--branch`, `-b` | Branch to clone |
| `--outputDir`, `-o` | Output directory |
| `--input`, `-i` | YAML/JSON values file |
| `--noCache`, `--nc` | Bypass cached GitHub/template data |

## `scan`

Report the placeholders in a directory or repo without writing anything — the
read-only entry point for scripts and agents.

```sh
yankrun scan --dir ./project
yankrun scan --dir ./project --json
yankrun scan --repo https://github.com/AxeForging/template-tester.git --json
```

`--json` prints a versioned envelope (see [Agent & automation](#agent--automation)).
The payload carries discovered keys, per-file counts, and the `yankrun.yaml`
manifest when present.

## `mcp`

Serve the templating engine over the Model Context Protocol (stdio) so agents
can drive it natively.

```sh
yankrun mcp
```

Tools: `yankrun_scan`, `yankrun_evaluate`, `yankrun_apply` (dry-run returns
per-file diffs), `yankrun_clone`, `yankrun_generate`, `yankrun_manifest`,
`yankrun_templates`. Register with an MCP client:

```json
{ "mcpServers": { "yankrun": { "command": "yankrun", "args": ["mcp"] } } }
```

## `serve`

Open the local web workbench.

```sh
yankrun serve --dir ./project --input values.yaml
yankrun serve --dir ./project --addr 127.0.0.1:19090
yankrun serve --dir ./project --dryRun
```

The server binds to `127.0.0.1:17817` by default.

Use `serve` when you want:

- Local, Clone, and Generate modes in one UI
- file-level placeholder trees
- evaluated transform previews
- idle refresh while editing values
- edit the delimiter pair in the browser and rescan instantly, no restart
- saved presets per repo/template in browser IndexedDB
- JSON import/export for presets

Key flags:

| Flag | Description |
|------|-------------|
| `--dir`, `-d` | Local directory for Local mode |
| `--input`, `-i` | YAML/JSON values file |
| `--addr` | Listen address |
| `--dryRun`, `--dr` | Force preview-only mode |
| `--ignore` | Glob pattern to skip |

## `tui`

Run a conservative terminal workflow for local templating.

```sh
yankrun tui --dir ./project --input values.yaml
yankrun tui --dir ./project --dryRun
```

The TUI is a full-screen workbench (the terminal twin of `serve`): scan → fill
values → preview per-file diffs → apply. It requires an interactive terminal.

## `setup`

Manage `~/.yankrun/config.yaml`.

```sh
yankrun setup
yankrun setup --show
yankrun setup --reset
```

Use this to configure default delimiters, file size limits, template repos, and GitHub discovery.

## Agent & automation

`template`, `clone`, and `generate` (plus `scan`) are safe to run unattended.

| Feature | How |
|---------|-----|
| Machine-readable output | `--json` — one envelope on stdout, logs on stderr |
| Never prompt | `--yes` (alias `--no-input`); fails fast if a required value is missing |
| Values from stdin | `--input -` (YAML or JSON) |
| Values from the environment | `YANKRUN_VAR_<KEY>=value` |

Value precedence, lowest to highest: **manifest defaults < `--input` file < `YANKRUN_VAR_*` < interactive answers.**

The `--json` envelope is versioned and stable:

```json
{ "schemaVersion": 1, "command": "template", "ok": true, "data": { /* ApplyResult or Summary */ } }
```

On failure: `{ "schemaVersion": 1, "command": "...", "ok": false, "error": { "code": 3, "message": "..." } }`.

### Exit codes

| Code | Meaning |
|------|---------|
| 0 | success |
| 1 | unexpected internal error |
| 2 | usage error (bad flags / invalid invocation) |
| 3 | invalid input values or manifest violation (e.g. missing required in non-interactive mode) |
| 4 | directory, template, or branch not found |
| 5 | git clone / network failure |
| 130 | user cancelled an interactive prompt |

## Template manifest (`yankrun.yaml`)

A template repo may ship an optional `yankrun.yaml` at its root that declares the
variables it expects. It powers richer prompts (TUI, web, forms), pre-apply
validation, and agent self-discovery via `scan --json`. Templates without a
manifest keep working unchanged.

```yaml
version: 1
name: go-service
description: A sample Go microservice template
variables:
  - key: APP_NAME
    description: Human-friendly application name
    required: true
  - key: MODULE
    description: Go module path
    required: true
    pattern: "^[a-z0-9./-]+$"
  - key: ENV
    default: dev
    enum: [dev, staging, prod]
ignore_patterns:
  - "*.secret"
post_generate:
  hints:
    - run go mod tidy
```

A `required` value that is missing in non-interactive mode, or a value that
violates an `enum` or `pattern`, exits with code 3.

