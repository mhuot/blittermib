package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPathStaysInside covers the path-safety guard: relative dst must
// not escape --root via `..` or absolute path.
func TestPathStaysInside(t *testing.T) {
	root, err := filepath.Abs(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		dst  string
		want bool
	}{
		{"mibs/vendors/9-cisco/CISCO-MIB", true},
		{"mibs/ietf/core/SNMPv2-SMI", true},
		{"./mibs/CISCO-MIB", true},
		{"a/b/c", true},
		{"../escape", false},
		{"../../etc/passwd", false},
		{"mibs/../../escape", false},
		{"/etc/passwd", false}, // absolute
		{"/tmp/foo", false},    // absolute
		{".", true},            // edge: same as root
	}
	for _, c := range cases {
		got := pathStaysInside(root, c.dst)
		if got != c.want {
			t.Errorf("pathStaysInside(root, %q) = %v, want %v", c.dst, got, c.want)
		}
	}
}

// TestLoadPlanRejectsBadHeader ensures the apply step refuses to run
// on a TSV whose first row isn't the canonical header.
func TestLoadPlanRejectsBadHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.tsv")
	body := strings.Join([]string{
		"foo\tbar\tbaz\tqux\tquux",
		"row\tdata\tA\t9\thigh",
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadPlan(path); err == nil {
		t.Error("loadPlan accepted a TSV with a wrong header; want error")
	}
}

// TestLoadPlanToleratesCommentsAndBlanks asserts hand-edited plans
// with `#` comments and blank lines parse cleanly.
func TestLoadPlanToleratesCommentsAndBlanks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.tsv")
	body := strings.Join([]string{
		"src_path\tdst_path\tmodule\tpen\tconfidence",
		"# this row is commented out",
		"",
		"old/A\tmibs/ietf/core/A\tA\t-\thigh",
		"   ",
		"# another comment",
		"old/B\tmibs/ietf/core/B\tB\t-\thigh",
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	rows, err := loadPlan(path)
	if err != nil {
		t.Fatalf("loadPlan: %v", err)
	}
	if len(rows) != 2 {
		t.Errorf("loadPlan returned %d rows, want 2: %v", len(rows), rows)
	}
}

// TestLoadPlanRejectsShortRow ensures a misformed data row is caught
// rather than silently truncated.
func TestLoadPlanRejectsShortRow(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plan.tsv")
	body := strings.Join([]string{
		"src_path\tdst_path\tmodule\tpen\tconfidence",
		"old/A\tmibs/A\tA", // only 3 fields
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadPlan(path); err == nil {
		t.Error("loadPlan accepted a 3-field row; want error")
	}
}

// TestVendorBucket pulls the {PEN}-{slug} segment from a dst path
// and reports "" for non-vendor destinations.
func TestVendorBucket(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"mibs/vendors/9-cisco/CISCO-MIB", "9-cisco"},
		{"mibs/vendors/22610-a10/A10-MIB", "22610-a10"},
		{"mibs/ietf/interfaces/IF-MIB", ""},
		{"mibs/iana/IANAifType-MIB", ""},
		{"", ""},
	}
	for _, c := range cases {
		got := vendorBucket(c.in)
		if got != c.want {
			t.Errorf("vendorBucket(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestWalkSourceFiltersSymlink asserts a symlink to a real MIB file
// is filtered out by the irregular-type guard — even though its
// target has the marker, descending the link could escape the
// corpus root.
func TestWalkSourceFiltersSymlink(t *testing.T) {
	root := t.TempDir()
	mib := func(name string) string { return name + " DEFINITIONS ::= BEGIN\nEND\n" }

	target := filepath.Join(t.TempDir(), "OUTSIDE-MIB")
	if err := os.WriteFile(target, []byte(mib("OUTSIDE-MIB")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "linked-mib")); err != nil {
		t.Skipf("os.Symlink unsupported: %v", err)
	}

	files, err := walkSource(root)
	if err != nil {
		t.Fatalf("walkSource: %v", err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "linked-mib") {
			t.Errorf("walkSource returned symlink %s; symlinks must be skipped", f)
		}
	}
}

// TestWalkSourceFindsRealMIB is the positive-path counterpart to
// TestWalkSourceFiltersNonMIBs — guards against a regression that
// over-aggressively filters everything.
func TestWalkSourceFindsRealMIB(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "vendors/9-cisco/CISCO-EXAMPLE-MIB")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("CISCO-EXAMPLE-MIB DEFINITIONS ::= BEGIN\nEND\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err := walkSource(root)
	if err != nil {
		t.Fatalf("walkSource: %v", err)
	}
	if len(files) != 1 || !strings.HasSuffix(files[0], "CISCO-EXAMPLE-MIB") {
		t.Errorf("walkSource didn't return the real MIB: %v", files)
	}
}

// TestWalkSourceFiltersNonMIBs asserts the lexical-marker gate keeps
// non-MIB files (LICENSE, README, _overrides.yaml, LICENSES/*.txt)
// out of the plan even when their extension matches the heuristic.
func TestWalkSourceFiltersNonMIBs(t *testing.T) {
	root := t.TempDir()
	mib := func(name string) string { return name + " DEFINITIONS ::= BEGIN\nEND\n" }
	cases := []struct {
		path, body string
	}{
		{filepath.Join(root, "ietf/core/SNMPv2-SMI"), mib("SNMPv2-SMI")},
		{filepath.Join(root, "vendors/9-cisco/CISCO-EXAMPLE-MIB"), mib("CISCO-EXAMPLE-MIB")},
		{filepath.Join(root, "LICENSE"), "Copyright (c) 2024\n"},
		{filepath.Join(root, "README.md"), "# corpus\n"},
		{filepath.Join(root, "_overrides.yaml"), "licenses: {}\n"},
		{filepath.Join(root, "LICENSES/cisco.txt"), "Cisco MIB License\n"},
		{filepath.Join(root, "vendors/9-cisco/SHIM.mib"), "not a mib\n"},
	}
	for _, c := range cases {
		if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(c.path, []byte(c.body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	files, err := walkSource(root)
	if err != nil {
		t.Fatalf("walkSource: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("walkSource returned %d files, want 2: %v", len(files), files)
	}
	for _, want := range []string{"SNMPv2-SMI", "CISCO-EXAMPLE-MIB"} {
		hit := false
		for _, f := range files {
			if strings.HasSuffix(f, "/"+want) {
				hit = true
			}
		}
		if !hit {
			t.Errorf("walkSource missed %s; got %v", want, files)
		}
	}
	// Negative assertions.
	for _, bad := range []string{"LICENSE", "README.md", "_overrides.yaml", "LICENSES/cisco.txt", "SHIM.mib"} {
		for _, f := range files {
			if strings.HasSuffix(f, bad) {
				t.Errorf("walkSource unexpectedly returned %s; got %v", bad, files)
			}
		}
	}
}

// TestDedupDestinations asserts that two entries with the same dst
// get re-routed to unsorted with low confidence.
func TestDedupDestinations(t *testing.T) {
	entries := []planEntry{
		{SrcPath: "old/A.mib", DstPath: "mibs/ietf/core/SAMENAME", Confidence: ConfidenceHigh},
		{SrcPath: "old/B.mib", DstPath: "mibs/ietf/core/SAMENAME", Confidence: ConfidenceHigh},
		{SrcPath: "old/C.mib", DstPath: "mibs/ietf/core/UNIQUE", Confidence: ConfidenceHigh},
	}
	got := dedupDestinations(entries, "mibs")
	if got != 2 {
		t.Errorf("dedupDestinations returned %d, want 2", got)
	}
	for _, e := range entries[:2] {
		if !strings.Contains(e.DstPath, "/unsorted/") {
			t.Errorf("dup %s not rerouted: %q", e.SrcPath, e.DstPath)
		}
		if e.Confidence != ConfidenceLow {
			t.Errorf("dup %s confidence = %s, want low", e.SrcPath, e.Confidence)
		}
	}
	if entries[2].DstPath != "mibs/ietf/core/UNIQUE" {
		t.Errorf("non-dup C touched: %q", entries[2].DstPath)
	}
}
