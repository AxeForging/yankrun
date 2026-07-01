package web

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

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

func mustGetJSON(t *testing.T, url string, out any) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET %s status = %d body=%s", url, resp.StatusCode, b)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatal(err)
	}
}

func mustPostJSON(t *testing.T, url string, payload, out any) {
	t.Helper()
	buf, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST %s status = %d body=%s", url, resp.StatusCode, b)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatal(err)
	}
}

func TestValidateDelimiters(t *testing.T) {
	cases := []struct {
		name      string
		start     string
		end       string
		wantErr   bool
		errSubstr string
		wantStart string
		wantEnd   string
	}{
		{name: "valid custom pair", start: "<%", end: "%>", wantStart: "<%", wantEnd: "%>"},
		{name: "valid default pair", start: "[[", end: "]]", wantStart: "[[", wantEnd: "]]"},
		{name: "trims surrounding whitespace", start: "  <%  ", end: "  %>  ", wantStart: "<%", wantEnd: "%>"},
		{name: "regex metacharacters are treated literally", start: "(", end: ")", wantStart: "(", wantEnd: ")"},
		{name: "unicode delimiters", start: "«", end: "»", wantStart: "«", wantEnd: "»"},
		{name: "non-overlapping equal-length pair", start: "ab", end: "ba", wantStart: "ab", wantEnd: "ba"},
		{name: "both empty", start: "", end: "", wantErr: true, errSubstr: "required"},
		{name: "start empty", start: "", end: "]]", wantErr: true, errSubstr: "required"},
		{name: "end empty", start: "[[", end: "", wantErr: true, errSubstr: "required"},
		{name: "whitespace-only start counts as empty", start: "   ", end: "]]", wantErr: true, errSubstr: "required"},
		{name: "whitespace-only end counts as empty", start: "[[", end: "   ", wantErr: true, errSubstr: "required"},
		{name: "equal delimiters", start: "##", end: "##", wantErr: true, errSubstr: "different"},
		{name: "start contains end", start: "[[[", end: "[[", wantErr: true, errSubstr: "contain"},
		{name: "end contains start", start: "[[", end: "[[[", wantErr: true, errSubstr: "contain"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start, end, err := ValidateDelimiters(tc.start, tc.end)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("ValidateDelimiters(%q, %q) = nil error, want error", tc.start, tc.end)
				}
				if tc.errSubstr != "" && !strings.Contains(err.Error(), tc.errSubstr) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tc.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateDelimiters(%q, %q) unexpected error: %v", tc.start, tc.end, err)
			}
			if start != tc.wantStart || end != tc.wantEnd {
				t.Fatalf("ValidateDelimiters(%q, %q) = (%q, %q), want (%q, %q)", tc.start, tc.end, start, end, tc.wantStart, tc.wantEnd)
			}
		})
	}
}

// TestSetDelimitersRejectsEmptyWithoutHanging guards against a real bug found
// while building this feature: the literal scan in services/replacer.go finds
// delimiters with strings.Index, which returns 0 without consuming input for
// an empty needle, so an empty/empty pair spins forever on any non-empty file
// instead of erroring. SetDelimiters must reject the pair before it ever
// reaches the scanner.
func TestSetDelimitersRejectsEmptyWithoutHanging(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.txt"), []byte("Hello [[NAME]]"), 0644); err != nil {
		t.Fatal(err)
	}
	s := testServer(t, dir, false)

	done := make(chan error, 1)
	go func() {
		_, err := s.SetDelimiters("", "")
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("SetDelimiters(\"\", \"\") = nil error, want rejection")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("SetDelimiters(\"\", \"\") did not return within 2s; the empty-delimiter scan likely hung")
	}
}

