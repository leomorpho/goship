#!/usr/bin/env bash

set -euo pipefail

echo "== fresh-app CI lane =="
echo "running real generated-app proof targets"

run_checked_go_test() {
  local regex="$1"
  local output

  set +e
  output=$(go test ./tools/cli/ship/internal/commands -run "$regex" -count=1 2>&1)
  local status=$?
  set -e
  printf '%s\n' "$output"
  if [ $status -ne 0 ]; then
    exit $status
  fi
  if printf '%s\n' "$output" | grep -Eq '\[no tests to run\]|\[no test files\]'; then
    echo "fresh-app CI lane failed: targeted proof did not execute real tests" >&2
    exit 1
  fi
}

run_checked_go_test 'TestFreshApp$'
run_checked_go_test 'TestFreshAppStartupSmoke$'
run_checked_go_test 'TestFreshAppAPI$'
run_checked_go_test 'TestFreshAppAPIStartupSmoke$'
run_checked_go_test 'TestFreshAppAdminDashboardRequiresAdmin$'
run_checked_go_test 'TestFreshAppAdminDashboardCanManageGeneratedResource$'
run_checked_go_test 'TestFreshAppMailerPreviewFlow$'
run_checked_go_test 'TestFreshAppSupportedBatteryCombinationStaysBuildable$'
run_checked_go_test 'TestFreshApp(StorageModuleEnablesProfileUpload|EmailSubscriptionsModuleEnablesProfileToggle|PaidSubscriptionsModuleEnablesProfileToggle|NotificationsModuleEnablesHomeFeedInbox)$'
run_checked_go_test 'TestRuntimeReportIncludesContractVersionsAndModuleAdoption$'

echo "fresh-app CI lane passed"
