package security

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCronEntrypointContract_RedSpec(t *testing.T) {
	root := repoRootForSecurityContractTest(t)

	managedDoc := mustReadSecurityContractText(t, filepath.Join(root, "docs", "architecture", "09-standalone-and-managed-mode.md"))
	code := mustReadSecurityContractText(t, filepath.Join(root, "framework", "security", "managed_hooks.go"))

	assert.Contains(t, strings.ToLower(managedDoc), "cron entrypoint contract")
	assert.Contains(t, code, "CronRequest")
	assert.Contains(t, code, "CronRequestVerifier")
	assert.Contains(t, code, "SignCronRequest")
	assert.Contains(t, code, "VerifyCronRequest")
	assert.Contains(t, code, "signed internal cron endpoint(s)")
}
