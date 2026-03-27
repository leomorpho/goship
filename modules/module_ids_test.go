package modules_test

import (
	"testing"

	"github.com/leomorpho/goship-modules/jobs"
	"github.com/leomorpho/goship-modules/notifications"
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	"github.com/leomorpho/goship-modules/storage"
	"github.com/leomorpho/goship/modules/ai"
	"github.com/leomorpho/goship/modules/auditlog"
	"github.com/leomorpho/goship/modules/flags"
	"github.com/leomorpho/goship/modules/i18n"
)

func TestCanonicalModuleIDsAreUnique(t *testing.T) {
	t.Parallel()

	ids := []string{
		ai.ModuleID,
		auditlog.ModuleID,
		flags.ModuleID,
		i18n.ModuleID,
		jobs.ModuleID,
		notifications.ModuleID,
		paidsubscriptions.ModuleID,
		storage.ModuleID,
	}

	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == "" {
			t.Fatal("module id must not be empty")
		}
		if _, ok := seen[id]; ok {
			t.Fatalf("duplicate module id %q", id)
		}
		seen[id] = struct{}{}
	}
}
