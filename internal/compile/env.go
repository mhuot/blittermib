package compile

import (
	"os"
	"strings"
)

// smiEnv returns an environment slice for an smidump / smilint
// subprocess that injects `SMIPATH=<paths joined by ':'>`. Used
// instead of repeated `-p` flags because libsmi 0.5.0 treats the
// two differently:
//
//   - `-p path` triggers libsmi's directory introspection ("unable
//     to determine SMI version") and silently skips paths that
//     don't contain a recognised collection layout (`iana`, `ietf`,
//     `site` subdirs). Flat directories like `mibs/upload/` or
//     `mibs/vendors/9-cisco/` get dropped, so IMPORTS resolution
//     fails for any file in them.
//   - `SMIPATH` env var bypasses the introspection and treats every
//     entry as a flat MIB directory. Always works for the corpus
//     subdirs we care about.
//
// SMIPATH overrides libsmi's compiled-in default search path, so the
// caller is responsible for including every directory needed —
// typically the entire recursive expansion of the corpus root.
//
// An empty paths slice yields an unset SMIPATH (libsmi falls back
// to its default), preserving the prior behaviour for callers that
// pass no paths at all.
func smiEnv(paths []string) []string {
	env := os.Environ()
	if len(paths) == 0 {
		return env
	}
	out := make([]string, 0, len(env)+1)
	for _, e := range env {
		if strings.HasPrefix(e, "SMIPATH=") {
			continue
		}
		out = append(out, e)
	}
	out = append(out, "SMIPATH="+strings.Join(paths, ":"))
	return out
}
