package server

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"
)

// statusRecorder captures the HTTP status written by a handler so the
// access log can include it.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

// withLogging emits one slog INFO record per request with method, path,
// status, byte count, and duration. Health checks are demoted to DEBUG
// so they don't pollute the production log stream.
func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)

		level := slog.LevelInfo
		// Probe endpoints (liveness AND readiness) are hit every few
		// seconds by the kubelet / Docker healthcheck — keep both out
		// of the INFO stream.
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			level = slog.LevelDebug
		}
		slog.Log(r.Context(), level, "http",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"bytes", rec.bytes,
			"dur", time.Since(start),
		)
	})
}

// withRecover catches panics from a downstream handler, logs the stack,
// and serves a 500. Without this, a single bug would take the server
// down per request — http.Server recovers per-goroutine but the broken
// connection would already have shipped half a response.
func withRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic in handler",
					"path", r.URL.Path,
					"err", rec,
					"stack", string(debug.Stack()),
				)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// publicCacheWriter stamps a shared-cache Cache-Control header on
// successful responses and strips it from everything else. It is
// optimistic: the middleware pre-sets the header, then this writer
// removes it the moment a non-2xx status is seen (a redirect, 404, or
// error must never be cached as if it were canonical content).
//
// Both entry points are covered. templ handlers render with an implicit
// 200 — they call Write without WriteHeader — so Write forces a 200
// through WriteHeader first; handlers that set an explicit status hit
// WriteHeader directly. Either way the strip decision runs exactly once
// before headers flush.
type publicCacheWriter struct {
	http.ResponseWriter
	wroteHeader bool
}

func (w *publicCacheWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.wroteHeader = true
		if code != http.StatusOK && code != http.StatusPartialContent {
			w.Header().Del("Cache-Control")
		}
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *publicCacheWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// withPublicCache marks the deterministic, user-independent reference
// pages as cacheable by a shared cache (CDN edge / reverse proxy) so
// most requests never reach the origin. These pages carry no cookies
// and render identically for every visitor, so `public` is safe.
//
// max-age is short (browsers) while s-maxage is longer (shared caches),
// which the CDN can be told to purge on corpus reload. Dev builds send
// no-store instead: local edits change rendered HTML with no version
// bump, so nothing may be held. Only GET/HEAD are eligible.
func (s *Server) withPublicCache(next http.Handler) http.Handler {
	cacheControl := "public, max-age=300, s-maxage=3600"
	if s.version == "dev" {
		cacheControl = "no-store"
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", cacheControl)
		next.ServeHTTP(&publicCacheWriter{ResponseWriter: w}, r)
	})
}

// chain composes middlewares right-to-left so the first middleware in
// the argument list is the outermost.
func chain(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}
