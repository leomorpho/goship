package generators

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRunGenerateEvent(t *testing.T) {
	dir := t.TempDir()
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	code := RunGenerateEvent([]string{"UserLoggedIn"}, GenerateEventDeps{
		Out:      out,
		Err:      errOut,
		HasFile:  func(path string) bool { _, err := os.Stat(path); return err == nil },
		TypesDir: dir,
	})
	require.Equal(t, 0, code)
	require.Empty(t, errOut.String())

	content, err := os.ReadFile(filepath.Join(dir, "user_logged_in.go"))
	require.NoError(t, err)
	require.Contains(t, string(content), "type UserLoggedIn struct")
}
