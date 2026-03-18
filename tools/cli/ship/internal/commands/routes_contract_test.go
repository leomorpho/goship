package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestRouteGroupContract_RedSpec(t *testing.T) {
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

	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	if code := RunRoutes([]string{"--json"}, RoutesDeps{Out: out, Err: errOut, FindGoModule: findDescribeGoModule}); code != 0 {
		t.Fatalf("exit code = %d, stderr=%s", code, errOut.String())
	}

	var payload []routeRow
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("decode json: %v", err)
	}

	want := []routeRow{
		{Method: "GET", Path: `"/login"`, Auth: "public", Handler: "login.Get"},
		{Method: "POST", Path: `"/login"`, Auth: "auth", Handler: "login.Post"},
		{Method: "DELETE", Path: `"/logout"`, Auth: "public", Handler: "login.Delete"},
	}

	for _, expected := range want {
		matched := false
		for _, actual := range payload {
			if actual.Method != expected.Method || actual.Path != expected.Path || actual.Auth != expected.Auth {
				continue
			}
			if actual.Handler != expected.Handler {
				t.Fatalf("route %s %s = %+v, want handler %q", expected.Method, expected.Path, actual, expected.Handler)
			}
			if actual.File == "" {
				t.Fatalf("route %s %s should include source file metadata", expected.Method, expected.Path)
			}
			matched = true
			break
		}
		if !matched {
			t.Fatalf("missing route contract %+v in %+v", expected, payload)
		}
	}
}
