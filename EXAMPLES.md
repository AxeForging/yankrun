# YankRun Examples

Practical examples for common template workflows.

## Try the public sample template

```sh
cat > values.yaml <<'EOF'
variables:
  - key: APP_NAME
    value: TemplateTester
  - key: PROJECT_NAME
    value: DemoProject
  - key: USER_NAME
    value: tester
  - key: USER_EMAIL
    value: tester@example.com
  - key: VERSION
    value: 9.9.9
EOF

yankrun clone \
  --repo https://github.com/AxeForging/template-tester.git \
  --input values.yaml \
  --outputDir ./template-test
```

## Preview a repo in the web workbench

```sh
yankrun serve --dir ./template-test --input values.yaml
```

Open `http://127.0.0.1:17817`, then use:

- **Local** for the directory passed with `--dir`
- **Clone** for a direct SSH/HTTPS repo URL
- **Generate** for templates configured in `~/.yankrun/config.yaml`

The workbench shows a file tree for each placeholder and evaluated transform previews such as `APP_NAME:toUpperCase -> TEMPLATETESTER`.

## Clone with SSH auth

```sh
yankrun clone \
  --repo git@github.com:AxeForging/template-tester.git \
  --branch main \
  --input values.yaml \
  --outputDir ./template-test
```

This uses the same local SSH setup as Git. If your key or agent works with `git clone`, YankRun should work too.

## Configure templates for `generate`

Create or edit `~/.yankrun/config.yaml`:

```yaml
templates:
  - name: template-tester
    url: https://github.com/AxeForging/template-tester.git
    description: YankRun sample template repo
    default_branch: main
```

Then run:

```sh
yankrun generate --template template-tester --input values.yaml --outputDir ./generated-app
```

Or use the **Generate** tab in `yankrun serve`.

## Template an existing project

```sh
cat > updates.yaml <<'EOF'
variables:
  - key: COMPANY_NAME
    value: AxeForge
  - key: SUPPORT_EMAIL
    value: support@example.com
  - key: VERSION
    value: 2.0.0
EOF

yankrun template --dir ./project --input updates.yaml --dryRun
yankrun template --dir ./project --input updates.yaml
```

## Avoid Helm or Go template conflicts

YankRun defaults to `[[` and `]]`, so it will not touch Helm/Go template syntax like `{{ .Values.name }}`.

If a project already uses `[[` and `]]`, choose custom delimiters:

```sh
yankrun template \
  --dir ./project \
  --input values.yaml \
  --startDelim "<%" \
  --endDelim "%>"
```

Template files can then contain:

```text
name: <%APP_NAME%>
version: <%VERSION%>
```

## Process `.tpl` files

```sh
yankrun template \
  --dir ./project \
  --input values.yaml \
  --processTemplates
```

`README.md.tpl` becomes `README.md` after template processing. Existing target files are not overwritten silently.

## Use transforms

Template:

```text
Package: [[APP_NAME:toLowerCase]]
Constant: [[APP_NAME:toUpperCase]]
URL slug: [[PROJECT_NAME:gsub( ,-):toLowerCase]]
```

Values:

```yaml
variables:
  - key: APP_NAME
    value: My App
  - key: PROJECT_NAME
    value: My Project
```

Result:

```text
Package: my app
Constant: MY APP
URL slug: my-project
```

## Save and reuse presets in `serve`

1. Run `yankrun serve --dir ./project --input values.yaml`.
2. Preview a Local, Clone, or Generate run.
3. The run is saved locally in browser IndexedDB.
4. Use the left preset rail to search by repo, template, branch, output directory, or value keys.
5. Export/import presets as JSON when moving between browsers.

