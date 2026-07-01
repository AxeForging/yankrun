package web

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/internal/workflow"
	"github.com/AxeForging/yankrun/services"
)

//go:embed templates/*.html.tmpl
var templatesFS embed.FS

//go:embed static
var staticFS embed.FS

type Server struct {
	addr             string
	mux              *http.ServeMux
	page             *template.Template
	mu               sync.Mutex
	dir              string
	input            string
	startDelim       string
	endDelim         string
	fileSizeLimit    string
	ignorePatterns   []string
	processTemplates bool
	onlyTemplates    bool
	forceDryRun      bool
	verbose          bool
	parser           services.ReplacementParser
	replacer         services.Replacer
	cloner           services.Cloner
	config           *domain.Config
}

type Options struct {
	Addr             string
	Dir              string
	Input            string
	StartDelim       string
	EndDelim         string
	FileSizeLimit    string
	IgnorePatterns   []string
	ProcessTemplates bool
	OnlyTemplates    bool
	ForceDryRun      bool
	Verbose          bool
	Parser           services.ReplacementParser
	Replacer         services.Replacer
	Cloner           services.Cloner
	Config           *domain.Config
}

type PlaceholderSummary struct {
	Counts map[string]int         `json:"counts"`
	Files  []workflow.FileSummary `json:"files"`
	Keys   []string               `json:"keys"`
	Values map[string]string      `json:"values"`
}

type ApplyRequest struct {
	Values map[string]string `json:"values"`
	DryRun bool              `json:"dryRun"`
}

type EvaluateRequest struct {
	Summary PlaceholderSummary `json:"summary"`
	Values  map[string]string  `json:"values"`
}

type ApplyResponse struct {
	Applied      bool               `json:"applied"`
	ForcedDryRun bool               `json:"forcedDryRun"`
	TotalMatches int                `json:"totalMatches"`
	Placeholders int                `json:"placeholders"`
	Summary      PlaceholderSummary `json:"summary"`
}

type TemplateOption struct {
	Name          string `json:"name"`
	URL           string `json:"url"`
	Description   string `json:"description"`
	DefaultBranch string `json:"defaultBranch"`
}

type CloneRequest struct {
	Repo      string            `json:"repo"`
	OutputDir string            `json:"outputDir"`
	Branch    string            `json:"branch"`
	Values    map[string]string `json:"values"`
	DryRun    bool              `json:"dryRun"`
}

type GenerateRequest struct {
	Template  string            `json:"template"`
	OutputDir string            `json:"outputDir"`
	Branch    string            `json:"branch"`
	Values    map[string]string `json:"values"`
	DryRun    bool              `json:"dryRun"`
}

type DelimitersRequest struct {
	StartDelim string `json:"startDelim"`
	EndDelim   string `json:"endDelim"`
}

func New(opts Options) (*Server, error) {
	page, err := loadTemplate()
	if err != nil {
		return nil, err
	}
	if opts.Addr == "" {
		opts.Addr = "127.0.0.1:17817"
	}
	if opts.StartDelim == "" {
		opts.StartDelim = "[["
	}
	if opts.EndDelim == "" {
		opts.EndDelim = "]]"
	}
	if opts.FileSizeLimit == "" {
		opts.FileSizeLimit = "3 mb"
	}
	s := &Server{
		addr:             opts.Addr,
		mux:              http.NewServeMux(),
		page:             page,
		dir:              opts.Dir,
		input:            opts.Input,
		startDelim:       opts.StartDelim,
		endDelim:         opts.EndDelim,
		fileSizeLimit:    opts.FileSizeLimit,
		ignorePatterns:   opts.IgnorePatterns,
		processTemplates: opts.ProcessTemplates,
		onlyTemplates:    opts.OnlyTemplates,
		forceDryRun:      opts.ForceDryRun,
		verbose:          opts.Verbose,
		parser:           opts.Parser,
		replacer:         opts.Replacer,
		cloner:           opts.Cloner,
		config:           opts.Config,
	}
	if s.config == nil {
		s.config = &domain.Config{}
	}
	s.routes()
	return s, nil
}

func (s *Server) Addr() string { return s.addr }

