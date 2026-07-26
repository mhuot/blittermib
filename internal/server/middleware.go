package server

import (
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
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
			"ip", clientIP(r),
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"bytes", rec.bytes,
			"dur", time.Since(start),
		)
	})
}

// clientIP returns the originating client address. The app's only
// ingress is the nginx-director reverse proxy, which sets X-Real-IP
// (to $remote_addr) and appends to X-Forwarded-For; the service is
// never exposed directly, so trusting those headers is safe here.
// Falls back to the transport peer for direct/local requests.
func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		// The first hop is the original client; later entries are proxies.
		if comma := strings.IndexByte(fwd, ','); comma >= 0 {
			return strings.TrimSpace(fwd[:comma])
		}
		return strings.TrimSpace(fwd)
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
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

// chain composes middlewares right-to-left so the first middleware in
// the argument list is the outermost.
func chain(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}
