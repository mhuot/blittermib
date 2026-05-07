package mibcorpus

import (
	"bytes"
	"io"
	"os"
)

// DefinitionsBeginMarker is the lexical anchor every SMIv2 module
// must contain. Used by the loader, mib-migrate, and mib-ingest as
// a fast gate to filter non-MIB files (LICENSE, README, partial
// downloads) without paying the libsmi parse cost.
var DefinitionsBeginMarker = []byte("DEFINITIONS ::= BEGIN")

// HasMIBOpener returns true when the first 32 KB of the file
// contains the `DEFINITIONS ::= BEGIN` marker. The 32 KB cap
// comfortably accommodates real-world Cisco/Juniper headers (which
// can run several KB of copyright/IPR boilerplate before the
// opener) while still keeping the per-file cost cheap on a multi-
// thousand-MIB corpus walk.
//
// Reads `sniffBytes + len(marker)-1` bytes to defend against the
// marker straddling the byte-N boundary — without the overlap a
// marker that spans bytes 32766–32786 would be missed.
//
// Uses `io.ReadFull` for short-read safety. An empty file or any
// short-read EOF flavour is reported as "no marker" without
// surfacing the EOF — those are non-MIBs, not I/O errors.
func HasMIBOpener(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	const sniffBytes = 32 * 1024
	buf := make([]byte, sniffBytes+len(DefinitionsBeginMarker)-1)
	n, err := io.ReadFull(f, buf)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return bytes.Contains(buf[:n], DefinitionsBeginMarker), nil
	}
	if err != nil {
		return false, err
	}
	return bytes.Contains(buf[:n], DefinitionsBeginMarker), nil
}