func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) ListenAndServe() error {
	srv := &http.Server{
		Addr:              s.addr,
		Handler:           s.mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return srv.ListenAndServe()
}

func (s *Server) ListenAndServeContext(ctx context.Context) error {
	srv := &http.Server{
		Addr:              s.addr,
		Handler:           s.mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()
	return srv.ListenAndServe()
}

func (s *Server) routes() {
	sub, _ := fs.Sub(staticFS, "static")
	s.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(sub))))
	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/api/scan", s.handleScan)
	s.mux.HandleFunc("/api/apply", s.handleApply)
	s.mux.HandleFunc("/api/evaluate", s.handleEvaluate)
	s.mux.HandleFunc("/api/templates", s.handleTemplates)
	s.mux.HandleFunc("/api/clone", s.handleClone)
	s.mux.HandleFunc("/api/generate", s.handleGenerate)
	s.mux.HandleFunc("/api/delimiters", s.handleSetDelimiters)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.mu.Lock()
	dir, startDelim, endDelim := s.dir, s.startDelim, s.endDelim
	s.mu.Unlock()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = s.page.Execute(w, map[string]any{
		"Dir":         dir,
		"StartDelim":  startDelim,
		"EndDelim":    endDelim,
		"ForceDryRun": s.forceDryRun,
	})
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	summary, err := s.Scan()
	writeJSON(w, summary, err)
}

func (s *Server) handleApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	var req ApplyRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	resp, err := s.Apply(req)
	writeJSON(w, resp, err)
}

func (s *Server) handleEvaluate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	var req EvaluateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 2<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	summary := req.Summary
	summary.Values = req.Values
	for i := range summary.Files {
		for j := range summary.Files[i].Previews {
			preview := &summary.Files[i].Previews[j]
			value, found, err := s.replacer.EvaluatePlaceholder(preview.Expression, req.Values)
			preview.Value = value
			preview.Missing = !found
			preview.Error = ""
			if err != nil {
				preview.Error = err.Error()
			}
		}
	}
	writeJSON(w, summary, nil)
}

func (s *Server) handleSetDelimiters(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	var req DelimitersRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	summary, err := s.SetDelimiters(req.StartDelim, req.EndDelim)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, summary, nil)
}

// SetDelimiters swaps the active start/end delimiters and returns a fresh scan
// of the current directory using them.
func (s *Server) SetDelimiters(start, end string) (PlaceholderSummary, error) {
	start, end, err := ValidateDelimiters(start, end)
	if err != nil {
		return PlaceholderSummary{}, err
	}
	s.mu.Lock()
	s.startDelim = start
	s.endDelim = end
	s.mu.Unlock()
	return s.Scan()
}

// ValidateDelimiters rejects delimiter pairs that would make scanning hang or
// silently corrupt results, and returns the trimmed start/end pair to use.
//
// The literal scan (walkAndAnalyzeFiles in services/replacer.go) finds each
// delimiter with strings.Index, which returns 0 without consuming any input
// for an empty needle. If both delimiters were empty the scan loop would spin
// forever on any non-empty file, hanging the request goroutine indefinitely -
// so empty (or whitespace-only) delimiters are rejected outright. Requiring
// start != end and that neither contains the other rules out the remaining
// cases where the scan and regex-based replace paths would silently disagree
// on where a placeholder begins and ends.
func ValidateDelimiters(start, end string) (string, string, error) {
	start = strings.TrimSpace(start)
	end = strings.TrimSpace(end)
	if start == "" || end == "" {
		return "", "", fmt.Errorf("start and end delimiters are required")
	}
	if start == end {
		return "", "", fmt.Errorf("start and end delimiters must be different")
	}
	if strings.Contains(start, end) || strings.Contains(end, start) {
		return "", "", fmt.Errorf("start and end delimiters must not contain each other")
	}
	return start, end, nil
}

func (s *Server) Scan() (PlaceholderSummary, error) {
	s.mu.Lock()
	dir := s.dir
	s.mu.Unlock()
	summary, err := s.scanDir(dir)
	return PlaceholderSummary(summary), err
}

func (s *Server) scanDir(dir string) (workflow.Summary, error) {
	var provided domain.InputReplacement
	if s.input != "" {
		parsed, err := s.parser.Parse(s.input)
		if err != nil {
			return workflow.Summary{}, err
		}
		provided = parsed
	}
	return s.engine().ScanDir(dir, s.settings(), provided)
}

func (s *Server) Apply(req ApplyRequest) (ApplyResponse, error) {
	s.mu.Lock()
	dir := s.dir
	s.mu.Unlock()
	result, err := s.engine().ApplyDir(dir, s.settings(), s.provided(), req.Values, req.DryRun, s.forceDryRun)
	if err != nil {
		return ApplyResponse{}, err
	}
	return applyResponseFromWorkflow(result), nil
}

func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, s.TemplateOptions(r.Context()), nil)
}

func (s *Server) handleClone(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	var req CloneRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	resp, err := s.Clone(req)
	writeJSON(w, resp, err)
}

func (s *Server) handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	var req GenerateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	resp, err := s.Generate(req)
	writeJSON(w, resp, err)
}

