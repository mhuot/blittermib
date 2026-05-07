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
            # Lexical-marker gate. SMI grammar allows arbitrary
            # whitespace between the three tokens — must stay in
            # sync with mibcorpus.DefinitionsBeginMarker on the
            # Go side.
            if grep -qE 'DEFINITIONS[[:space:]]+::=[[:space:]]+BEGIN' "$f" 2>/dev/null; then
                printf '%s\n' "$f"
            fi
        done
}

# Extract the SMIv2 module name from a MIB file: the first token
# that appears before `DEFINITIONS ::= BEGIN`. Returns empty on
# non-match.
#
# Single-process awk implementation — a `sed | grep -m1 | sed`
# pipeline tripped pipefail+SIGPIPE on large MIBs (grep -m1 closes
# the pipe before upstream sed finishes pumping, sed exits
# non-zero, pipefail escalates).
#
# `--`-to-EOL comments are stripped before matching so a
# documentation line like `-- Defines XYZ-MIB DEFINITIONS ::= BEGIN`
# in a copyright preamble doesn't mask the real opener.
mib_module_name() {
    local f="$1"
    awk '
        { sub(/--.*/, "") }
        /^[[:space:]]*[A-Za-z][A-Za-z0-9-]*[[:space:]]+DEFINITIONS[[:space:]]+::=[[:space:]]+BEGIN/ {
            sub(/^[[:space:]]*/, "")
            sub(/[[:space:]]+DEFINITIONS.*/, "")
            print
            exit
        }
    ' "$f"
}
