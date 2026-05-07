# How to add a MIB to blittermib

A concrete, step-by-step walkthrough. **5–10 minutes per MIB** once you
have a clean source file.

For the reference matter (full license-tag matrix, override semantics,
4-tier CI table), see [CONTRIBUTING.md](CONTRIBUTING.md).

## Prerequisites

```bash
git clone https://github.com/no42-org/blittermib.git
cd blittermib

# libsmi (Tier 3 parser + smilint debugging)
brew install libsmi              # macOS
sudo apt install libsmi2-tools   # Debian/Ubuntu
sudo dnf install libsmi-devel    # Fedora/RHEL

# Go 1.26+ for `make index` / `make verify`
go version

# GNU grep on macOS (Tier 1 uses `grep -P`)
brew install grep
```

## 1. Inspect the source

Open the MIB and find three things:

1. **Module name** — the identifier in `<NAME> DEFINITIONS ::= BEGIN`.
   Example: `CISCO-RTTMON-MIB`.
2. **Module OID** — the `::= { ... }` at the end of the
   `MODULE-IDENTITY` block. Example: `::= { ciscoMgmt 42 }` resolving
   to `.1.3.6.1.4.1.9.9.42`.
3. **Copyright owner** — usually a `-- Copyright (c) <year> <Vendor>`
   line near the top.

Get the file from an authoritative source (vendor's MIB distribution
page, RFC archive). Don't reverse-engineer from a commercial product
without permission.

## 2. Decide where the file goes

Match the MIB's OID against the routing table:

| OID prefix              | Destination                              |
|-------------------------|------------------------------------------|
| `.1.3.6.1.4.1.{PEN}.*`  | `mibs/vendors/{PEN}-{slug}/`             |
| `.1.3.6.1.2.1.*`        | `mibs/ietf/{group}/`                     |
| `.1.3.6.1.6.*`          | `mibs/iana/`                             |
| `.1.3.6.1.3.*`          | `mibs/experimental/`                     |
| anything else           | `mibs/unsorted/` — open a discussion     |

**Vendor case (most common):**

- PEN is the 7th dotted segment of the OID.
- Look it up in `internal/iana/pen.txt` or at
  <https://www.iana.org/assignments/enterprise-numbers/>.
- Slug is the kebab-cased vendor name with suffix words stripped
  (`Inc`, `Corp`, `Networks`, `Systems`, …). See
  `internal/iana/pen.go::Slug` for the canonical rules. Existing
  examples: `9-cisco`, `22610-a10`, `2636-juniper`.
- If the vendor's directory doesn't exist yet, just create it.

**IETF case:**

- Look up the module name in `mibs/_groups.yaml`. Drop the file in
  that group's directory.
- Not listed? Either add an entry to `_groups.yaml` for the right
  group, or place under `mibs/ietf/other/`.

## 3. Place the file with the canonical filename

The filename must equal the MODULE-IDENTITY name with **no extension**.
CI Tier 2 enforces this.

```bash
cp ~/Downloads/CISCO-RTTMON-MIB.mib mibs/vendors/9-cisco/CISCO-RTTMON-MIB
#                    extension dropped ──────────────────────────────────^
```

## 4. Regenerate the metadata index

```bash
make index
```

This rewrites `mibs/INDEX.yaml`. Open the diff and verify two things:

1. **`license:`** matches the actual copyright owner. The auto-detector
   reads the first 200 lines and matches 11 starter patterns.
   `Copyright (c) <year> Cisco Systems` → `license: cisco`. No match →
   `license: unknown`.
2. **`pen:` and `vendor:`** match the directory you placed the file in.
   `vendors/9-cisco/...` → `pen: 9` / `vendor: cisco`.

## 5. Fix the license tag if needed

If step 4 produced `license: unknown` and you know the correct tag:

```yaml
# mibs/_overrides.yaml
licenses:
  YOUR-MODULE-NAME: vendor-public   # or rfc-editor, cisco, etc.
```

Re-run `make index`. The override wins over the auto-detector.

The valid tags are the files in `mibs/LICENSES/`. Need a new tag? Add
a regex to `cmd/mib-index/license.go::licensePatterns` AND a matching
`LICENSES/<tag>.txt` — discuss with the maintainer first.

## 6. Local pre-flight

```bash
make verify-mibs    # Tier 1 + 2 + 3 (libsmi-driven)
make verify         # gofmt + vet + tests
```

If Tier 3 fails on smilint, run it directly for per-MIB diagnostics:

```bash
smilint -p mibs/vendors/9-cisco mibs/vendors/9-cisco/CISCO-RTTMON-MIB
```

Most common Tier 3 failure: **missing IMPORTS**. Your MIB imports a
parent (e.g. `CISCO-SMI`) that isn't in the corpus yet. Add the parent
in the same PR — or verify it's already there under another directory.

## 7. Commit and open the PR

```bash
git checkout -b add-cisco-rttmon-mib
git add mibs/vendors/9-cisco/CISCO-RTTMON-MIB mibs/INDEX.yaml
# (and mibs/_overrides.yaml if you edited it)
git commit -m "feat(mibs): add CISCO-RTTMON-MIB"
git push -u origin add-cisco-rttmon-mib
```

Open the PR via the **picker URL** so the checklist appears:

```
https://github.com/no42-org/blittermib/compare/main...add-cisco-rttmon-mib?template=add-mib.md
```

Without `?template=add-mib.md`, GitHub renders an empty PR body — the
template at `.github/PULL_REQUEST_TEMPLATE/add-mib.md` is only surfaced
via the picker URL.

## 8. What CI does

Four blocking tiers:

| Tier | Check                                     | Typical failure                  |
|------|-------------------------------------------|----------------------------------|
| 1    | ASCII clean + `DEFINITIONS ::= BEGIN ... END` | non-ASCII bytes, missing END |
| 2    | filename = MODULE-IDENTITY name           | extension not stripped           |
| 3    | smidump + smilint zero errors             | unsatisfied IMPORTS              |
| 4    | per-module error-set comparison vs `main` | redefined symbol, namespace collision |

Plus an **INDEX.yaml drift check** — fails if you forgot to commit
`make index` output.

A green CI + maintainer review = merge.

## Common gotchas

| Symptom                                                       | Cause                                                  | Fix                                                       |
|---------------------------------------------------------------|--------------------------------------------------------|-----------------------------------------------------------|
| Tier 2: filename mismatch                                      | left the `.mib` / `.txt` extension on                   | rename the file                                            |
| Tier 3: "module 'X' not found"                                 | imported MIB isn't in the corpus                        | add the parent MIB to the same PR                          |
| Drift check fails                                              | forgot `make index` after placing the MIB               | run `make index`, commit the diff                          |
| Sticky comment lists your MIB as `unknown`                     | header doesn't match any auto-detect pattern            | add an entry to `_overrides.yaml`                          |
| Tier 4 fails on a MIB you didn't touch                         | your new MIB redefines a symbol the existing MIB uses   | rename the conflict, or coordinate with the maintainer    |
| `make verify-mibs-lexical` fails on macOS with `grep: invalid option -- P` | macOS BSD grep doesn't support PCRE                  | `brew install grep`, put GNU grep first on PATH           |
