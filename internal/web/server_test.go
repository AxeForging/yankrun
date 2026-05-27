package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/services"
)

type fakeCloner struct {
	lastRepo   string
	lastBranch string
}

func (f *fakeCloner) CloneRepository(repoURL, outputDir string) error {
	f.lastRepo = repoURL
	return os.WriteFile(filepath.Join(outputDir, "app.txt"), []byte("Hello [[NAME]]"), 0644)
}

func (f *fakeCloner) CloneRepositoryBranch(repoURL, branch, outputDir string) error {
	f.lastRepo = repoURL
	f.lastBranch = branch
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(outputDir, ".git"), 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outputDir, "app.txt"), []byte(fmt.Sprintf("%s [[NAME]]", branch)), 0644)
}

func (f *fakeCloner) ListRemoteBranches(repoURL string) ([]string, error) {
	f.lastRepo = repoURL
	return []string{"main", "feature"}, nil
}

func (f *fakeCloner) SetSSHKeyPath(path string) {}

func testServer(t *testing.T, dir string, forceDryRun bool) *Server {
	t.Helper()
	parser := &services.YAMLJSONParser{FileSystem: &services.OsFileSystem{}}
	replacer := &services.FileReplacer{FileSystem: &services.OsFileSystem{}}
	s, err := New(Options{
		Addr:          "127.0.0.1:0",
		Dir:           dir,
		StartDelim:    "[[",
		EndDelim:      "]]",
		FileSizeLimit: "3 mb",
		ForceDryRun:   forceDryRun,
		Parser:        parser,
		Replacer:      replacer,
		Cloner:        &fakeCloner{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestScanAndDryRun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.txt")
	if err := os.WriteFile(path, []byte("Hello [[NAME:toUpperCase]]"), 0644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(testServer(t, dir, false).Handler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/scan")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("scan status = %d", resp.StatusCode)
	}
	var summary PlaceholderSummary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		t.Fatal(err)
	}
	if summary.Counts["NAME"] != 1 {
		t.Fatalf("NAME count = %d, want 1", summary.Counts["NAME"])
	}
	if len(summary.Files) != 1 || summary.Files[0].Path != "app.txt" || summary.Files[0].Counts["NAME"] != 1 {
		t.Fatalf("summary files = %+v, want app.txt with NAME count", summary.Files)
	}

	body := bytes.NewBufferString(`{"values":{"NAME":"World"},"dryRun":true}`)
	resp, err = http.Post(server.URL+"/api/apply", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var applied ApplyResponse
	if err := json.NewDecoder(resp.Body).Decode(&applied); err != nil {
		t.Fatal(err)
	}
	if applied.Applied {
		t.Fatal("dry-run apply should not modify files")
	}
	if len(applied.Summary.Files) != 1 || len(applied.Summary.Files[0].Previews) != 1 {
		t.Fatalf("preview summary files = %+v, want evaluated preview", applied.Summary.Files)
	}
	preview := applied.Summary.Files[0].Previews[0]
	if preview.Expression != "NAME:toUpperCase" || preview.Value != "WORLD" || preview.Missing || preview.Error != "" {
		t.Fatalf("preview = %+v, want NAME:toUpperCase -> WORLD", preview)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "Hello [[NAME:toUpperCase]]" {
		t.Fatalf("file changed during dry run: %q", string(got))
	}
}

func TestEvaluateUpdatesPreviewValues(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.txt"), []byte("Env [[ENVIRONMENT|default:dev]] App [[APP_NAME:toUpperCase]]"), 0644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(testServer(t, dir, false).Handler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/scan")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var summary PlaceholderSummary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		t.Fatal(err)
	}
	if summary.Counts["ENVIRONMENT"] != 1 {
		t.Fatalf("ENVIRONMENT count = %d, want 1; summary=%+v", summary.Counts["ENVIRONMENT"], summary)
	}

	body, err := json.Marshal(EvaluateRequest{
		Summary: summary,
		Values:  map[string]string{"ENVIRONMENT": "prod", "APP_NAME": "demo"},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err = http.Post(server.URL+"/api/evaluate", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var evaluated PlaceholderSummary
	if err := json.NewDecoder(resp.Body).Decode(&evaluated); err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, file := range evaluated.Files {
		for _, preview := range file.Previews {
			got[preview.Expression] = preview.Value
		}
	}
	if got["ENVIRONMENT|default:dev"] != "prod" || got["APP_NAME:toUpperCase"] != "DEMO" {
		t.Fatalf("evaluated previews = %+v", got)
	}
}

func TestApplyAndForcedDryRun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.txt")
	if err := os.WriteFile(path, []byte("Hello [[NAME]]"), 0644); err != nil {
		t.Fatal(err)
	}

	forced := httptest.NewServer(testServer(t, dir, true).Handler())
	body := bytes.NewBufferString(`{"values":{"NAME":"Blocked"},"dryRun":false}`)
	resp, err := http.Post(forced.URL+"/api/apply", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	var forcedResp ApplyResponse
	if err := json.NewDecoder(resp.Body).Decode(&forcedResp); err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	forced.Close()
	if forcedResp.Applied || !forcedResp.ForcedDryRun {
		t.Fatalf("forced dry-run response = %+v", forcedResp)
	}

	live := httptest.NewServer(testServer(t, dir, false).Handler())
	defer live.Close()
	body = bytes.NewBufferString(`{"values":{"NAME":"World"},"dryRun":false}`)
	resp, err = http.Post(live.URL+"/api/apply", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var liveResp ApplyResponse
	if err := json.NewDecoder(resp.Body).Decode(&liveResp); err != nil {
		t.Fatal(err)
	}
	if !liveResp.Applied {
		t.Fatalf("expected live apply, got %+v", liveResp)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "Hello World" {
		t.Fatalf("file = %q, want applied value", string(got))
	}
}

func TestCloneAndGenerateUseSharedWorkflow(t *testing.T) {
	dir := t.TempDir()
	cloner := &fakeCloner{}
	parser := &services.YAMLJSONParser{FileSystem: &services.OsFileSystem{}}
	replacer := &services.FileReplacer{FileSystem: &services.OsFileSystem{}}
	s, err := New(Options{
		Addr:          "127.0.0.1:0",
		Dir:           dir,
		StartDelim:    "[[",
		EndDelim:      "]]",
		FileSizeLimit: "3 mb",
		Parser:        parser,
		Replacer:      replacer,
		Cloner:        cloner,
		Config: &domain.Config{Templates: []domain.TemplateRepo{
			{Name: "starter", URL: "git@example.com:org/starter.git", DefaultBranch: "feature"},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}

	cloneOut := filepath.Join(t.TempDir(), "clone")
	cloneResp, err := s.Clone(CloneRequest{
		Repo:      "https://example.com/repo.git",
		Branch:    "main",
		OutputDir: cloneOut,
		Values:    map[string]string{"NAME": "World"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !cloneResp.Applied || cloner.lastRepo != "https://example.com/repo.git" || cloner.lastBranch != "main" {
		t.Fatalf("clone response=%+v cloner=%+v", cloneResp, cloner)
	}
	got, err := os.ReadFile(filepath.Join(cloneOut, "app.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "main World" {
		t.Fatalf("clone output = %q", string(got))
	}

	generateOut := filepath.Join(t.TempDir(), "generate")
	generateResp, err := s.Generate(GenerateRequest{
		Template:  "starter",
		OutputDir: generateOut,
		Values:    map[string]string{"NAME": "Team"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !generateResp.Applied || cloner.lastRepo != "git@example.com:org/starter.git" || cloner.lastBranch != "feature" {
		t.Fatalf("generate response=%+v cloner=%+v", generateResp, cloner)
	}
	if _, err := os.Stat(filepath.Join(generateOut, ".git")); !os.IsNotExist(err) {
		t.Fatalf("generate should remove .git, stat err=%v", err)
	}
	got, err = os.ReadFile(filepath.Join(generateOut, "app.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "feature Team" {
		t.Fatalf("generate output = %q", string(got))
	}
}
