package ship

import (
	"bytes"
	"testing"
)

func TestRunGenerateModel_InvalidName(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	runner := &fakeRunner{}
	cli := CLI{Out: out, Err: errOut, Runner: runner}

	code := cli.runGenerateModel([]string{"post"})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if errOut.Len() == 0 {
		t.Fatalf("stderr should include validation error")
	}
	if len(runner.calls) != 0 {
		t.Fatalf("runner calls = %v, want none", runner.calls)
	}
}
