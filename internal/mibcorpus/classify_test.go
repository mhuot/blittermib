package mibcorpus

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestClassify covers the destination-routing rules per design.md
// Decision 9 of the mib-corpus change. Tests the pure classification
// function — the libsmi-driven plan pipelines that consume Classify
// are integration-tested separately.
func TestClassify(t *testing.T) {
	groups := NewGroupMap(map[string]string{
		"IF-MIB":           "interfaces",
		"SNMPv2-SMI":       "core",
		"INET-ADDRESS-MIB": "transport",
	})

	cases := []struct {
		name       string
		oid        string
		module     string
		wantDir    string
		wantPEN    uint32
		wantVendor string
		wantConf   Confidence
	}{
		{
			name: "Cisco vendor MIB (well-known PEN)",
			oid:  "1.3.6.1.4.1.9.9.42", module: "CISCO-RTTMON-MIB",
			wantDir: "vendors/9-cisco", wantPEN: 9, wantVendor: "cisco", wantConf: ConfidenceHigh,
		},
		{
			name: "A10 vendor MIB (Networks suffix stripped)",
			oid:  "1.3.6.1.4.1.22610.1", module: "A10-AX-MIB",
			wantDir: "vendors/22610-a10", wantPEN: 22610, wantVendor: "a10", wantConf: ConfidenceHigh,
		},
		{
			name: "Vendor MIB with PEN unknown to curated registry",
			oid:  "1.3.6.1.4.1.999999.1", module: "MYSTERY-MIB",
			wantDir: "vendors/999999-unknown", wantPEN: 999999, wantVendor: "unknown", wantConf: ConfidenceMedium,
		},
		{
			name: "IETF MIB with curated group",
			oid:  "1.3.6.1.2.1.31", module: "IF-MIB",
			wantDir: "ietf/interfaces", wantConf: ConfidenceHigh,
		},
		{
			name: "IETF MIB without curated group → ietf/other",
			oid:  "1.3.6.1.2.1.99", module: "WEIRD-RFC-MIB",
			wantDir: "ietf/other", wantConf: ConfidenceHigh,
		},
		{
			name: "IANA registry MIB",
			oid:  "1.3.6.1.6.3.10", module: "SNMP-FRAMEWORK-MIB",
			wantDir: "iana", wantConf: ConfidenceHigh,
		},
		{
			name: "Experimental",
			oid:  "1.3.6.1.3.42", module: "EXP-MIB",
			wantDir: "experimental", wantConf: ConfidenceHigh,
		},
		{
			name: "Empty OID → unsorted",
			oid:  "", module: "BROKEN",
			wantDir: "unsorted", wantConf: ConfidenceLow,
		},
		{
			name: "OID outside known prefixes → unsorted",
			oid:  "1.2.840.113549", module: "SOMEONE-ELSES-MIB",
			wantDir: "unsorted", wantConf: ConfidenceLow,
		},
		{
			name: "Vendor OID with PEN 0 (reserved)",
			oid:  "1.3.6.1.4.1.0.1", module: "RESERVED",
			wantDir: "unsorted", wantConf: ConfidenceLow,
		},
		{
			name: "Leading-dot OID is normalised",
			oid:  ".1.3.6.1.4.1.9.9.42", module: "CISCO-RTTMON-MIB",
			wantDir: "vendors/9-cisco", wantPEN: 9, wantVendor: "cisco", wantConf: ConfidenceHigh,
		},
		{
			name: "Whitespace-padded OID is normalised",
			oid:  "  1.3.6.1.2.1.31  ", module: "IF-MIB",
			wantDir: "ietf/interfaces", wantConf: ConfidenceHigh,
		},
		{
			name: "Leading-zero PEN is rejected as ambiguous",
			oid:  "1.3.6.1.4.1.09.1", module: "AMBIGUOUS",
			wantDir: "unsorted", wantConf: ConfidenceLow,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Classify(tc.oid, tc.module, groups, nil)
			if got.DstDir != tc.wantDir {
				t.Errorf("DstDir = %q, want %q", got.DstDir, tc.wantDir)
			}
			if got.PEN != tc.wantPEN {
				t.Errorf("PEN = %d, want %d", got.PEN, tc.wantPEN)
			}
			if got.Vendor != tc.wantVendor {
				t.Errorf("Vendor = %q, want %q", got.Vendor, tc.wantVendor)
			}
			if got.Confidence != tc.wantConf {
				t.Errorf("Confidence = %s, want %s", got.Confidence, tc.wantConf)
			}
		})
	}
}

// TestSlugOverrideWins asserts that a caller-supplied override beats
// iana.Slug's rule output.
func TestSlugOverrideWins(t *testing.T) {
	overrides := map[string]string{
		"Cisco Systems, Inc.": "cisco-pinned",
	}
	cls := Classify("1.3.6.1.4.1.9.9.42", "CISCO-RTTMON-MIB",
		NewGroupMap(nil), overrides)
	if cls.Vendor != "cisco-pinned" {
		t.Errorf("Vendor = %q, want cisco-pinned (override should win)", cls.Vendor)
	}
	if cls.DstDir != "vendors/9-cisco-pinned" {
		t.Errorf("DstDir = %q", cls.DstDir)
	}
}

// TestLoadGroupsMissing asserts a missing _groups.yaml is non-fatal
// and yields an empty map.
func TestLoadGroupsMissing(t *testing.T) {
	g, err := LoadGroups(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err != nil {
		t.Fatalf("LoadGroups(missing): %v", err)
	}
	if got := g.GroupOf("IF-MIB"); got != "" {
		t.Errorf("GroupOf on empty map = %q, want \"\"", got)
	}
}

// TestLoadGroupsParse covers the inverted-map shape produced by a
// minimal _groups.yaml file.
func TestLoadGroupsParse(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "_groups.yaml")
	body := strings.Join([]string{
		"core: [SNMPv2-SMI, SNMPv2-TC]",
		"transport: [TCP-MIB, UDP-MIB]",
		"interfaces: [IF-MIB]",
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	g, err := LoadGroups(path)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"SNMPv2-SMI": "core",
		"TCP-MIB":    "transport",
		"IF-MIB":     "interfaces",
		"NOT-LISTED": "",
	}
	for mod, want := range cases {
		if got := g.GroupOf(mod); got != want {
			t.Errorf("GroupOf(%q) = %q, want %q", mod, got, want)
		}
	}
}

// TestLoadGroupsRejectsDuplicate covers the duplicate-module guard:
// the same module listed under two groups must surface as an error
// rather than producing non-deterministic group assignment via map
// iteration order.
func TestLoadGroupsRejectsDuplicate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "_groups.yaml")
	body := "core: [DUPE-MIB]\ntransport: [DUPE-MIB]\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadGroups(path); err == nil {
		t.Error("LoadGroups accepted a module listed in two groups; want error")
	}
}
