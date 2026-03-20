package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestRunRoutes(t *testing.T) {
	root := t.TempDir()
	writeDescribeFixture(t, root)

	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWD) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	t.Run("table output", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		if code := RunRoutes([]string{}, RoutesDeps{Out: out, Err: errOut, FindGoModule: findDescribeGoModule}); code != 0 {
			t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
		}
		if errOut.Len() != 0 {
			t.Fatalf("stderr = %q, want empty", errOut.String())
		}
		text := out.String()
		if !strings.Contains(text, "METHOD") || !strings.Contains(text, "PATH") || !strings.Contains(text, "AUTH") || !strings.Contains(text, "HANDLER") {
			t.Fatalf("stdout = %q, want table header", text)
		}
		if !strings.Contains(text, "GET") || !strings.Contains(text, "/login") || !strings.Contains(text, "public") {
			t.Fatalf("stdout = %q, want public route row", text)
		}
		if !strings.Contains(text, "POST") || !strings.Contains(text, "auth") {
			t.Fatalf("stdout = %q, want auth route row", text)
		}
	})

	t.Run("json output", func(t *testing.T) {
		out := &bytes.Buffer{}
		errOut := &bytes.Buffer{}
		if code := RunRoutes([]string{"--json"}, RoutesDeps{Out: out, Err: errOut, FindGoModule: findDescribeGoModule}); code != 0 {
			t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
		}
		if errOut.Len() != 0 {
			t.Fatalf("stderr = %q, want empty", errOut.String())
		}

		var payload []routeRow
		if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
			t.Fatalf("decode json: %v", err)
		}
		if len(payload) != 3 {
			t.Fatalf("routes len = %d, want 3", len(payload))
		}
		if payload[0].Method == "" || payload[0].Path == "" || payload[0].Handler == "" {
			t.Fatalf("first route = %+v, want populated fields", payload[0])
		}
		foundAuth := false
		for _, route := range payload {
			if route.Auth == "auth" {
				foundAuth = true
				break
			}
		}
		if !foundAuth {
			t.Fatalf("payload = %+v, want auth route", payload)
		}
	})

	t.Run("help", func(t *testing.T) {
		out := &bytes.Buffer{}
		if code := RunRoutes([]string{"--help"}, RoutesDeps{Out: out, Err: &bytes.Buffer{}, FindGoModule: findDescribeGoModule}); code != 0 {
			t.Fatalf("exit code = %d, want 0", code)
		}
		if !strings.Contains(out.String(), "ship routes commands:") {
			t.Fatalf("stdout = %q, want help", out.String())
		}
	})
}
