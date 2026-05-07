package main

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
)

// licensePattern pairs a regex against the canonical tag emitted in
// INDEX.yaml. New patterns should be added alongside a matching
// `mibs/LICENSES/<tag>.txt` file so INDEX.yaml never references a
// missing license text.
type licensePattern struct {
	tag string
	re  *regexp.Regexp
}

// licensePatterns currently recognises only the rfc-editor boilerplate
// ("The Internet Society" / "IETF Trust"). The v1.0 starter set
// included Cisco/Juniper/HP(E)/Aruba/Huawei/A10/Mellanox/Brocade/
// Extreme entries; those were pruned when the corpus was reduced to
// just the standard libsmi seed. When a vendor MIB drop returns,
// re-add the relevant pattern here AND restore
// `mibs/LICENSES/<tag>.txt` in the same change.
var licensePatterns = []licensePattern{
	{tag: "rfc-editor", re: regexp.MustCompile(`(?i)\bCopyright\b[^\n]*(?:The Internet Society|IETF Trust)\b`)},
}

// licenseScanLines bounds how much of each MIB the detector reads —
// MIB headers reliably carry the copyright notice in the first ~200
// lines, and reading further wastes I/O on a 5000-file corpus.
const licenseScanLines = 200

// detectLicense returns the matching tag for the first ~200 lines of
// the supplied reader, or "unknown" when no pattern matches. A short
// reader is fine; the scanner exits at EOF. A scanner I/O error is
// returned alongside whatever partial classification we managed.
func detectLicense(r io.Reader) string {
	tag, _ := detectLicenseE(r)
	return tag
}

// detectLicenseE is the error-returning form. Used internally so
// callers that care about I/O errors during scanning can surface
// them.
func detectLicenseE(r io.Reader) (string, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)

	var head []byte
	for n := 0; n < licenseScanLines && sc.Scan(); n++ {
		head = append(head, sc.Bytes()...)
		head = append(head, '\n')
	}
	if err := sc.Err(); err != nil {
		return "unknown", fmt.Errorf("license scan: %w", err)
	}

	for _, p := range licensePatterns {
		if p.re.Match(head) {
			return p.tag, nil
		}
	}
	return "unknown", nil
}
