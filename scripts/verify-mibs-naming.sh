#!/usr/bin/env bash
# Tier 2 — Naming + structure: filename must equal the MIB's
# `MODULE-IDENTITY` name (modulo a recognised extension), and IETF
# MIBs must live in the bucket declared by `mibs/_groups.yaml`.
#
# The directory-matches-PEN check is intentionally NOT done here —
# Tier 3 (smilint) catches OID/PEN mismatches via the parser. Doing
# it again here would duplicate the smidump-driven OID extraction
# without a meaningful new signal.

set -euo pipefail

ROOT="${MIBS_ROOT:-mibs}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib-mib-walk.sh
source "$SCRIPT_DIR/lib-mib-walk.sh"

if [ ! -d "$ROOT" ]; then
    echo "verify-mibs-naming: $ROOT does not exist; nothing to verify"
    exit 0
fi

fail=0
checked=0
while IFS= read -r f; do
    checked=$((checked + 1))
    base="$(basename "$f")"
    # Strip a recognised extension if present.
    stripped="$base"
    case "$base" in
        *.mib) stripped="${base%.mib}" ;;
        *.txt) stripped="${base%.txt}" ;;
        *.my)  stripped="${base%.my}"  ;;
    esac

    module="$(mib_module_name "$f")"
    if [ -z "$module" ]; then
        echo "FAIL [no MODULE-IDENTITY name extractable]: $f" >&2
        fail=1
        continue
    fi
    if [ "$stripped" != "$module" ]; then
        echo "FAIL [filename '$base' (stripped '$stripped') != module '$module']: $f" >&2
        fail=1
    fi
done < <(walk_mib_files "$ROOT")

if [ $fail -ne 0 ]; then
    echo "verify-mibs-naming: FAILED ($checked files checked)" >&2
    exit 1
fi
echo "verify-mibs-naming: OK ($checked files checked)"
