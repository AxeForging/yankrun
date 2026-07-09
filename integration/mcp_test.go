package integration

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestMCPStdioHandshake execs the real `yankrun mcp` binary, performs the MCP
// initialize handshake over stdio, and drives a scan tool call end-to-end.
func TestMCPStdioHandshake(t *testing.T) {
	bin := buildBinary(t)
	dir := t.TempDir()
	writeFile(t, dir, "app.txt", "name=[[APP_NAME]]\n")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	cmd := exec.Command(bin, "mcp")
	cmd.Dir = repoRoot(t)
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "yankrun-it", Version: "0"}, nil)
	cs, err := client.Connect(ctx, &mcpsdk.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("connect to mcp: %v", err)
	}
	defer cs.Close()

	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools.Tools) < 7 {
		t.Fatalf("expected >= 7 tools, got %d", len(tools.Tools))
	}

	res, err := cs.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      "yankrun_scan",
		Arguments: map[string]any{"dir": dir},
	})
	if err != nil {
		t.Fatalf("call scan: %v", err)
	}
	if res.IsError {
		t.Fatalf("scan returned error result: %+v", res.Content)
	}

	var scan struct {
		Keys []string `json:"keys"`
	}
	decodeStructured(t, res, &scan)
	if len(scan.Keys) != 1 || scan.Keys[0] != "APP_NAME" {
		t.Fatalf("unexpected scan keys: %v", scan.Keys)
	}
}

func decodeStructured(t *testing.T, res *mcpsdk.CallToolResult, v any) {
	t.Helper()
	if res.StructuredContent != nil {
		b, _ := json.Marshal(res.StructuredContent)
		if err := json.Unmarshal(b, v); err == nil {
			return
		}
	}
	for _, c := range res.Content {
		if tc, ok := c.(*mcpsdk.TextContent); ok {
			if err := json.Unmarshal([]byte(tc.Text), v); err != nil {
				t.Fatalf("decode content: %v", err)
			}
			return
		}
	}
	t.Fatal("no decodable content")
}
