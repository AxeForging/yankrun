package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanIncludesManifest(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "yankrun.yaml"), []byte("version: 1\nvariables:\n  - key: NAME\n    required: true\n    enum: [a, b]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "app.txt"), []byte("Hi [[NAME]]"), 0o644); err != nil {
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
	if summary.Manifest == nil {
		t.Fatal("scan payload missing manifest")
	}
	v := summary.Manifest.Variable("NAME")
	if v == nil || !v.Required || len(v.Enum) != 2 {
		t.Fatalf("manifest variable = %+v, want required with 2 enum options", v)
	}
}

func TestDryRunApplyIncludesDiff(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.txt"), []byte("name=[[NAME]]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(testServer(t, dir, false).Handler())
	defer server.Close()

	resp, err := http.Post(server.URL+"/api/apply", "application/json",
		bytes.NewBufferString(`{"values":{"NAME":"demo"},"dryRun":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var applied ApplyResponse
	if err := json.NewDecoder(resp.Body).Decode(&applied); err != nil {
		t.Fatal(err)
	}
	if applied.Applied {
		t.Fatal("dry run must not apply")
	}
	if len(applied.Summary.Files) == 0 || !strings.Contains(applied.Summary.Files[0].Diff, "+name=demo") {
		t.Fatalf("expected dry-run diff, got %+v", applied.Summary.Files)
	}
}

func TestStaticBrandAssetsServed(t *testing.T) {
	server := httptest.NewServer(testServer(t, t.TempDir(), false).Handler())
	defer server.Close()

	cases := []struct{ path, wantType string }{
		{"/static/axeforge.css", "text/css"},
		{"/static/style.css", "text/css"},
		{"/static/favicon.svg", "image/svg"},
		{"/static/fonts/inter-400.woff2", "font/woff2"},
		{"/static/fonts/jetbrains-mono-400.woff2", "font/woff2"},
	}
	for _, tc := range cases {
		resp, err := http.Get(server.URL + tc.path)
		if err != nil {
			t.Fatalf("%s: %v", tc.path, err)
		}
		body, _ := readAll(resp)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s: status %d", tc.path, resp.StatusCode)
		}
		if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, tc.wantType) {
			t.Fatalf("%s: content-type %q, want %q", tc.path, ct, tc.wantType)
		}
		if tc.path == "/static/axeforge.css" {
			s := string(body)
			if !strings.Contains(s, "--af-bg") {
				t.Fatalf("axeforge.css missing brand tokens")
			}
			if strings.Contains(s, "fonts.googleapis.com") {
				t.Fatalf("axeforge.css must self-host fonts (no CDN import) for offline serve")
			}
		}
	}
}

func TestIndexUsesBrandShell(t *testing.T) {
	server := httptest.NewServer(testServer(t, t.TempDir(), false).Handler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := readAll(resp)
	html := string(body)
	for _, want := range []string{`data-flavor="forge"`, `content="#05070a"`, "/static/axeforge.css", "class=\"topbar\""} {
		if !strings.Contains(html, want) {
			t.Fatalf("index missing %q", want)
		}
	}
}

func readAll(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(resp.Body)
	return buf.Bytes(), err
}
