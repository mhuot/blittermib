# Copilot review instructions — blittermib

Go web app that compiles SNMP MIBs (libsmi/smidump) into a SQLite corpus and
serves a server-rendered browse/search UI (templ + htmx/Alpine, vanilla JS
assets), a JSON API, and MCP tools. This repo is a deployment fork of
no42-org/blittermib; changes here usually become upstream PRs.

## Repo-specific invariants to verify

- **Single DB connection.** The store pins `SetMaxOpenConns(1)` because SQLite
  PRAGMAs are per-connection. Every query serializes on that one connection —
  flag anything that can hold it long: unbounded queries, missing timeouts,
  N+1 query loops, work inside row iteration. One slow query stalls the whole
  site; treat query-cost regressions as correctness bugs, not style.
- **FTS5 search cost.** `sanitizeFTS` turns each token into a prefix match and
  `ORDER BY bm25` scores every match before `LIMIT`. Short or broad prefixes
  are expensive; the per-token minimum length (`minFTSTokenLen`) and the
  per-query timeout in `store.Search` exist for this reason.
- **Sentinel error contracts.** Callers test with `errors.Is`:
  `store.ErrNotFound`, `store.ErrQueryTooShort`, `context.DeadlineExceeded`
  (may arrive wrapped or bare). Flag error swallowing that turns a failure
  into a fake empty result, and flag handlers that log or 500 a client abort
  (`context.Canceled`) — aborts are steady-state typeahead behavior.
- **Client/server mirrors.** `internal/server/assets/palette.js` mirrors
  server-side search gating (per-token minimum length, OID-shape exemption
  matching `oidPrefixQuery`). If one side changes, check the other still
  matches, including how characters are counted (Go runes vs JS code units).
- **MCP error contract.** MCP tool handlers deliberately propagate descriptive
  store errors instead of returning empty results — an LLM caller can act on
  "query too short"; an empty hit list reads as "nothing exists".
- **Accessibility is a hard requirement (WCAG AAA target).** In JS/templ
  changes, verify ARIA combobox/listbox wiring, `aria-activedescendant`,
  focus management/restoration, and reduced-motion paths are preserved.

## Skip

- `internal/web/*_templ.go` is generated from the sibling `.templ` sources —
  review the `.templ` file, not the generated Go.
- `compose.override.yml`, `.github/workflows/deploy.yml`,
  `.github/workflows/sync-upstream.yml`, `.github/actionlint.yaml` are
  fork-only deployment infra and are never sent upstream.
- Formatting is enforced by CI (gofmt, lint) — don't comment on it.

## Style

- Prefer findings with a concrete failure scenario (inputs/state → wrong
  behavior) over general suggestions; verify claims against the code first.
- Comments should state constraints the code can't express; don't ask for
  comments that restate the code.
