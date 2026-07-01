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

The TUI scans, summarizes, previews replacement counts, and writes only when not in dry-run mode.

## `setup`

Manage `~/.yankrun/config.yaml`.

```sh
yankrun setup
yankrun setup --show
yankrun setup --reset
```

Use this to configure default delimiters, file size limits, template repos, and GitHub discovery.

