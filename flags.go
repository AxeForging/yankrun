package main

import "github.com/urfave/cli/v3"

var inputFlag = &cli.StringFlag{
	Name:    "input",
	Aliases: []string{"i"},
	Value:   "",
	Usage:   "Input file with values for replacement (use '-' to read from stdin)",
	Sources: cli.EnvVars("YANKRUN_INPUT"),
}

var repoFlag = &cli.StringFlag{
	Name:    "repo",
	Aliases: []string{"r"},
	Value:   "",
	Usage:   "Url to execute clone actions",
	Sources: cli.EnvVars("YANKRUN_REPO"),
}

var outputDirFlag = &cli.StringFlag{
	Name:    "outputDir",
	Aliases: []string{"od"},
	Value:   "",
	Usage:   "Output directory to execute clone actions",
	Sources: cli.EnvVars("YANKRUN_OUTPUT_DIR"),
}

var dirFlag = &cli.StringFlag{
	Name:    "dir",
	Aliases: []string{"d"},
	Value:   "",
	Usage:   "Target directory for templating (used by template command)",
	Sources: cli.EnvVars("YANKRUN_DIR"),
}

var verboseFlag = &cli.BoolFlag{
	Name:    "verbose",
	Aliases: []string{"v"},
	Usage:   "Enable verbose mode for detailed logs",
}

var fileSizeLimitFlag = &cli.StringFlag{
	Name:    "fileSizeLimit",
	Aliases: []string{"fl"},
	Value:   "3 mb",
	Usage:   "File size limit to ignore replacements from files that exceed the limit",
}

var startDelimFlag = &cli.StringFlag{
	Name:    "startDelim",
	Aliases: []string{"sd"},
	Value:   "[[",
	Usage:   "Template start delimiter (default [[)",
}

var endDelimFlag = &cli.StringFlag{
	Name:    "endDelim",
	Aliases: []string{"ed"},
	Value:   "]]",
	Usage:   "Template end delimiter (default ]])",
}

var interactiveFlag = &cli.BoolFlag{
	Name:    "interactive",
	Aliases: []string{"prompt", "p"},
	Usage:   "Prompt for values for discovered placeholders before applying",
}

var branchFlag = &cli.StringFlag{
	Name:    "branch",
	Aliases: []string{"b"},
	Value:   "",
	Usage:   "Branch to use for generate/clone when non-interactive",
}

var templateNameFlag = &cli.StringFlag{
	Name:    "template",
	Aliases: []string{"templateName"},
	Value:   "",
	Usage:   "Template name or URL substring to select non-interactively",
}

var processTemplatesFlag = &cli.BoolFlag{
	Name:    "processTemplates",
	Aliases: []string{"pt"},
	Usage:   "Process .tpl files by evaluating templates and removing .tpl suffix",
}

var onlyTemplatesFlag = &cli.BoolFlag{
	Name:    "onlyTemplates",
	Aliases: []string{"ot"},
	Usage:   "When used with --processTemplates, only process .tpl files and ignore all other files",
}

var dryRunFlag = &cli.BoolFlag{
	Name:    "dryRun",
	Aliases: []string{"dr"},
	Usage:   "Preview what would be changed without writing any files",
}

var ignoreFlag = &cli.StringSliceFlag{
	Name:  "ignore",
	Usage: "Glob patterns for files/directories to skip (e.g. --ignore '*.generated.*' --ignore 'migrations/*')",
}

var noCacheFlag = &cli.BoolFlag{
	Name:    "noCache",
	Aliases: []string{"nc"},
	Usage:   "Bypass cache and fetch fresh data from remote",
}

var sshKeyFlag = &cli.StringFlag{
	Name:  "ssh-key",
	Value: "",
	Usage: "Path to SSH private key (auto-detects id_ed25519, id_ecdsa, id_rsa if not set)",
}

var addrFlag = &cli.StringFlag{
	Name:  "addr",
	Value: "127.0.0.1:17817",
	Usage: "Address for the local serve command",
}

// jsonFlag switches a command to machine-readable output: a single JSON
// envelope on stdout, logs suppressed to warn+ on stderr.
var jsonFlag = &cli.BoolFlag{
	Name:  "json",
	Usage: "Emit a machine-readable JSON envelope on stdout instead of human output",
}

// yesFlag guarantees non-interactive execution: never prompt, fail fast if a
// required value is missing. --no-input is an alias so both agent idioms work.
var yesFlag = &cli.BoolFlag{
	Name:    "yes",
	Aliases: []string{"no-input", "y"},
	Usage:   "Assume yes / never prompt; fail if a required value is missing",
}
