package server

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/no42-org/blittermib/internal/mibcorpus"
	"github.com/no42-org/blittermib/internal/model"
)

// maxUploadFileSize is the per-file cap (D7). 10 MB covers all
// real-world standard and major-vendor MIBs with headroom; anything
// larger is rejected as "definitely not a MIB."
const maxUploadFileSize = 10 << 20

// uploadOutcome is the per-file response shape. JSON tags align with
// the spec scenarios in
// openspec/changes/web-upload/specs/web-upload/spec.md.
//
// httpStatus is internal — it lets the handler pick a meaningful
// HTTP status code on a single-file batch (200/400/409/413/422) so
// the spec's "the response is 413" scenarios hold without making the
// caller scrape an HTTP body to discover what went wrong. For
// multi-file batches, we always return 200 and the per-file outcomes
// carry the detail.
type uploadOutcome struct {
	Name        string             `json:"name"`
	OK          bool               `json:"ok"`
	Module      string             `json:"module,omitempty"`
	Symbols     int                `json:"symbols,omitempty"`
	OID         string             `json:"oid,omitempty"`
	Diagnostics []model.Diagnostic `json:"diagnostics,omitempty"`
	Replaced    bool               `json:"replaced,omitempty"`
	Error       string             `json:"error,omitempty"`

	httpStatus int // not serialised
}

type uploadResponse struct {
	Uploaded []uploadOutcome `json:"uploaded"`
}

// handleUpload implements the multi-file batched upload pipeline
// per design.md D4 + D5 + D6a:
//
//  1. validate each multipart part's filename (ValidModuleName)
//  2. enforce 10 MB cap per file via a LimitReader
//  3. atomic-write the bytes to mibs/upload/.tmp/<name>.upload
//  4. sniff the temp file with HasMIBOpener
//  5. on collision, refuse unless ?replace=true
//  6. rename(2) into mibs/upload/<name>
//  7. once ALL accepted parts are written, fire one loadFiles call
//     so smidump's IMPORTS resolver sees the whole batch on disk
//  8. emit per-file outcomes as JSON
//
// All step-level failures attach to the per-file outcome and the
// handler keeps processing remaining parts. The watcher will fire
// after the renames and recompile redundantly (D6b — accepted).
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	replaceQ := r.URL.Query().Get("replace")
	replace, _ := strconv.ParseBool(replaceQ)

	mr, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "expected multipart/form-data", http.StatusBadRequest)
		return
	}

	tmpDir := filepath.Join(s.uploadDir, ".tmp")
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		s.internalError(w, r, err)
		return
	}

	var outcomes []uploadOutcome
	var acceptedPaths []string

	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			http.Error(w, "malformed multipart body", http.StatusBadRequest)
			return
		}
		oc := s.processUploadPart(part, replace, tmpDir)
		_ = part.Close()
		outcomes = append(outcomes, oc)
		if oc.OK {
			acceptedPaths = append(acceptedPaths, filepath.Join(s.uploadDir, oc.Name))
		}
	}

	// D14 — write all parts first, fire loadFiles ONCE so smidump's
	// IMPORTS resolver sees prerequisites on disk regardless of part
	// arrival order in the multipart body.
	if len(acceptedPaths) > 0 && s.loadFiles != nil {
		results := s.loadFiles(r.Context(), acceptedPaths)
		byPath := make(map[string]LoadOutcome, len(results))
		for _, r := range results {
			byPath[r.Path] = r
		}
		for i := range outcomes {
			if !outcomes[i].OK {
				continue
			}
			path := filepath.Join(s.uploadDir, outcomes[i].Name)
			res, ok := byPath[path]
			if !ok {
				continue
			}
			if res.Err != nil {
				outcomes[i].OK = false
				outcomes[i].Error = "compile failed: " + res.Err.Error()
				continue
			}
			if res.Module != nil {
				outcomes[i].Module = res.Module.Name
				outcomes[i].OID = res.Module.OIDRoot
			}
			outcomes[i].Symbols = res.SymbolCount
			outcomes[i].Diagnostics = res.Diagnostics
		}
	}

	status := http.StatusOK
	if len(outcomes) == 1 && !outcomes[0].OK && outcomes[0].httpStatus != 0 {
		status = outcomes[0].httpStatus
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(uploadResponse{Uploaded: outcomes}); err != nil {
		slog.Warn("upload response encode failed", "err", err)
	}
}

