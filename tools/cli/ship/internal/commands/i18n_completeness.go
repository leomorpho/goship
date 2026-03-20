package commands

import i18ncheck "github.com/leomorpho/goship/tools/cli/ship/internal/i18ncheck"

type I18nCompletenessIssue = i18ncheck.CompletenessIssue

func CollectI18nCompletenessIssues(root string) ([]I18nCompletenessIssue, error) {
	return i18ncheck.CollectCompletenessIssues(root)
}
