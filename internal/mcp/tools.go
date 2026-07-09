package mcp

import (
	"context"
	"fmt"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/internal/workflow"
	"github.com/AxeForging/yankrun/services"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Tool input types. Field tags drive the JSON Schema the SDK advertises to
// clients, so descriptions here become the agent-facing docs.

type scanInput struct {
	Dir            string   `json:"dir" jsonschema:"the directory to scan for placeholders"`
	StartDelim     string   `json:"startDelim,omitempty" jsonschema:"start delimiter (default [[)"`
	EndDelim       string   `json:"endDelim,omitempty" jsonschema:"end delimiter (default ]])"`
	IgnorePatterns []string `json:"ignorePatterns,omitempty" jsonschema:"glob patterns to skip"`
	OnlyTemplates  bool     `json:"onlyTemplates,omitempty" jsonschema:"only consider .tpl files"`
}

type evaluateInput struct {
	Dir            string            `json:"dir" jsonschema:"the directory to scan"`
	Values         map[string]string `json:"values" jsonschema:"placeholder values to preview"`
	StartDelim     string            `json:"startDelim,omitempty"`
	EndDelim       string            `json:"endDelim,omitempty"`
	IgnorePatterns []string          `json:"ignorePatterns,omitempty"`
}

type applyInput struct {
	Dir              string            `json:"dir" jsonschema:"the directory to template in place"`
	Values           map[string]string `json:"values" jsonschema:"placeholder values to apply"`
	DryRun           bool              `json:"dryRun,omitempty" jsonschema:"preview only; include per-file diffs, write nothing"`
	StartDelim       string            `json:"startDelim,omitempty"`
	EndDelim         string            `json:"endDelim,omitempty"`
	IgnorePatterns   []string          `json:"ignorePatterns,omitempty"`
	ProcessTemplates bool              `json:"processTemplates,omitempty" jsonschema:"process .tpl files and drop the suffix"`
	OnlyTemplates    bool              `json:"onlyTemplates,omitempty"`
}

type cloneInput struct {
	Repo       string            `json:"repo" jsonschema:"HTTPS or SSH git URL"`
	Branch     string            `json:"branch,omitempty"`
	OutputDir  string            `json:"outputDir,omitempty" jsonschema:"where to clone; required unless dryRun"`
	Values     map[string]string `json:"values,omitempty"`
	DryRun     bool              `json:"dryRun,omitempty"`
	StartDelim string            `json:"startDelim,omitempty"`
	EndDelim   string            `json:"endDelim,omitempty"`
}

type generateInput struct {
	Template  string            `json:"template" jsonschema:"configured template name or URL substring"`
	Branch    string            `json:"branch,omitempty"`
	OutputDir string            `json:"outputDir,omitempty" jsonschema:"where to create the project; required unless dryRun"`
	Values    map[string]string `json:"values,omitempty"`
	DryRun    bool              `json:"dryRun,omitempty"`
}

type manifestInput struct {
	Dir string `json:"dir" jsonschema:"the template directory to read yankrun.yaml from"`
}

type templatesInput struct{}

// Tool output types.

type cloneOutput struct {
	WorkDir string               `json:"workDir"`
	Result  workflow.ApplyResult `json:"result"`
}

type manifestOutput struct {
	HasManifest bool             `json:"hasManifest"`
	Manifest    *domain.Manifest `json:"manifest,omitempty"`
}

type templatesOutput struct {
	Templates []domain.TemplateRepo `json:"templates"`
}

// registerTools wires every tool to its handler on the SDK server.
func (s *Server) registerTools(srv *mcpsdk.Server) {
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "yankrun_scan",
		Description: "Scan a directory for template placeholders. Returns discovered keys, per-file counts, and the yankrun.yaml manifest (if any).",
	}, s.handleScan)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "yankrun_evaluate",
		Description: "Preview how given values resolve against a directory's placeholders, including transform results, without writing anything.",
	}, s.handleEvaluate)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "yankrun_apply",
		Description: "Apply values to a directory in place. With dryRun, returns per-file unified diffs and writes nothing.",
	}, s.handleApply)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "yankrun_clone",
		Description: "Clone a git repository and apply template values to the working tree.",
	}, s.handleClone)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "yankrun_generate",
		Description: "Clone a configured template as a fresh project (removes .git) and apply values.",
	}, s.handleGenerate)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "yankrun_manifest",
		Description: "Read a template's yankrun.yaml manifest (declared variables, defaults, enums, required flags, hints).",
	}, s.handleManifest)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        "yankrun_templates",
		Description: "List configured and GitHub-discovered template repositories available to generate.",
	}, s.handleTemplates)
}

