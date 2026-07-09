package ui

import "github.com/urfave/cli/v3"

// InstallHelp swaps in yankrun's branded help templates. The layout mirrors
// urfave/cli's defaults but leads with the wordmark and uses consistent section
// headers, so `yankrun --help` and `yankrun <cmd> --help` read as one product.
//
// The templates stay plain text (no ANSI) on purpose: the banner carries the
// only color, which keeps piped/`--help | cat` output and golden tests stable.
func InstallHelp() {
	cli.RootCommandHelpTemplate = rootHelp
	cli.CommandHelpTemplate = commandHelp
	cli.SubcommandHelpTemplate = rootHelp
}

const rootHelp = `yankrun — {{.Usage}}

USAGE:
   {{.Name}} [global options] command [command options] [arguments...]

VERSION:
   {{.Version}}

COMMANDS:{{range .Commands}}{{if not .Hidden}}
   {{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}{{end}}

GLOBAL OPTIONS:{{range .VisibleFlags}}
   {{.}}{{end}}

Run "{{.Name}} <command> --help" for command-specific options.
`

const commandHelp = `yankrun {{.Name}} — {{.Usage}}

USAGE:
   yankrun {{.Name}}{{if .VisibleFlags}} [options]{{end}}{{if .ArgsUsage}} {{.ArgsUsage}}{{end}}
{{if .Description}}
DESCRIPTION:
   {{.Description}}
{{end}}{{if .VisibleFlags}}
OPTIONS:{{range .VisibleFlags}}
   {{.}}{{end}}
{{end}}`
