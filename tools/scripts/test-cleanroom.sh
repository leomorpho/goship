#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"

TMP_ROOT="$(mktemp -d "${TMPDIR:-/tmp}/ship-cleanroom.XXXXXX")"
trap 'rm -rf "${TMP_ROOT}"' EXIT

SHIP_BIN="${TMP_ROOT}/ship"
WORKSPACE="${TMP_ROOT}/workspace"
APP_NAME="cleanroom_app"
APP_DIR="${WORKSPACE}/${APP_NAME}"
TOOLS_DIR="${TMP_ROOT}/tools"

mkdir -p "${WORKSPACE}" "${TOOLS_DIR}"

echo "Building ship CLI..."
(cd "${ROOT_DIR}" && go build -o "${SHIP_BIN}" ./tools/cli/ship/cmd/ship)

if [[ "${SHIP_CLEANROOM_FAKE_TOOLS:-0}" == "1" ]]; then
  cat > "${TOOLS_DIR}/goose" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
dir=""
if [[ "${1:-}" == "-dir" ]]; then
  dir="${2:-}"
  shift 2
fi
if [[ "${1:-}" == "create" ]]; then
  name="${2:-migration}"
  ts="$(date +%Y%m%d%H%M%S)"
  mkdir -p "${dir}"
  cat > "${dir}/${ts}_${name}.sql" <<SQL
-- +goose Up
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 1;
-- +goose StatementEnd
SQL
  exit 0
fi
exit 0
EOF
  chmod +x "${TOOLS_DIR}/goose"

  cat > "${TOOLS_DIR}/bobgen-sql" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
cfg=""
while [[ $# -gt 0 ]]; do
  if [[ "$1" == "-c" ]]; then
    cfg="$2"
    shift 2
    continue
  fi
  shift
done
if [[ -n "${cfg}" && -f "${cfg}" ]]; then
  mkdir -p db/gen
  : > db/gen/.cleanroom-bobgen
fi
exit 0
EOF
  chmod +x "${TOOLS_DIR}/bobgen-sql"
else
  echo "Installing bobgen-sql..."
  GOBIN="${TOOLS_DIR}" go install github.com/stephenafamo/bob/gen/bobgen-sql@latest
fi

export PATH="${TOOLS_DIR}:${PATH}"

echo "Scaffolding fresh app..."
(cd "${WORKSPACE}" && "${SHIP_BIN}" new "${APP_NAME}" --module "example.com/${APP_NAME}")

echo "Preparing DB + migrations..."
cd "${APP_DIR}"

# Generated scaffold defaults bobgen engine to postgres; use sqlite for clean-room DB flow.
sed -i.bak 's/engine: "postgres"/engine: "sqlite"/' db/bobgen.yaml
rm -f db/bobgen.yaml.bak

"${SHIP_BIN}" doctor
"${SHIP_BIN}" db:make create_users

MIGRATION_FILE="$(ls -1 db/migrate/migrations/*_create_users.sql | head -n1)"
cat > "${MIGRATION_FILE}" <<'SQL'
-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
  id INTEGER PRIMARY KEY,
  email TEXT NOT NULL UNIQUE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
SQL

DB_URL="sqlite3://${APP_DIR}/.tmp/cleanroom.sqlite3"
mkdir -p .tmp

echo "Running DB flow..."
DATABASE_URL="${DB_URL}" "${SHIP_BIN}" db:migrate
DATABASE_URL="${DB_URL}" "${SHIP_BIN}" db:status
"${SHIP_BIN}" db:generate

echo "Running app tests..."
go test ./...

echo "Clean-room verification passed."
