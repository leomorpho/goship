package server

import (
	"encoding/json"
	"errors"
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
	docsRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(docsRoot, "reference"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docsRoot, "reference", "01-cli.md"), []byte("ship dev\nship test"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &mcpServer{docsRoot: docsRoot, repoRoot: docsRoot}
	prevLookPath := lookPathShip
	prevRunShip := runShipJSON
	t.Cleanup(func() {
		lookPathShip = prevLookPath
		runShipJSON = prevRunShip
	})

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

func TestCallShipDoctor(t *testing.T) {
	docsRoot := t.TempDir()
	s := &mcpServer{docsRoot: docsRoot, repoRoot: docsRoot}

	t.Run("returns ship doctor payload", func(t *testing.T) {
		prevLookPath := lookPathShip
		prevRunShip := runShipJSON
		t.Cleanup(func() {
			lookPathShip = prevLookPath
			runShipJSON = prevRunShip
		})
		lookPathShip = func(file string) (string, error) { return "/usr/bin/ship", nil }
		runShipJSON = func(name string, args ...string) ([]byte, error) {
			return []byte(`{"ok":true,"issues":[]}`), nil
		}

		res, err := s.callShipDoctor(json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("callShipDoctor error: %v", err)
		}
		if len(res.Content) != 1 || res.Content[0].Text != `{"ok":true,"issues":[]}` {
			t.Fatalf("content = %+v, want doctor json", res.Content)
		}
	})

	t.Run("missing ship binary returns config issue", func(t *testing.T) {
		prevLookPath := lookPathShip
		prevRunShip := runShipJSON
		t.Cleanup(func() {
			lookPathShip = prevLookPath
			runShipJSON = prevRunShip
		})
		lookPathShip = func(file string) (string, error) { return "", errors.New("missing") }

		res, err := s.callShipDoctor(json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("callShipDoctor error: %v", err)
		}
		if !strings.Contains(res.Content[0].Text, `"ship binary not found in PATH"`) {
			t.Fatalf("content = %q, want missing ship message", res.Content[0].Text)
		}
	})

	t.Run("invalid json output returns config issue", func(t *testing.T) {
		prevLookPath := lookPathShip
		prevRunShip := runShipJSON
		t.Cleanup(func() {
			lookPathShip = prevLookPath
			runShipJSON = prevRunShip
		})
		lookPathShip = func(file string) (string, error) { return "/usr/bin/ship", nil }
		runShipJSON = func(name string, args ...string) ([]byte, error) {
			return []byte("not-json"), nil
		}

		res, err := s.callShipDoctor(json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("callShipDoctor error: %v", err)
		}
		if !strings.Contains(res.Content[0].Text, `"invalid ship doctor JSON output: not-json"`) {
			t.Fatalf("content = %q, want invalid json message", res.Content[0].Text)
		}
	})
}

func TestCallShipRoutesAndModules(t *testing.T) {
	docsRoot := t.TempDir()
	s := &mcpServer{docsRoot: docsRoot, repoRoot: docsRoot}

	baseDescribeJSON := `{"routes":[{"method":"GET","path":"/","handler":"landingPage.Get","auth":false,"file":"app/router.go:1"},{"method":"GET","path":"/auth","handler":"home.Get","auth":true,"file":"app/router.go:2"}],"modules":[{"id":"notifications","installed":true,"routes":0,"migrations":1}]}`

	t.Run("ship_routes returns filtered routes", func(t *testing.T) {
		prevLookPath := lookPathShip
		prevRunShip := runShipJSON
		t.Cleanup(func() {
			lookPathShip = prevLookPath
			runShipJSON = prevRunShip
		})
		lookPathShip = func(file string) (string, error) { return "/usr/bin/ship", nil }
		runShipJSON = func(name string, args ...string) ([]byte, error) {
			return []byte(baseDescribeJSON), nil
		}

		res, err := s.callShipRoutes(json.RawMessage(`{"filter":"auth"}`))
		if err != nil {
			t.Fatalf("callShipRoutes error: %v", err)
		}
		if res.IsError {
			t.Fatalf("expected non-error result, got %+v", res)
		}
		if !strings.Contains(res.Content[0].Text, `"/auth"`) || strings.Contains(res.Content[0].Text, `"path":"/"`) {
			t.Fatalf("content = %q, want only auth route", res.Content[0].Text)
		}
	})

	t.Run("ship_modules returns modules", func(t *testing.T) {
		prevLookPath := lookPathShip
		prevRunShip := runShipJSON
		t.Cleanup(func() {
			lookPathShip = prevLookPath
			runShipJSON = prevRunShip
		})
		lookPathShip = func(file string) (string, error) { return "/usr/bin/ship", nil }
		runShipJSON = func(name string, args ...string) ([]byte, error) {
			return []byte(baseDescribeJSON), nil
		}

		res, err := s.callShipModules(json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("callShipModules error: %v", err)
		}
		if res.IsError {
			t.Fatalf("expected non-error result, got %+v", res)
		}
		if !strings.Contains(res.Content[0].Text, `"notifications"`) {
			t.Fatalf("content = %q, want module payload", res.Content[0].Text)
		}
	})

	t.Run("ship_routes missing binary returns empty error payload", func(t *testing.T) {
		prevLookPath := lookPathShip
		prevRunShip := runShipJSON
		t.Cleanup(func() {
			lookPathShip = prevLookPath
			runShipJSON = prevRunShip
		})
		lookPathShip = func(file string) (string, error) { return "", errors.New("missing") }

		res, err := s.callShipRoutes(json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("callShipRoutes error: %v", err)
		}
		if !res.IsError || res.Content[0].Text != `{"routes":[]}` {
			t.Fatalf("result = %+v, want empty error payload", res)
		}
	})
}

func TestCallShipVerify(t *testing.T) {
	docsRoot := t.TempDir()
	s := &mcpServer{docsRoot: docsRoot, repoRoot: docsRoot}

	t.Run("returns ship verify payload", func(t *testing.T) {
		prevLookPath := lookPathShip
		prevRunShip := runShipJSON
		t.Cleanup(func() {
			lookPathShip = prevLookPath
			runShipJSON = prevRunShip
		})
		lookPathShip = func(file string) (string, error) { return "/usr/bin/ship", nil }
		runShipJSON = func(name string, args ...string) ([]byte, error) {
			if got := strings.Join(args, " "); got != "verify --json --skip-tests" {
				t.Fatalf("args = %q, want verify --json --skip-tests", got)
			}
			return []byte(`{"ok":true,"steps":[{"name":"go test ./...","ok":true,"output":"skipped via --skip-tests"}]}`), nil
		}

		res, err := s.callShipVerify(json.RawMessage(`{"skip_tests":true}`))
		if err != nil {
			t.Fatalf("callShipVerify error: %v", err)
		}
		if len(res.Content) != 1 || !strings.Contains(res.Content[0].Text, `"skipped via --skip-tests"`) {
			t.Fatalf("content = %+v, want verify json", res.Content)
		}
	})

	t.Run("missing ship binary returns structured failure", func(t *testing.T) {
		prevLookPath := lookPathShip
		prevRunShip := runShipJSON
		t.Cleanup(func() {
			lookPathShip = prevLookPath
			runShipJSON = prevRunShip
		})
		lookPathShip = func(file string) (string, error) { return "", errors.New("missing") }

		res, err := s.callShipVerify(json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("callShipVerify error: %v", err)
		}
		if !strings.Contains(res.Content[0].Text, `"ship binary not found in PATH"`) {
			t.Fatalf("content = %q, want missing ship message", res.Content[0].Text)
		}
	})

	t.Run("invalid json output returns structured failure", func(t *testing.T) {
		prevLookPath := lookPathShip
		prevRunShip := runShipJSON
		t.Cleanup(func() {
			lookPathShip = prevLookPath
			runShipJSON = prevRunShip
		})
		lookPathShip = func(file string) (string, error) { return "/usr/bin/ship", nil }
		runShipJSON = func(name string, args ...string) ([]byte, error) {
			return []byte("not-json"), nil
		}

		res, err := s.callShipVerify(json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("callShipVerify error: %v", err)
		}
		if !strings.Contains(res.Content[0].Text, `"invalid ship verify JSON output: not-json"`) {
			t.Fatalf("content = %q, want invalid json message", res.Content[0].Text)
		}
	})
}

func TestCallShipScaffold(t *testing.T) {
	docsRoot := t.TempDir()
	s := &mcpServer{docsRoot: docsRoot, repoRoot: docsRoot}

	t.Run("success", func(t *testing.T) {
		prevLookPath := lookPathShip
		prevRunShip := runShipJSON
		prevRunGit := runGitStatus
		t.Cleanup(func() {
			lookPathShip = prevLookPath
			runShipJSON = prevRunShip
			runGitStatus = prevRunGit
		})

		lookPathShip = func(file string) (string, error) { return "/usr/bin/ship", nil }
		var usedArgs []string
		runShipJSON = func(name string, args ...string) ([]byte, error) {
			usedArgs = args
			return []byte("done"), nil
		}
		statusCalls := 0
		runGitStatus = func(dir string) (map[string]string, error) {
			statusCalls++
			if statusCalls == 1 {
				return map[string]string{"README.md": "??"}, nil
			}
			return map[string]string{
				"README.md":                   "??",
				"app/web/controllers/posts.go": "??",
			}, nil
		}

		res, err := s.callShipScaffold(json.RawMessage(`{"resource":"Post","fields":[{"name":"Title","type":"string"}]}`))
		if err != nil {
			t.Fatalf("callShipScaffold error: %v", err)
		}
		if res.IsError {
			t.Fatalf("unexpected IsError")
		}
		if len(usedArgs) != 3 || usedArgs[0] != "make:scaffold" || usedArgs[1] != "Post" || usedArgs[2] != "title:string" {
			t.Fatalf("args = %v, want make:scaffold Post title:string", usedArgs)
		}

		var payload shipScaffoldResult
		if len(res.Content) == 0 {
			t.Fatalf("missing content")
		}
		if err := json.Unmarshal([]byte(res.Content[0].Text), &payload); err != nil {
			t.Fatalf("unmarshal payload: %v", err)
		}
		if !payload.OK {
			t.Fatalf("expected ok true")
		}
		if len(payload.FilesCreated) != 1 || payload.FilesCreated[0] != "app/web/controllers/posts.go" {
			t.Fatalf("files = %v, want posts controller", payload.FilesCreated)
		}
		if len(payload.Errors) != 0 {
			t.Fatalf("unexpected errors: %v", payload.Errors)
		}
	})

	t.Run("missing resource", func(t *testing.T) {
		_, err := s.callShipScaffold(json.RawMessage(`{"fields":[]}`))
		if err == nil {
			t.Fatalf("expected error for missing resource")
		}
	})

	t.Run("missing ship binary", func(t *testing.T) {
		prevLookPath := lookPathShip
		prevRunGit := runGitStatus
		t.Cleanup(func() {
			lookPathShip = prevLookPath
			runGitStatus = prevRunGit
		})

		lookPathShip = func(file string) (string, error) { return "", errors.New("no ship") }
		runGitStatus = func(dir string) (map[string]string, error) {
			return map[string]string{}, nil
		}

		res, err := s.callShipScaffold(json.RawMessage(`{"resource":"Post","fields":[]}`))
		if err != nil {
			t.Fatalf("callShipScaffold error: %v", err)
		}
		if len(res.Content) == 0 {
			t.Fatalf("missing content")
		}
		var payload shipScaffoldResult
		if err := json.Unmarshal([]byte(res.Content[0].Text), &payload); err != nil {
			t.Fatalf("unmarshal missing binary payload: %v", err)
		}
		if payload.OK {
			t.Fatalf("expected ok false")
		}
		if !res.IsError {
			t.Fatalf("expected IsError true")
		}
		if len(payload.Errors) == 0 || !strings.Contains(payload.Errors[0], "ship binary not found") {
			t.Fatalf("errors = %v, want ship binary message", payload.Errors)
		}
	})
}
