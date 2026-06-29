package mibimport

import (
	"io"
	"log/slog"
	"testing"
	"time"
)

// TestParseCompileFloor covers BLITTERMIB_COMPILE_TIMEOUT resolution:
// the floor is raise-only, and empty / sub-default / unparseable /
// non-positive values fall back to defaultCompileFloor.
func TestParseCompileFloor(t *testing.T) {
	// The sub-default case intentionally triggers parseCompileFloor's
	// warning; discard logs so `go test` output stays clean.
	saved := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
	t.Cleanup(func() { slog.SetDefault(saved) })

	cases := []struct {
		name string
		in   string
		want time.Duration
	}{
		{"empty -> default", "", defaultCompileFloor},
		{"raise floor", "20m", 20 * time.Minute},
		{"exactly default -> default", "5m", defaultCompileFloor},
		{"sub-default -> default (warns)", "30s", defaultCompileFloor},
		{"unparseable -> default", "garbage", defaultCompileFloor},
		{"zero -> default", "0s", defaultCompileFloor},
		{"negative -> default", "-5m", defaultCompileFloor},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := parseCompileFloor(c.in); got != c.want {
				t.Errorf("parseCompileFloor(%q) = %s, want %s", c.in, got, c.want)
			}
		})
	}
}

// TestScaledCompileTimeout covers the 1 s/file scaling against the
// floor. compileFloor is pinned per case (and restored) so the test is
// hermetic regardless of BLITTERMIB_COMPILE_TIMEOUT in the environment.
func TestScaledCompileTimeout(t *testing.T) {
	saved := compileFloor
	t.Cleanup(func() { compileFloor = saved })

	cases := []struct {
		name  string
		floor time.Duration
		n     int
		want  time.Duration
	}{
		{"default floor, small batch -> floor", defaultCompileFloor, 1, defaultCompileFloor},
		{"default floor, large batch -> scaling wins", defaultCompileFloor, 1000, 1000 * time.Second},
		{"raised floor wins over smaller scaling", 20 * time.Minute, 1000, 20 * time.Minute},
		{"raised floor, larger scaling wins", 20 * time.Minute, 2000, 2000 * time.Second},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			compileFloor = c.floor
			if got := scaledCompileTimeout(c.n); got != c.want {
				t.Errorf("scaledCompileTimeout(%d) with floor %s = %s, want %s", c.n, c.floor, got, c.want)
			}
		})
	}
}
