package server

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// sitemapNamespace is the XML namespace every sitemaps.org document
// must declare on its root element.
const sitemapNamespace = "http://www.sitemaps.org/schemas/sitemap/0.9"

// sitemapPageSize caps the number of <url> entries per sitemap file.
// The sitemaps protocol allows at most 50,000 URLs (and 50 MB
// uncompressed) per file; at the URL cap our short reference URLs stay
// far under the byte cap, so a single limit governs both. When the
// corpus exceeds one page, /sitemap.xml becomes an index pointing at
// /sitemaps/{n}.xml children.
const sitemapPageSize = 50000

// canonicalBaseURL derives the absolute scheme://host origin for the
// current request, used to build fully-qualified <loc> values (the
// sitemaps protocol requires absolute URLs).
//
// Behind a TLS-terminating reverse proxy — the recommended public
// deployment — the origin only sees plaintext HTTP, so the
// X-Forwarded-Proto header is honored first. A direct TLS listener
// falls back to r.TLS, and a plain HTTP origin to "http". r.Host
// reflects the requested host, which is the canonical hostname when a
// single front end fronts the deployment.
func canonicalBaseURL(r *http.Request) string {
	scheme := "http"
	if forwardedProto := r.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
		// A proxy chain may send a comma-separated list
		// ("https, http"); the first token is the client-facing scheme.
		if comma := strings.IndexByte(forwardedProto, ','); comma >= 0 {
			forwardedProto = forwardedProto[:comma]
		}
		scheme = strings.TrimSpace(forwardedProto)
	} else if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// sitemapPaths returns every crawlable, canonical path in the corpus,
// in a stable order: the static browse entry points, then one page per
// module, then one canonical page per symbol. Redirecting surfaces
// (/o/{oid}) and the query-parameterised workspace views are
// deliberately excluded — a sitemap should list only canonical,
// 200-returning URLs.
func (s *Server) sitemapPaths(ctx context.Context) ([]string, error) {
	paths := []string{"/", "/m", "/tree", "/diagnostics"}

	modules, err := s.store.ListModules(ctx)
	if err != nil {
		return nil, err
	}
	for _, module := range modules {
		paths = append(paths, "/m/"+module.Name)
	}

	symbolRefs, err := s.store.ListSymbolRefs(ctx)
	if err != nil {
		return nil, err
	}
	for _, ref := range symbolRefs {
		paths = append(paths, "/s/"+ref.Module+"::"+ref.Name)
	}
	return paths, nil
}

// sitemapURL is one <url> entry in a urlset.
type sitemapURL struct {
	Location string `xml:"loc"`
}

// sitemapURLSet is a single sitemap file listing concrete page URLs.
type sitemapURLSet struct {
	XMLName   xml.Name     `xml:"urlset"`
	Namespace string       `xml:"xmlns,attr"`
	URLs      []sitemapURL `xml:"url"`
}

// sitemapReference is one <sitemap> child entry in an index.
type sitemapReference struct {
	Location string `xml:"loc"`
}

// sitemapIndexDoc is the top-level index pointing at child sitemaps.
type sitemapIndexDoc struct {
	XMLName   xml.Name           `xml:"sitemapindex"`
	Namespace string             `xml:"xmlns,attr"`
	Sitemaps  []sitemapReference `xml:"sitemap"`
}

// handleSitemapIndex serves /sitemap.xml: an index that references one
// /sitemaps/{n}.xml child per page of URLs. An index is emitted even
// for a single page so crawlers always discover the same stable entry
// point regardless of corpus size.
func (s *Server) handleSitemapIndex(w http.ResponseWriter, r *http.Request) {
	paths, err := s.sitemapPaths(r.Context())
	if err != nil {
		http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
		return
	}
	pageCount := (len(paths) + sitemapPageSize - 1) / sitemapPageSize
	if pageCount < 1 {
		pageCount = 1
	}
	baseURL := canonicalBaseURL(r)
	index := sitemapIndexDoc{Namespace: sitemapNamespace}
	for pageNumber := 1; pageNumber <= pageCount; pageNumber++ {
		index.Sitemaps = append(index.Sitemaps, sitemapReference{
			Location: fmt.Sprintf("%s/sitemaps/%d.xml", baseURL, pageNumber),
		})
	}
	writeSitemapXML(w, index)
}

// handleSitemapPage serves /sitemaps/{n}.xml: page n (1-based) of the
// canonical URL list. Out-of-range or malformed page numbers 404.
func (s *Server) handleSitemapPage(w http.ResponseWriter, r *http.Request) {
	pageParam := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/sitemaps/"), ".xml")
	pageNumber, err := strconv.Atoi(pageParam)
	if err != nil || pageNumber < 1 {
		http.NotFound(w, r)
		return
	}
	paths, err := s.sitemapPaths(r.Context())
	if err != nil {
		http.Error(w, "sitemap unavailable", http.StatusInternalServerError)
		return
	}
	startIndex := (pageNumber - 1) * sitemapPageSize
	if startIndex >= len(paths) {
		http.NotFound(w, r)
		return
	}
	endIndex := startIndex + sitemapPageSize
	if endIndex > len(paths) {
		endIndex = len(paths)
	}
	baseURL := canonicalBaseURL(r)
	urlSet := sitemapURLSet{Namespace: sitemapNamespace}
	for _, path := range paths[startIndex:endIndex] {
		urlSet.URLs = append(urlSet.URLs, sitemapURL{Location: baseURL + path})
	}
	writeSitemapXML(w, urlSet)
}

// handleRobots serves /robots.txt: crawlers are welcomed to the
// canonical reference pages and steered away from dynamic, non-indexable
// surfaces (search, JSON APIs, the walk decoder, import/upload). The
// sitemap is advertised with an absolute URL so crawlers can discover
// the full corpus.
func (s *Server) handleRobots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	fmt.Fprintf(w, "User-agent: *\n"+
		"Disallow: /search\n"+
		"Disallow: /api/\n"+
		"Disallow: /walk\n"+
		"Disallow: /import\n"+
		"Disallow: /upload\n"+
		"\n"+
		"Sitemap: %s/sitemap.xml\n", canonicalBaseURL(r))
}

// writeSitemapXML renders an XML document with the declaration prolog
// and a one-hour cache window (the corpus changes rarely and the
// response is cheap to regenerate, so a CDN edge can hold it).
func writeSitemapXML(w http.ResponseWriter, document any) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	if _, err := w.Write([]byte(xml.Header)); err != nil {
		return
	}
	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")
	_ = encoder.Encode(document)
}