func TestSetDelimitersEndpointRescansWithNewPair(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "default.txt"), []byte("Hi [[OTHER]]"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "custom.txt"), []byte("Hello <%NAME%>"), 0644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(testServer(t, dir, false).Handler())
	defer server.Close()

	var before PlaceholderSummary
	mustGetJSON(t, server.URL+"/api/scan", &before)
	if before.Counts["OTHER"] != 1 || before.Counts["NAME"] != 0 {
		t.Fatalf("before switch summary = %+v, want OTHER=1 NAME=0", before.Counts)
	}

	var after PlaceholderSummary
	mustPostJSON(t, server.URL+"/api/delimiters", DelimitersRequest{StartDelim: "<%", EndDelim: "%>"}, &after)
	if after.Counts["NAME"] != 1 || after.Counts["OTHER"] != 0 {
		t.Fatalf("after switch summary = %+v, want NAME=1 OTHER=0 (old default pair must stop matching)", after.Counts)
	}

	body := bytes.NewBufferString(`{"values":{"NAME":"World"},"dryRun":false}`)
	resp, err := http.Post(server.URL+"/api/apply", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var applied ApplyResponse
	if err := json.NewDecoder(resp.Body).Decode(&applied); err != nil {
		t.Fatal(err)
	}
	if !applied.Applied || applied.TotalMatches != 1 {
		t.Fatalf("apply response = %+v, want 1 applied match using the new delimiters", applied)
	}
	got, err := os.ReadFile(filepath.Join(dir, "custom.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "Hello World" {
		t.Fatalf("custom.txt = %q, want replaced using the new delimiters", string(got))
	}
	unrelated, err := os.ReadFile(filepath.Join(dir, "default.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(unrelated) != "Hi [[OTHER]]" {
		t.Fatalf("default.txt changed unexpectedly: %q", string(unrelated))
	}
}

func TestSetDelimitersEndpointRejectsInvalidAndLeavesStateUnchanged(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.txt"), []byte("Hello [[NAME]]"), 0644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(testServer(t, dir, false).Handler())
	defer server.Close()

	cases := []struct {
		name  string
		start string
		end   string
	}{
		{name: "both empty", start: "", end: ""},
		{name: "start empty", start: "", end: "]]"},
		{name: "end empty", start: "[[", end: ""},
		{name: "equal delimiters", start: "##", end: "##"},
		{name: "start contains end", start: "[[[", end: "[["},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf, err := json.Marshal(DelimitersRequest{StartDelim: tc.start, EndDelim: tc.end})
			if err != nil {
				t.Fatal(err)
			}
			resp, err := http.Post(server.URL+"/api/delimiters", "application/json", bytes.NewReader(buf))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", resp.StatusCode)
			}
			var errBody map[string]string
			if err := json.NewDecoder(resp.Body).Decode(&errBody); err != nil {
				t.Fatal(err)
			}
			if errBody["error"] == "" {
				t.Fatal("expected a non-empty error message")
			}

			var summary PlaceholderSummary
			mustGetJSON(t, server.URL+"/api/scan", &summary)
			if summary.Counts["NAME"] != 1 {
				t.Fatalf("after rejected change, NAME count = %d, want 1 (delimiters must stay unchanged)", summary.Counts["NAME"])
			}
		})
	}
}

func TestSetDelimitersEndpointRejectsNonPost(t *testing.T) {
	dir := t.TempDir()
	server := httptest.NewServer(testServer(t, dir, false).Handler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/delimiters")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", resp.StatusCode)
	}
}

func TestSetDelimitersEndpointRejectsMalformedJSON(t *testing.T) {
	dir := t.TempDir()
	server := httptest.NewServer(testServer(t, dir, false).Handler())
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/delimiters", "application/json", bytes.NewBufferString("{not json"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestSetDelimitersPersistsForSettingsUsedByCloneAndGenerate(t *testing.T) {
	dir := t.TempDir()
	s := testServer(t, dir, false)

	if _, err := s.SetDelimiters("<%", "%>"); err != nil {
		t.Fatal(err)
	}
	got := s.settings()
	if got.StartDelim != "<%" || got.EndDelim != "%>" {
		t.Fatalf("settings() after SetDelimiters = %+v, want StartDelim=<%% EndDelim=%%>", got)
	}
}

func TestIndexReflectsCurrentDelimitersAfterChange(t *testing.T) {
	dir := t.TempDir()
	server := httptest.NewServer(testServer(t, dir, false).Handler())
	defer server.Close()

	var summary PlaceholderSummary
	mustPostJSON(t, server.URL+"/api/delimiters", DelimitersRequest{StartDelim: "<%", EndDelim: "%>"}, &summary)

	resp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	html := string(body)
	if !strings.Contains(html, `value="&lt;%"`) {
		t.Fatalf("index page does not reflect updated start delimiter:\n%s", html)
	}
	if !strings.Contains(html, `value="%&gt;"`) {
		t.Fatalf("index page does not reflect updated end delimiter:\n%s", html)
	}
}

func TestSetDelimitersConcurrentAccessDoesNotRace(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.txt"), []byte("Hello [[NAME]] and <%NAME%>"), 0644); err != nil {
		t.Fatal(err)
	}
	s := testServer(t, dir, false)

	pairs := [][2]string{{"[[", "]]"}, {"<%", "%>"}, {"{{", "}}"}}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		pair := pairs[i%len(pairs)]
		wg.Add(2)
		go func(start, end string) {
			defer wg.Done()
			_, _ = s.SetDelimiters(start, end)
		}(pair[0], pair[1])
		go func() {
			defer wg.Done()
			_, _ = s.Scan()
		}()
	}
	wg.Wait()
}
