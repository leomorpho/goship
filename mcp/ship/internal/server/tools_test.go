package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveDocPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	tests := []struct {
		name     string
		input    string
		wantRel  string
		wantFail bool
	}{
		{name: "simple file", input: "architecture/01-architecture.md", wantRel: "architecture/01-architecture.md"},
		{name: "without extension", input: "reference/01-cli", wantRel: "reference/01-cli.md"},
		{name: "with docs prefix", input: "docs/00-index.md", wantRel: "00-index.md"},
		{name: "parent traversal", input: "../secret", wantFail: true},
		{name: "nested parent traversal", input: "architecture/../../secret", wantFail: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, gotRel, err := resolveDocPath(root, tc.input)
			if tc.wantFail {
				if err == nil {
					t.Fatalf("expected error for %q", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveDocPath(%q) error: %v", tc.input, err)
			}
			if gotRel != tc.wantRel {
				t.Fatalf("resolveDocPath(%q) rel = %q, want %q", tc.input, gotRel, tc.wantRel)
			}
		})
	}
}

func TestSearchDocs(t *testing.T) {
	t.Parallel()

	docsRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(docsRoot, "architecture"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsRoot, "00-index.md"), []byte("GoShip docs index\nShip CLI"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsRoot, "architecture", "01-architecture.md"), []byte("Runtime architecture\nship worker"), 0o644); err != nil {
		t.Fatal(err)
	}

	matches, err := searchDocs(docsRoot, "ship", 10)
	if err != nil {
		t.Fatalf("searchDocs error: %v", err)
	}
	if len(matches) < 2 {
		t.Fatalf("expected at least 2 matches, got %d", len(matches))
	}
}

func TestHandleToolsCall(t *testing.T) {
	t.Parallel()

	docsRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(docsRoot, "reference"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsRoot, "reference", "01-cli.md"), []byte("ship dev\nship test"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &mcpServer{docsRoot: docsRoot}

	tests := []struct {
		name     string
		method   string
		args     any
		wantText string
		wantErr  bool
	}{
		{name: "ship_help general", method: "ship_help", args: map[string]any{"topic": "general"}, wantText: "ship - GoShip CLI"},
		{name: "docs_get", method: "docs_get", args: map[string]any{"path": "reference/01-cli.md"}, wantText: "# reference/01-cli.md"},
		{name: "docs_search", method: "docs_search", args: map[string]any{"query": "ship", "limit": 5}, wantText: "Matches for \"ship\""},
		{name: "unknown", method: "nope", args: map[string]any{}, wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			argsJSON, err := json.Marshal(tc.args)
			if err != nil {
				t.Fatal(err)
			}
			paramsJSON, err := json.Marshal(toolCallParams{Name: tc.method, Arguments: argsJSON})
			if err != nil {
				t.Fatal(err)
			}

			res, err := s.handleToolsCall(paramsJSON)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("handleToolsCall error: %v", err)
			}
			if len(res.Content) == 0 {
				t.Fatalf("expected content")
			}
			if !strings.Contains(res.Content[0].Text, tc.wantText) {
				t.Fatalf("response %q does not contain %q", res.Content[0].Text, tc.wantText)
			}
		})
	}
}
