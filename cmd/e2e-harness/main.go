// Command e2e-harness boots a blittermib server against an in-memory,
// deterministically-seeded corpus for end-to-end browser tests.
//
// It is intentionally NOT the production entry point: it needs no
// smidump, no on-disk corpus, and no assets directory (static files are
// embedded), so Playwright can launch it with a single `go run` and get
// a ready server in well under a second. The seed is the shared
// storetest.SeedIFMIB fixture, so the browser tests and the Go unit
// tests exercise the same corpus.
//
// Not built into the Docker image (the Dockerfile only builds
// cmd/blittermib); it exists purely for `make e2e` / CI.
package main

import (
	"cmp"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/no42-org/blittermib/internal/server"
	"github.com/no42-org/blittermib/internal/store"
	"github.com/no42-org/blittermib/internal/store/storetest"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("e2e-harness: %v", err)
	}
}

func run() error {
	// PORT lets Playwright's webServer pick the port; default matches the
	// config's baseURL for a bare `go run ./cmd/e2e-harness`. Loopback
	// only — this is a local test fixture, never a network service.
	addr := "127.0.0.1:" + cmp.Or(os.Getenv("PORT"), "8081")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	st, err := store.OpenInMemory(ctx)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer func() { _ = st.Close() }()

	if err := storetest.SeedIFMIB(ctx, st); err != nil {
		return fmt.Errorf("seed store: %w", err)
	}

	srv := server.New(st, addr, "e2e", "/nonexistent/mibs")
	// Production opens this from the boot goroutine once the first corpus
	// load finishes; here the seed is synchronous, so open it immediately
	// and /readyz serves 200 (what Playwright's webServer waits on).
	srv.SetReady()

	// Start reuses the production server lifecycle (the tuned http.Server
	// built by server.New, plus graceful shutdown), so the harness
	// exercises the same connection config that ships. It blocks until
	// ctx is cancelled (SIGINT/SIGTERM from Playwright's teardown).
	// fmt (not log) avoids gosec G706 on the env-derived addr; this is a
	// plain stdout status line, and Server.Start logs the real bind.
	fmt.Printf("e2e-harness listening on http://%s\n", addr)
	return srv.Start(ctx)
}