func (s *Server) TemplateOptions(ctx context.Context) []TemplateOption {
	out := make([]TemplateOption, 0, len(s.config.Templates))
	for _, t := range s.config.Templates {
		out = append(out, TemplateOption{Name: t.Name, URL: t.URL, Description: t.Description, DefaultBranch: t.DefaultBranch})
	}
	if s.config.GitHub.User != "" || len(s.config.GitHub.Orgs) > 0 {
		found, err := services.NewGitHubClient().ListRepos(ctx, s.config.GitHub)
		if err == nil {
			for _, r := range found {
				out = append(out, TemplateOption{Name: r.FullName, URL: r.SSHURL, Description: r.Description, DefaultBranch: r.DefaultBranch})
			}
		}
	}
	return out
}

func (s *Server) Clone(req CloneRequest) (ApplyResponse, error) {
	if strings.TrimSpace(req.Repo) == "" {
		return ApplyResponse{}, fmt.Errorf("repo is required")
	}
	return s.cloneInto(req.Repo, req.Branch, req.OutputDir, req.Values, req.DryRun, false)
}

func (s *Server) Generate(req GenerateRequest) (ApplyResponse, error) {
	if strings.TrimSpace(req.Template) == "" {
		return ApplyResponse{}, fmt.Errorf("template is required")
	}
	selected, ok := s.findTemplate(req.Template)
	if !ok {
		return ApplyResponse{}, fmt.Errorf("template not found")
	}
	branch := req.Branch
	if branch == "" {
		branch = selected.DefaultBranch
	}
	if branch == "" {
		branch = "main"
	}
	return s.cloneInto(selected.URL, branch, req.OutputDir, req.Values, req.DryRun, true)
}

func (s *Server) findTemplate(needle string) (TemplateOption, bool) {
	needle = strings.ToLower(needle)
	for _, t := range s.TemplateOptions(context.Background()) {
		if strings.ToLower(t.Name) == needle || strings.ToLower(t.URL) == needle || strings.Contains(strings.ToLower(t.Name), needle) || strings.Contains(strings.ToLower(t.URL), needle) {
			return t, true
		}
	}
	return TemplateOption{}, false
}

func (s *Server) cloneInto(repo, branch, outputDir string, values map[string]string, dryRun bool, removeGit bool) (ApplyResponse, error) {
	workDir, result, err := s.engine().CloneAndApply(workflow.CloneOptions{
		Repo:      repo,
		Branch:    branch,
		OutputDir: outputDir,
		RemoveGit: removeGit,
		DryRun:    dryRun,
		ForceDry:  s.forceDryRun,
	}, s.settings(), values)
	if err != nil {
		return ApplyResponse{}, err
	}
	if !(dryRun || s.forceDryRun) {
		s.mu.Lock()
		s.dir = workDir
		s.mu.Unlock()
	}
	return applyResponseFromWorkflow(result), nil
}

func loadTemplate() (*template.Template, error) {
	b, err := fs.ReadFile(templatesFS, "templates/workspace.html.tmpl")
	if err != nil {
		return nil, fmt.Errorf("read workspace template: %w", err)
	}
	t, err := template.New("workspace").Parse(string(b))
	if err != nil {
		return nil, fmt.Errorf("parse workspace template: %w", err)
	}
	return t, nil
}

func writeJSON(w http.ResponseWriter, payload any, err error) {
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func (s *Server) settings() workflow.TemplateSettings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return workflow.TemplateSettings{
		StartDelim:       s.startDelim,
		EndDelim:         s.endDelim,
		FileSizeLimit:    s.fileSizeLimit,
		ProcessTemplates: s.processTemplates,
		OnlyTemplates:    s.onlyTemplates,
		Verbose:          s.verbose,
		IgnorePatterns:   s.ignorePatterns,
	}
}

func (s *Server) engine() workflow.Engine {
	return workflow.Engine{Parser: s.parser, Replacer: s.replacer, Cloner: s.cloner}
}

func (s *Server) provided() domain.InputReplacement {
	if s.input == "" {
		return domain.InputReplacement{}
	}
	provided, err := s.parser.Parse(s.input)
	if err != nil {
		return domain.InputReplacement{}
	}
	return provided
}

func applyResponseFromWorkflow(result workflow.ApplyResult) ApplyResponse {
	return ApplyResponse{
		Applied:      result.Applied,
		ForcedDryRun: result.ForcedDryRun,
		TotalMatches: result.TotalMatches,
		Placeholders: result.Placeholders,
		Summary:      PlaceholderSummary(result.Summary),
	}
}
