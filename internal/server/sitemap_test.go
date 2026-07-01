package server

import (
	"net/http"
	"strings"
	"testing"
)

// TestRobotsTxt checks the robots response steers crawlers away from
// dynamic surfaces and advertises an absolute sitemap URL.
func TestRobotsTxt(t *testing.T) {
	ts := newTestServer(t)

	resp, err := http.Get(ts.URL + "/robots.txt")
	if err != nil {
		t.Fatalf("GET /robots.txt: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if contentType := resp.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain", contentType)
	}
	robotsBody := body(t, resp)
	for _, want := range []string{
		"User-agent: *",
		"Disallow: /api/",
		"Disallow: /search",
		"Sitemap: " + ts.URL + "/sitemap.xml",
	} {
		if !strings.Contains(robotsBody, want) {
			t.Errorf("robots.txt missing %q\n%s", want, robotsBody)
		}
	}
}

// TestSitemapIndex checks /sitemap.xml is a well-formed index that
// references at least the first child page.
func TestSitemapIndex(t *testing.T) {
	ts := newTestServer(t)

	resp, err := http.Get(ts.URL + "/sitemap.xml")
	if err != nil {
		t.Fatalf("GET /sitemap.xml: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if contentType := resp.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "application/xml") {
		t.Errorf("Content-Type = %q, want application/xml", contentType)
	}
	indexBody := body(t, resp)
	for _, want := range []string{
		"<sitemapindex",
		ts.URL + "/sitemaps/1.xml",
	} {
		if !strings.Contains(indexBody, want) {
			t.Errorf("sitemap index missing %q\n%s", want, indexBody)
		}
	}
}

// TestSitemapPage checks page 1 lists the seeded module and symbol
// canonical URLs, and that an out-of-range page 404s.
func TestSitemapPage(t *testing.T) {
	ts := newTestServer(t)

	resp, err := http.Get(ts.URL + "/sitemaps/1.xml")
	if err != nil {
		t.Fatalf("GET /sitemaps/1.xml: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	pageBody := body(t, resp)
	for _, want := range []string{
		"<urlset",
		ts.URL + "/m/IF-MIB",
		ts.URL + "/s/IF-MIB::ifTable",
		ts.URL + "/s/IF-MIB::ifInOctets",
	} {
		if !strings.Contains(pageBody, want) {
			t.Errorf("sitemap page missing %q\n%s", want, pageBody)
		}
	}

	// Only one page of URLs is seeded, so page 2 must not exist.
	resp2, err := http.Get(ts.URL + "/sitemaps/2.xml")
	if err != nil {
		t.Fatalf("GET /sitemaps/2.xml: %v", err)
	}
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("page 2 status = %d, want 404", resp2.StatusCode)
	}
	_ = body(t, resp2)
}
