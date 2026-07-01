package server

import (
	"net/http"
	"strings"
	"testing"
)

// TestPublicCacheOnReferencePages checks canonical reference pages are
// marked cacheable by a shared cache while non-200 responses and
// dynamic surfaces are not.
func TestPublicCacheOnReferencePages(t *testing.T) {
	ts := newTestServer(t)
	// newTestServer follows redirects by default; disable that so the
	// /o/ redirect's own headers can be inspected.
	client := &http.Client{
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	cases := []struct {
		path       string
		wantStatus int
		wantCached bool
	}{
		{"/s/IF-MIB::ifTable", http.StatusOK, true},
		{"/m/IF-MIB", http.StatusOK, true},
		{"/tree", http.StatusOK, true},
		{"/", http.StatusOK, true},
		// Unknown symbol → 404, header must be stripped.
		{"/s/IF-MIB::doesNotExist", http.StatusNotFound, false},
		// OID lookup is a redirect — never cached as canonical content.
		{"/o/1.3.6.1.2.1.2.2", http.StatusFound, false},
		// Search is dynamic and not wrapped at all.
		{"/search?q=if", http.StatusOK, false},
	}

	for _, tc := range cases {
		resp, err := client.Get(ts.URL + tc.path)
		if err != nil {
			t.Fatalf("GET %s: %v", tc.path, err)
		}
		if resp.StatusCode != tc.wantStatus {
			t.Errorf("%s: status = %d, want %d", tc.path, resp.StatusCode, tc.wantStatus)
		}
		cacheControl := resp.Header.Get("Cache-Control")
		isCached := strings.Contains(cacheControl, "public")
		if isCached != tc.wantCached {
			t.Errorf("%s: Cache-Control = %q, wantCached = %v", tc.path, cacheControl, tc.wantCached)
		}
		_ = body(t, resp)
	}
}
