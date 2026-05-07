#!/usr/bin/env bash
# Shared helpers for the mibs/ verification scripts. Sourced by the
# verify-mibs-* and diff-parse scripts.
#
# Defines `walk_mib_files` which prints (one path per line) every
# MIB-shaped file under a root, applying the same skip rules as
# `cmd/blittermib/loader.go::walkMIBFiles`:
#
#   - skip dirs whose basename starts with `.` (hidden / `.git`)
#   - skip the corpus's own metadata files (`README.md`,
#     `CONTRIBUTING.md`, `_groups.yaml`, `_overrides.yaml`,
#     `INDEX.yaml`)
#   - skip everything under `LICENSES/`
#   - skip files whose basename starts with `.`
#   - keep only `.mib`, `.txt`, `.my`, or no-extension files
#   - require the file to contain `DEFINITIONS ::= BEGIN`
#     (lexical-marker check; matches the loader's runtime gate)

set -euo pipefail

# walk_mib_files <root>
walk_mib_files() {
    local root="$1"
    [ -d "$root" ] || return 0
    find "$root" \
        \( -type d \( -name '.*' -o -name 'LICENSES' \) -prune \) -o \
        \( -type f -print \) | while IFS= read -r f; do
            local base
            base="$(basename "$f")"
            case "$base" in
                README.md|CONTRIBUTING.md|_groups.yaml|_overrides.yaml|INDEX.yaml) continue ;;
                .*) continue ;;
            esac
            case "$base" in
                *.mib|*.txt|*.my) : ;;
                *.*) continue ;;  # any other extension
            esac
            # Lexical-marker gate.
            if grep -q 'DEFINITIONS ::= BEGIN' "$f" 2>/dev/null; then
                printf '%s\n' "$f"
            fi
        done
}

# Extract the SMIv2 module name from a MIB file: the first token
# that appears before `DEFINITIONS ::= BEGIN`. Returns empty on
# non-match.
#
# `--`-to-EOL comments are stripped before the regex match so a
# documentation line like `-- Defines XYZ-MIB DEFINITIONS ::= BEGIN`
# in a copyright preamble doesn't mask the real opener (grep's
# first-match semantics would otherwise pick the comment).
mib_module_name() {
    local f="$1"
    sed 's/--.*//' "$f" \
        | grep -m1 -E '^[[:space:]]*[A-Za-z][A-Za-z0-9-]*[[:space:]]+DEFINITIONS[[:space:]]*::=[[:space:]]*BEGIN' \
        | sed -E 's/^[[:space:]]*([A-Za-z][A-Za-z0-9-]*).*/\1/'
}
