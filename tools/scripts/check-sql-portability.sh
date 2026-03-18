#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="${ROOT_DIR:-$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)}"
SQL_PORTABILITY_SKIP_CONFIG="${SQL_PORTABILITY_SKIP_CONFIG:-0}"

cd "${ROOT_DIR}"

echo "Checking sql-core-v1 portability contract..."

if [[ "${SQL_PORTABILITY_SKIP_CONFIG}" != "1" ]]; then
  go test ./config -run 'TestRuntimeMetadata(SQLitePromotionContract|PostgresHasNoPromotionPath)$' -count=1
fi

sql_files=()
for dir in db/queries framework/repos/storage/queries; do
  if [[ -d "${dir}" ]]; then
    while IFS= read -r file; do
      sql_files+=("${file}")
    done < <(find "${dir}" -maxdepth 1 -type f -name '*.sql' | sort)
  fi
done

if [[ "${#sql_files[@]}" -eq 0 ]]; then
  echo "ERROR: no sql portability sources found"
  exit 1
fi

awk '
function trim(s) {
  sub(/^[[:space:]]+/, "", s)
  sub(/[[:space:]]+$/, "", s)
  return s
}

function flush_current() {
  if (current_name == "") {
    return
  }
  names[current_name] = 1
  file_for[current_name] = current_file
  body_for[current_name] = body_text
  branch_for[current_name] = current_branch
  current_name = ""
  current_file = ""
  body_text = ""
  current_branch = ""
}

function report_nonportable(query_name, file_path, detail) {
  printf("ERROR: non-portable SQL in query=%s file=%s: %s\n", query_name, file_path, detail) > "/dev/stderr"
  fail = 1
}

BEGIN {
  fail = 0
  body_text = ""
  current_name = ""
  current_branch = ""
  branch_hint = ""
  current_file = ""
}

{
  if (FNR == 1) {
    branch_hint = ""
  }

  line = $0
  trimmed = trim(line)

  if (trimmed ~ /^-- branch:/) {
    branch_hint = trimmed
    next
  }

  if (trimmed ~ /^-- name:/) {
    flush_current()
    current_name = trim(substr(trimmed, length("-- name:") + 1))
    current_file = FILENAME
    current_branch = branch_hint
    branch_hint = ""
    next
  }

  if (current_name != "") {
    if (body_text == "") {
      body_text = line
    } else {
      body_text = body_text "\n" line
    }
  }
}

END {
  flush_current()

  for (name in names) {
    file = file_for[name]
    body = body_for[name]
    branch = branch_for[name]

    if (file == "framework/repos/storage/queries/storage.sql") {
      if (branch == "") {
        report_nonportable(name, file, "branch handling required for storage portability split")
      }
      if (name ~ /_sqlite$/ && body ~ /\$[0-9]+/) {
        report_nonportable(name, file, "sqlite queries should use ? placeholders instead of $n")
      }
      if (name ~ /_postgres$/ && body ~ /\?/) {
        report_nonportable(name, file, "postgres queries should use $n placeholders instead of ?")
      }
      continue
    }

    if (file == "db/queries/migrations.sql" && name ~ /_postgres$/) {
      if (branch == "") {
        report_nonportable(name, file, "branch handling required for migration portability split")
      }
      if (branch !~ /postgres-only/) {
        report_nonportable(name, file, "postgres-only branch handling required")
      }
      if (body ~ /\?/) {
        report_nonportable(name, file, "postgres queries should use $n placeholders instead of ?")
      }
      continue
    }

    if (file == "db/queries/migrations.sql" && name ~ /_sqlite$/) {
      if (branch == "") {
        report_nonportable(name, file, "branch handling required for migration portability split")
      }
      if (body ~ /\$[0-9]+/) {
        report_nonportable(name, file, "sqlite queries should use ? placeholders instead of $n")
      }
    }
  }

  if (fail != 0) {
    exit 1
  }
}
' "${sql_files[@]}"

echo "SQL portability contract passed."
