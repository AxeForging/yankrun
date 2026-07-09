package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/AxeForging/yankrun/domain"
	"github.com/AxeForging/yankrun/services"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// connect wires an in-memory client to the yankrun MCP server and returns the
// client session. It shuts down on test cleanup.
func connect(t *testing.T, cfg *domain.Config) *mcpsdk.ClientSession {
	t.Helper()
	fs := &services.OsFileSystem{}
	srv := New(
		&services.YAMLJSONParser{FileSystem: fs},
		&services.FileReplacer{FileSystem: fs},
		&services.GitCloner{FileSystem: fs},
		cfg, "test",
	).sdkServer()

	serverT, clientT := mcpsdk.NewInMemoryTransports()
	ctx := context.Background()
	ss, err := srv.Connect(ctx, serverT, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { _ = ss.Close() })

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client", Version: "0"}, nil)
	cs, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

// callInto invokes a tool and decodes its structured output into v.
func callInto(t *testing.T, cs *mcpsdk.ClientSession, name string, args map[string]any, v any) *mcpsdk.CallToolResult {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	if v != nil && !res.IsError {
		decodeResult(t, res, v)
	}
	return res
}

func decodeResult(t *testing.T, res *mcpsdk.CallToolResult, v any) {
	t.Helper()
	if res.StructuredContent != nil {
		b, _ := json.Marshal(res.StructuredContent)
		if err := json.Unmarshal(b, v); err != nil {
			t.Fatalf("decode structured: %v", err)
		}
		return
	}
	for _, c := range res.Content {
		if tc, ok := c.(*mcpsdk.TextContent); ok {
			if err := json.Unmarshal([]byte(tc.Text), v); err != nil {
				t.Fatalf("decode text content: %v\n%s", err, tc.Text)
			}
			return
		}
	}
	t.Fatal("no decodable content in result")
}

func TestMCP_ListTools(t *testing.T) {
	cs := connect(t, &domain.Config{})
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	want := map[string]bool{
		"yankrun_scan": true, "yankrun_evaluate": true, "yankrun_apply": true,
		"yankrun_clone": true, "yankrun_generate": true, "yankrun_manifest": true,
		"yankrun_templates": true,
	}
	for _, tool := range res.Tools {
		delete(want, tool.Name)
	}
	if len(want) != 0 {
		t.Fatalf("missing tools: %v", want)
	}
}

func TestMCP_ScanAndManifest(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "yankrun.yaml", "version: 1\nvariables:\n  - key: APP_NAME\n    required: true\n")
	writeFile(t, dir, "app.txt", "name=[[APP_NAME]]\n")

	cs := connect(t, &domain.Config{})

	var scan struct {
		Keys     []string         `json:"keys"`
		Manifest *domain.Manifest `json:"manifest"`
	}
	callInto(t, cs, "yankrun_scan", map[string]any{"dir": dir}, &scan)
	if len(scan.Keys) != 1 || scan.Keys[0] != "APP_NAME" {
		t.Fatalf("unexpected keys: %v", scan.Keys)
	}
	if scan.Manifest == nil || scan.Manifest.Variable("APP_NAME") == nil {
		t.Fatalf("manifest not surfaced: %+v", scan.Manifest)
	}

	var man struct {
		HasManifest bool             `json:"hasManifest"`
		Manifest    *domain.Manifest `json:"manifest"`
	}
	callInto(t, cs, "yankrun_manifest", map[string]any{"dir": dir}, &man)
	if !man.HasManifest || man.Manifest == nil {
		t.Fatalf("manifest tool: %+v", man)
	}
}

func TestMCP_ApplyDryRunThenReal(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "app.txt", "name=[[APP_NAME]]\n")
	cs := connect(t, &domain.Config{})

	// Dry run: returns a diff, writes nothing.
	var dry struct {
		Applied bool `json:"applied"`
		Summary struct {
			Files []struct {
				Diff string `json:"diff"`
			} `json:"files"`
		} `json:"summary"`
	}
	callInto(t, cs, "yankrun_apply", map[string]any{
		"dir":    dir,
		"values": map[string]string{"APP_NAME": "demo"},
		"dryRun": true,
	}, &dry)
	if dry.Applied {
		t.Fatal("dry run must not apply")
	}
	if len(dry.Summary.Files) == 0 || dry.Summary.Files[0].Diff == "" {
		t.Fatalf("expected a diff, got %+v", dry.Summary.Files)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "app.txt")); string(got) != "name=[[APP_NAME]]\n" {
		t.Fatalf("dry run modified file: %q", string(got))
	}

	// Real apply: writes the value.
	var real struct {
		Applied      bool `json:"applied"`
		TotalMatches int  `json:"totalMatches"`
	}
	callInto(t, cs, "yankrun_apply", map[string]any{
		"dir":    dir,
		"values": map[string]string{"APP_NAME": "demo"},
	}, &real)
	if !real.Applied || real.TotalMatches != 1 {
		t.Fatalf("apply result: %+v", real)
	}
	if got, _ := os.ReadFile(filepath.Join(dir, "app.txt")); string(got) != "name=demo\n" {
		t.Fatalf("apply wrote %q", string(got))
	}
}

func TestMCP_GenerateUnknownTemplateIsError(t *testing.T) {
	cs := connect(t, &domain.Config{})
	res := callInto(t, cs, "yankrun_generate", map[string]any{
		"template":  "does-not-exist",
		"outputDir": t.TempDir(),
	}, nil)
	if !res.IsError {
		t.Fatal("expected an error result for unknown template")
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
