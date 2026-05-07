package main

// MigrationSlugOverrides pins per-name slug values used during the
// initial corpus migration. Most slug abbreviations live in
// internal/iana itself (the built-in override map handles common
// cases like "Hewlett Packard Enterprise" → "hp-enterprise"); this
// table is the migration-tool-specific layer for vendors whose IANA
// registry name slugs to a less-recognisable form than the operator
// community uses, or where two distinct PENs would otherwise collide
// on the same slug.
//
// Keys may be the verbatim upstream registry name OR its lowercased
// + trimmed form — the lookup tries both. Empty in v1.0; populate
// only when the migration plan surfaces an ambiguity.
//
// Example:
//
//	"Some Vendor With Awkward IANA Name": "shorter-slug",
var MigrationSlugOverrides = map[string]string{}
