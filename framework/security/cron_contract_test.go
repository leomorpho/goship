package security

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCronEntrypointContract_RedSpec(t *testing.T) {
	candidates := []string{
		"../../docs/architecture/09-standalone-and-managed-mode.md",
		"../docs/architecture/09-standalone-and-managed-mode.md",
	}

	var (
		contents []byte
		err      error
	)
	for _, candidate := range candidates {
		contents, err = os.ReadFile(candidate)
		if err == nil {
			break
		}
	}
	if err != nil {
		t.Skipf("runtime contract doc not present in this checkout; looked in %v", candidates)
	}

	text := string(contents)
	assert.Contains(t, text, "cron entrypoint contract")
	assert.Contains(t, text, "signed-request pattern")

	source, err := os.ReadFile("managed_hooks.go")
	require.NoError(t, err)

	code := string(source)
	assert.Contains(t, code, "CronRequest")
	assert.Contains(t, code, "CronRequestVerifier")
	assert.Contains(t, code, "SignCronRequest")
	assert.Contains(t, code, "VerifyCronRequest")
}
