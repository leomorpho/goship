package security

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCronEntrypointContract_RedSpec(t *testing.T) {
	contents, err := os.ReadFile("../../tools/private/control-plane/docs/03-customer-runtime-contract.md")
	require.NoError(t, err)

	text := string(contents)
	assert.Contains(t, text, "signed internal cron endpoint(s)")
	assert.Contains(t, text, "cron entrypoint contract")

	source, err := os.ReadFile("managed_hooks.go")
	require.NoError(t, err)

	code := string(source)
	assert.Contains(t, code, "CronRequest")
	assert.Contains(t, code, "CronRequestVerifier")
	assert.Contains(t, code, "SignCronRequest")
	assert.Contains(t, code, "VerifyCronRequest")
}
