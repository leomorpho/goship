package policies

import i18ncheck "github.com/leomorpho/goship/tools/cli/ship/v2/internal/i18ncheck"

type i18nCompletenessIssue = i18ncheck.CompletenessIssue

func collectI18nCompletenessIssues(root string) ([]i18nCompletenessIssue, error) {
	return i18ncheck.CollectCompletenessIssues(root)
}