func (s *Server) handleScan(_ context.Context, _ *mcpsdk.CallToolRequest, in scanInput) (*mcpsdk.CallToolResult, workflow.Summary, error) {
	settings := s.settings(in.StartDelim, in.EndDelim, in.IgnorePatterns, in.OnlyTemplates, false)
	summary, err := s.engine.ScanDir(in.Dir, settings, domain.InputReplacement{})
	return nil, summary, err
}

func (s *Server) handleEvaluate(_ context.Context, _ *mcpsdk.CallToolRequest, in evaluateInput) (*mcpsdk.CallToolResult, workflow.Summary, error) {
	settings := s.settings(in.StartDelim, in.EndDelim, in.IgnorePatterns, false, false)
	summary, err := s.engine.ScanDir(in.Dir, settings, replacementFrom(in.Values))
	return nil, summary, err
}

func (s *Server) handleApply(_ context.Context, _ *mcpsdk.CallToolRequest, in applyInput) (*mcpsdk.CallToolResult, workflow.ApplyResult, error) {
	settings := s.settings(in.StartDelim, in.EndDelim, in.IgnorePatterns, in.OnlyTemplates, in.ProcessTemplates)
	result, err := s.engine.ApplyDir(in.Dir, settings, domain.InputReplacement{}, in.Values, in.DryRun, false)
	return nil, result, err
}

func (s *Server) handleClone(_ context.Context, _ *mcpsdk.CallToolRequest, in cloneInput) (*mcpsdk.CallToolResult, cloneOutput, error) {
	settings := s.settings(in.StartDelim, in.EndDelim, nil, false, false)
	workDir, result, err := s.engine.CloneAndApply(workflow.CloneOptions{
		Repo:      in.Repo,
		Branch:    in.Branch,
		OutputDir: in.OutputDir,
		DryRun:    in.DryRun,
	}, settings, in.Values)
	if err != nil {
		return nil, cloneOutput{}, err
	}
	return nil, cloneOutput{WorkDir: workDir, Result: result}, nil
}

func (s *Server) handleGenerate(ctx context.Context, _ *mcpsdk.CallToolRequest, in generateInput) (*mcpsdk.CallToolResult, cloneOutput, error) {
	tmpl, ok := services.FindTemplate(ctx, s.cfg, in.Template)
	if !ok {
		return nil, cloneOutput{}, fmt.Errorf("template %q not found", in.Template)
	}
	branch := in.Branch
	if branch == "" {
		branch = tmpl.DefaultBranch
	}
	if branch == "" {
		branch = "main"
	}
	settings := s.settings("", "", nil, false, false)
	workDir, result, err := s.engine.CloneAndApply(workflow.CloneOptions{
		Repo:      tmpl.URL,
		Branch:    branch,
		OutputDir: in.OutputDir,
		RemoveGit: true,
		DryRun:    in.DryRun,
	}, settings, in.Values)
	if err != nil {
		return nil, cloneOutput{}, err
	}
	return nil, cloneOutput{WorkDir: workDir, Result: result}, nil
}

func (s *Server) handleManifest(_ context.Context, _ *mcpsdk.CallToolRequest, in manifestInput) (*mcpsdk.CallToolResult, manifestOutput, error) {
	manifest, err := s.engine.LoadManifest(in.Dir)
	if err != nil {
		return nil, manifestOutput{}, err
	}
	return nil, manifestOutput{HasManifest: manifest != nil, Manifest: manifest}, nil
}

func (s *Server) handleTemplates(ctx context.Context, _ *mcpsdk.CallToolRequest, _ templatesInput) (*mcpsdk.CallToolResult, templatesOutput, error) {
	return nil, templatesOutput{Templates: services.ListTemplates(ctx, s.cfg)}, nil
}

// replacementFrom converts a value map into an InputReplacement.
func replacementFrom(values map[string]string) domain.InputReplacement {
	var in domain.InputReplacement
	for k, v := range values {
		in.Variables = append(in.Variables, domain.Replacement{Key: k, Value: v})
	}
	return in
}
