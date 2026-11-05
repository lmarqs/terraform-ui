#!/usr/bin/env bash
# Regenerates the landing page's before/after panels from one real plan of this fixture:
#   docs/_includes/before-after/before.html   terraform plan -no-color, verbatim
#   docs/_includes/before-after/human.html    tfui's plan tree (macro screenshot of the TUI)
#   docs/_includes/before-after/agent.html    tfui plan --ci, the headless output an agent reads
#   docs/_data/before_after.yml               the numbers the page quotes
# Nothing reaches AWS: the provider has fake credentials and every plan runs with -refresh=false. The token counts come
# from Anthropic's token counting endpoint, so the run needs ANTHROPIC_API_KEY.
#
# Usage: ANTHROPIC_API_KEY=... demo/before-after/generate.sh [path/to/tfui]
set -euo pipefail
: "${ANTHROPIC_API_KEY:?set ANTHROPIC_API_KEY: the page quotes Claude token counts of both outputs}"

HERE="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"
TFUI="${1:-}"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"; rm -f "$HERE/human.txt" "$HERE/tfplan"' EXIT

if [ -z "$TFUI" ]; then
  TFUI="$WORK/tfui"
  (cd "$ROOT" && go build -o "$TFUI" ./cmd/tfui)
fi

cd "$HERE"
terraform init -input=false >/dev/null
terraform plan -no-color -input=false -refresh=false -out=tfplan >"$WORK/before.txt"
terraform show -json tfplan >"$WORK/plan.json"

"$TFUI" --plan "$WORK/plan.json" --state ./terraform.tfstate --macro ./before-after.tape >/dev/null
mv human.txt "$WORK/human.txt"

status=0
"$TFUI" plan --ci --terraform-bin "$HERE/terraform-offline" >"$WORK/agent.txt" || status=$?
[ "$status" -eq 2 ] || { echo "tfui plan --ci exited $status, expected 2 (changes present)" >&2; exit 1; }

python3 -I "$HERE/render.py" "$WORK" "$ROOT/docs"
echo "Wrote docs/_includes/before-after/ and docs/_data/before_after.yml"
