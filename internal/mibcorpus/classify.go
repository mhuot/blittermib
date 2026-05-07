// Package mibcorpus carries the classification + group-mapping logic
// shared between the corpus-management CLIs (`cmd/mib-migrate`,
// `cmd/mib-ingest`). The split is deliberate: both tools route MIBs
// to the same destination directories per design.md Decision 9 of the
// mib-corpus change, and divergence between them would surface as
// contributors getting different answers depending on which command
// they ran.
package mibcorpus

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/no42-org/blittermib/internal/iana"
)

// ValidModuleName matches the conservative character set we accept
// for a MODULE-IDENTITY name when synthesising a destination
// filename. Rejects path separators, `..`, and shell-active
// characters so an adversarial MIB can't produce a path-traversing
// destination via its module name.
var ValidModuleName = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// Confidence labels each classification entry; low-confidence rows are
// routed to mibs/unsorted/ for the maintainer to re-classify by hand.
type Confidence string

const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
)

// Classification is the result of mapping a parsed MIB to a corpus
// destination directory per design.md Decision 9 of the mib-corpus
// change.
type Classification struct {
	// DstDir is the destination subdirectory under mibs/, e.g.
	// "vendors/9-cisco" or "ietf/interfaces". Never absolute.
	DstDir string
	// PEN is the IANA Private Enterprise Number for vendor MIBs;
	// zero for non-vendor classifications.
	PEN uint32
	// Vendor is the kebab-case slug for vendor MIBs; empty otherwise.
	Vendor string
	// Confidence is the routing self-rating for this entry.
	Confidence Confidence
}

// Classify maps a MIB's MODULE-IDENTITY OID + module name to a corpus
// destination. The rules:
//
//   - .1.3.6.1.4.1.{PEN}.* → vendors/{PEN}-{slug}/  (high if PEN known,
//     medium if not)
//   - .1.3.6.1.2.1.*       → ietf/{group}/          (group from
//     groupMap, falls back to "other") (high)
//   - .1.3.6.1.6.*         → iana/                  (high)
//   - .1.3.6.1.3.*         → experimental/          (high)
//   - everything else      → unsorted/              (low)
//
// slugOverrides is consulted before iana.Slug when resolving the
// vendor slug; an entry whose key matches the upstream PEN org name
// (or its lowercased form) wins. Pass nil for no migration-specific
// overrides.
func Classify(oid, moduleName string, groups GroupMap, slugOverrides map[string]string) Classification {
	// Normalise: trim whitespace and a leading dot. The compile
	// pipeline currently emits dotless OIDs, but defending here means
	// a future change in upstream rendering doesn't silently route
	// everything to unsorted.
	oid = strings.TrimPrefix(strings.TrimSpace(oid), ".")
	if oid == "" {
		return Classification{DstDir: "unsorted", Confidence: ConfidenceLow}
	}
	parts := strings.Split(oid, ".")

	// Vendor: .1.3.6.1.4.1.{PEN}.*
	if hasPrefix(parts, "1.3.6.1.4.1") && len(parts) >= 7 {
		penStr := parts[6]
		penU64, err := strconv.ParseUint(penStr, 10, 32)
		if err != nil || penU64 == 0 {
			return Classification{DstDir: "unsorted", Confidence: ConfidenceLow}
		}
		// Reject leading-zero / non-canonical decimal — "01" parses
		// to 1 silently, which would route to a different vendor's
		// directory than the operator expects.
		if penStr != strconv.FormatUint(penU64, 10) {
			return Classification{DstDir: "unsorted", Confidence: ConfidenceLow}
		}
		pen := uint32(penU64)
		slug, conf := slugFor(pen, slugOverrides)
		return Classification{
			DstDir:     fmt.Sprintf("vendors/%d-%s", pen, slug),
			PEN:        pen,
			Vendor:     slug,
			Confidence: conf,
		}
	}

	// IETF mib-2: .1.3.6.1.2.1.*
	if hasPrefix(parts, "1.3.6.1.2.1") {
		group := groups.GroupOf(moduleName)
		if group == "" {
			group = "other"
		}
		return Classification{DstDir: "ietf/" + group, Confidence: ConfidenceHigh}
	}

	// IANA: .1.3.6.1.6.*
	if hasPrefix(parts, "1.3.6.1.6") {
		return Classification{DstDir: "iana", Confidence: ConfidenceHigh}
	}

	// Experimental: .1.3.6.1.3.*
	if hasPrefix(parts, "1.3.6.1.3") {
		return Classification{DstDir: "experimental", Confidence: ConfidenceHigh}
	}

	return Classification{DstDir: "unsorted", Confidence: ConfidenceLow}
}

// slugFor resolves a PEN to its kebab-case slug using (in order):
// caller-supplied slug overrides, then iana.Slug applied to the
// upstream registry name. Returns ("unknown", medium) if the PEN
// isn't in the curated registry — the maintainer can hand-edit
// downstream output before applying.
func slugFor(pen uint32, overrides map[string]string) (string, Confidence) {
	name, ok := iana.LookupPEN(pen)
	if !ok {
		return "unknown", ConfidenceMedium
	}
	if s, ok := lookupOverride(overrides, name); ok && s != "" {
		return s, ConfidenceHigh
	}
	if s := iana.Slug(name); s != "" {
		return s, ConfidenceHigh
	}
	return "unknown", ConfidenceMedium
}

func lookupOverride(m map[string]string, name string) (string, bool) {
	if m == nil {
		return "", false
	}
	if v, ok := m[name]; ok {
		return v, true
	}
	if v, ok := m[strings.ToLower(strings.TrimSpace(name))]; ok {
		return v, true
	}
	return "", false
}

func hasPrefix(parts []string, dottedPrefix string) bool {
	pp := strings.Split(dottedPrefix, ".")
	if len(parts) < len(pp) {
		return false
	}
	for i, p := range pp {
		if parts[i] != p {
			return false
		}
	}
	return true
}
