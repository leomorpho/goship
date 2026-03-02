#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

"${SCRIPT_DIR}/up.sh" "${1:-}"

npm install
npm run build
npx tailwindcss -i ./styles/styles.css -o ./static/styles_bundle.css

echo "Tip: run 'nvm use v18.20.7' if JS tooling fails."
overmind start

