#!/usr/bin/env bash
# Tier 1 — Lexical: every file under mibs/ that passes the MIB-shape
# filter must be ASCII-clean and contain `DEFINITIONS ::= BEGIN ... END`.
#
# Cheap (file-IO + grep), runs in a few seconds on a 5000-file corpus.
# An empty mibs/ directory passes (no files = nothing to fail).

set -euo pipefail

ROOT="${MIBS_ROOT:-mibs}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib-mib-walk.sh
source "$SCRIPT_DIR/lib-mib-walk.sh"

if [ ! -d "$ROOT" ]; then
    echo "verify-mibs-lexical: $ROOT does not exist; nothing to verify"
    exit 0
fi

fail=0
checked=0
while IFS= read -r f; do
    checked=$((checked + 1))
    # ASCII check: any byte > 0x7F is non-ASCII.
    if LC_ALL=C grep -lP '[^\x00-\x7F]' "$f" >/dev/null 2>&1; then
        echo "FAIL [non-ASCII]: $f" >&2
        fail=1
    fi
    # Structural markers. The walk_mib_files filter already gates on
    # `DEFINITIONS ::= BEGIN`; we additionally require an `END` token.
    if ! grep -qE '^[[:space:]]*END[[:space:]]*$' "$f"; then
        echo "FAIL [missing END marker]: $f" >&2
        fail=1
    fi
done < <(walk_mib_files "$ROOT")

if [ $fail -ne 0 ]; then
    echo "verify-mibs-lexical: FAILED ($checked files checked)" >&2
    exit 1
fi
echo "verify-mibs-lexical: OK ($checked files checked)"
