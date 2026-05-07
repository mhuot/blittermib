package main

import (
	"strings"
	"testing"
)

func TestLicenseDetector(t *testing.T) {
	cases := []struct {
		name, header, want string
	}{
		{"rfc-editor (Internet Society)", "-- Copyright (c) 2009 The Internet Society\n", "rfc-editor"},
		{"rfc-editor (IETF Trust)", "-- Copyright (c) 2024 IETF Trust and the persons identified\n", "rfc-editor"},
		{"unknown (no copyright)", "-- A header without any copyright line\n", "unknown"},
		{"unknown (vendor copyright, no pattern)", "-- Copyright Some Random Vendor LLC\n", "unknown"},
		{"unknown (Cisco — pattern was pruned)", "-- Copyright (c) 2024 Cisco Systems, Inc.\n", "unknown"},
		{"empty", "", "unknown"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := detectLicense(strings.NewReader(c.header))
			if got != c.want {
				t.Errorf("detectLicense(%q) = %q, want %q", c.header, got, c.want)
			}
		})
	}
}

// TestLicenseDetectorBoundedScan asserts the detector doesn't read
// past the configured line cap — a "Copyright ... IETF Trust" line
// buried below 200 lines should NOT match.
func TestLicenseDetectorBoundedScan(t *testing.T) {
	var buf strings.Builder
	for i := 0; i < licenseScanLines+10; i++ {
		buf.WriteString("-- filler\n")
	}
	buf.WriteString("-- Copyright IETF Trust\n")
	if got := detectLicense(strings.NewReader(buf.String())); got != "unknown" {
		t.Errorf("scan exceeded %d-line cap: got %q, want unknown", licenseScanLines, got)
	}
}
