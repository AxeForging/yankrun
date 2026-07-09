package main

import (
	"context"
	"fmt"
	"os"

	"github.com/AxeForging/yankrun/actions"
	"github.com/AxeForging/yankrun/helpers"
	"github.com/AxeForging/yankrun/internal/ui"
	"github.com/AxeForging/yankrun/services"

	"github.com/urfave/cli/v3"
)

// Version information - these will be set during build
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Instantiate required services
	fs := &services.OsFileSystem{}
	parser := &services.YAMLJSONParser{FileSystem: fs}
	replacer := &services.FileReplacer{FileSystem: fs}
	cloner := &services.GitCloner{FileSystem: fs}

	// Pass them to actions
	templateAction := actions.NewTemplateAction(fs, parser, replacer)
	cloneAction := actions.NewCloneAction(fs, parser, replacer, cloner)
	generateAction := actions.NewGenerateAction(fs, cloner, parser, replacer)
	serveAction := actions.NewServeAction(fs, parser, replacer, cloner)
	tuiAction := actions.NewTUIAction(fs, parser, replacer)
	scanAction := actions.NewScanAction(fs, parser, replacer, cloner)

	// Logger writes to stderr; data goes to stdout. Level is raised later when
	// --json is set so machine output stays clean.
	helpers.SetupLogger("info")
	ui.InstallHelp()

	cmd := &cli.Command{
		Name:    "yankrun",
		Usage:   "Template smarter — clone repos, replace tokens, scaffold projects",
		Version: Version,
		Commands: []*cli.Command{
			{
				Name:    "template",
				Aliases: []string{"t"},
				Usage:   "Template a local directory in place",
				Flags:   []cli.Flag{inputFlag, dirFlag, verboseFlag, fileSizeLimitFlag, startDelimFlag, endDelimFlag, interactiveFlag, processTemplatesFlag, onlyTemplatesFlag, dryRunFlag, ignoreFlag, jsonFlag, yesFlag},
				Action:  templateAction.Execute,
			},
			{
				Name:    "clone",
				Aliases: []string{"r"},
				Usage:   "Clone a repo and apply template file replacements",
				Flags:   []cli.Flag{repoFlag, inputFlag, outputDirFlag, verboseFlag, fileSizeLimitFlag, startDelimFlag, endDelimFlag, interactiveFlag, branchFlag, processTemplatesFlag, onlyTemplatesFlag, dryRunFlag, ignoreFlag, sshKeyFlag, jsonFlag, yesFlag},
				Action:  cloneAction.Execute,
			},
			{
				Name:   "generate",
				Usage:  "Choose a template repo/branch and clone it as a new repo (removes .git)",
				Flags:  []cli.Flag{inputFlag, outputDirFlag, verboseFlag, fileSizeLimitFlag, startDelimFlag, endDelimFlag, interactiveFlag, templateNameFlag, branchFlag, processTemplatesFlag, onlyTemplatesFlag, dryRunFlag, ignoreFlag, noCacheFlag, sshKeyFlag, jsonFlag, yesFlag},
				Action: generateAction.Execute,
			},
			{
				Name:   "scan",
				Usage:  "Scan a directory or repo for placeholders and report them (human or --json)",
				Flags:  []cli.Flag{dirFlag, repoFlag, branchFlag, inputFlag, verboseFlag, fileSizeLimitFlag, startDelimFlag, endDelimFlag, onlyTemplatesFlag, ignoreFlag, sshKeyFlag, jsonFlag},
				Action: scanAction.Execute,
			},
			{
				Name:   "tui",
				Usage:  "Open the interactive terminal workbench for templating a directory",
				Flags:  []cli.Flag{inputFlag, dirFlag, verboseFlag, fileSizeLimitFlag, startDelimFlag, endDelimFlag, processTemplatesFlag, onlyTemplatesFlag, dryRunFlag, ignoreFlag},
				Action: tuiAction.Execute,
			},
			{
				Name:   "serve",
				Usage:  "Run the local web workbench for scanning, previewing, and applying values",
				Flags:  []cli.Flag{inputFlag, dirFlag, addrFlag, verboseFlag, fileSizeLimitFlag, startDelimFlag, endDelimFlag, processTemplatesFlag, onlyTemplatesFlag, dryRunFlag, ignoreFlag},
				Action: serveAction.Execute,
			},
			{
				Name:  "setup",
				Usage: "Create or update ~/.yankrun/config.yaml (use --show to display, --reset to delete)",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "show", Usage: "show current configuration"},
					&cli.BoolFlag{Name: "reset", Usage: "delete ~/.yankrun/config.yaml"},
				},
				Action: func(_ context.Context, cmd *cli.Command) error {
					return actions.RunSetup(cmd.Bool("show"), cmd.Bool("reset"))
				},
			},
			{
				Name:  "version",
				Usage: "Show version information",
				Action: func(_ context.Context, _ *cli.Command) error {
					fmt.Printf("yankrun version %s\n", Version)
					fmt.Printf("Build time: %s\n", BuildTime)
					fmt.Printf("Git commit: %s\n", GitCommit)
					return nil
				},
			},
		},
		// One place maps every error to its documented exit code. Diagnostics go
		// to stderr; --json actions have already written their envelope to stdout.
		ExitErrHandler: func(_ context.Context, _ *cli.Command, err error) {
			if err == nil {
				return
			}
			if msg := err.Error(); msg != "" {
				fmt.Fprintln(os.Stderr, msg)
			}
			os.Exit(helpers.ExitCode(err))
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		// Reached only if ExitErrHandler didn't exit (defensive).
		os.Exit(helpers.ExitCode(err))
	}
}
