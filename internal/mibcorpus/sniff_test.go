package mibcorpus

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHasMIBOpener spot-checks the lexical gate that filters out
// LICENSE / README / partial-write garbage from the recursive walks
// that the loader, mib-migrate, and mib-ingest perform.
func TestHasMIBOpener(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		name, body string
		want       bool
	}{
		{"with-marker", "FOO-MIB DEFINITIONS ::= BEGIN\nEND\n", true},
		{"with-leading-comments", "-- header\n-- more header\nFOO DEFINITIONS ::= BEGIN\n", true},
		{"multi-space-between-tokens", "APPC-MIB DEFINITIONS        ::= BEGIN\nEND\n", true},
		{"tab-between-tokens", "APPC-MIB DEFINITIONS\t::=\tBEGIN\nEND\n", true},
		{"newline-between-tokens", "APPC-MIB DEFINITIONS\n::=\nBEGIN\nEND\n", true},
		{"no-marker", "Copyright (c) 2024 ...\nNot a MIB at all.\n", false},
		{"empty", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(dir, c.name)
			if err := os.WriteFile(path, []byte(c.body), 0o644); err != nil {
				t.Fatal(err)
			}
			got, err := HasMIBOpener(path)
			if err != nil {
				t.Fatalf("HasMIBOpener: %v", err)
			}
			if got != c.want {
				t.Errorf("HasMIBOpener(%q) = %v, want %v", c.name, got, c.want)
			}
		})
	}
}

// TestHasMIBOpenerStraddlesBoundary covers the overlap region: a
// marker that lands across the 32 KB sniff boundary should still be
// detected because the buffer reads `sniffBytes + len(marker)-1`.
func TestHasMIBOpenerStraddlesBoundary(t *testing.T) {
	const sniffBytes = 32 * 1024
	dir := t.TempDir()
	// Place the marker so its first byte lies at sniffBytes-2 (the
	// last 2 bytes of the 32 KB block + the rest in the overlap
	// region). Without the overlap read, this would be missed.
	prefixLen := sniffBytes - 2
	body := strings.Repeat("-", prefixLen) + "DEFINITIONS ::= BEGIN" + "\n"
	path := filepath.Join(dir, "STRADDLE-MIB")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := HasMIBOpener(path)
	if err != nil {
		t.Fatalf("HasMIBOpener: %v", err)
	}
	if !got {
		t.Errorf("marker straddling 32 KB boundary not detected")
	}
}