// processUploadPart handles a single multipart part end-to-end:
// validate, write to tmp, sniff, rename. Returns an outcome that
// already has Name + OK + Error + httpStatus filled in; the
// post-batch compile pass adds Module/OID/Symbols/Diagnostics for
// successful entries.
func (s *Server) processUploadPart(part interface {
	FileName() string
	io.Reader
}, replace bool, tmpDir string) uploadOutcome {
	name := filepath.Base(part.FileName())
	oc := uploadOutcome{Name: name}

	if !mibcorpus.ValidModuleName.MatchString(name) {
		_, _ = io.Copy(io.Discard, part)
		oc.Error = "invalid filename"
		oc.httpStatus = http.StatusBadRequest
		return oc
	}

	target := filepath.Join(s.uploadDir, name)
	existed := false
	if _, err := os.Lstat(target); err == nil {
		existed = true
		if !replace {
			_, _ = io.Copy(io.Discard, part)
			oc.Error = "destination already exists"
			oc.httpStatus = http.StatusConflict
			return oc
		}
	}

	tmpPath := filepath.Join(tmpDir, name+".upload")
	f, err := os.Create(tmpPath)
	if err != nil {
		_, _ = io.Copy(io.Discard, part)
		oc.Error = "create tmp: " + err.Error()
		oc.httpStatus = http.StatusInternalServerError
		return oc
	}
	// LimitReader reads at most maxUploadFileSize+1 bytes; if we
	// actually pulled that many, the source exceeded the cap.
	n, copyErr := io.Copy(f, io.LimitReader(part, maxUploadFileSize+1))
	syncErr := f.Sync()
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmpPath)
		oc.Error = "io: " + copyErr.Error()
		oc.httpStatus = http.StatusInternalServerError
		return oc
	}
	if syncErr != nil || closeErr != nil {
		_ = os.Remove(tmpPath)
		oc.Error = "fsync/close failed"
		oc.httpStatus = http.StatusInternalServerError
		return oc
	}
	if n > maxUploadFileSize {
		_ = os.Remove(tmpPath)
		oc.Error = "file exceeds 10 MB limit"
		oc.httpStatus = http.StatusRequestEntityTooLarge
		return oc
	}

	ok, err := mibcorpus.HasMIBOpener(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		oc.Error = "sniff failed: " + err.Error()
		oc.httpStatus = http.StatusInternalServerError
		return oc
	}
	if !ok {
		_ = os.Remove(tmpPath)
		oc.Error = "no MIB marker"
		oc.httpStatus = http.StatusUnprocessableEntity
		return oc
	}

	if err := os.Rename(tmpPath, target); err != nil {
		_ = os.Remove(tmpPath)
		oc.Error = "rename: " + err.Error()
		oc.httpStatus = http.StatusInternalServerError
		return oc
	}

	oc.OK = true
	oc.Replaced = existed
	return oc
}

// handleUploadDelete removes a single file from mibs/upload/. The
// only accepted path shape is /api/v1/upload/<name> where <name>
// passes ValidModuleName; anything escaping mibs/upload/ via
// ../-style traversal or absolute paths is refused.
//
// On success, the watcher's debounced reload drops the corresponding
// module from the store within ~250 ms (per the existing fsnotify
// pipeline); we don't need to wire a synchronous unload here.
func (s *Server) handleUploadDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	const prefix = "/api/v1/upload/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}
	rawName := strings.TrimPrefix(r.URL.Path, prefix)
	// URL-decoded by net/http during routing; defend against any
	// surviving traversal characters by going through filepath.Base
	// + ValidModuleName.
	name := filepath.Base(rawName)
	if name == "" || name == "." || name == ".." {
		http.Error(w, "invalid name", http.StatusBadRequest)
		return
	}
	if !mibcorpus.ValidModuleName.MatchString(name) {
		http.Error(w, "invalid name", http.StatusBadRequest)
		return
	}
	target := filepath.Join(s.uploadDir, name)
	// Defence-in-depth: even after ValidModuleName, refuse anything
	// that doesn't resolve to a child of s.uploadDir.
	rel, err := filepath.Rel(s.uploadDir, target)
	if err != nil || !filepath.IsLocal(rel) {
		http.Error(w, "invalid name", http.StatusBadRequest)
		return
	}
	if err := os.Remove(target); err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		s.internalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleUploadIndex(w http.ResponseWriter, r *http.Request) {
	// §5.1–§5.2: list mibs/upload/, render templ with drop zone +
	// per-file rows + delete buttons.
	http.Error(w, "upload-index handler not yet implemented", http.StatusNotImplemented)
}
