package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Entry mirrors one line of `du -a -b` output: a byte count and a path.
type Entry struct {
	Bytes int64  `json:"bytes"`
	Path  string `json:"path"`
}

// maxDuLine bounds a single du line. Paths on real filesystems don't get
// anywhere near this; it just keeps a corrupt input from growing the
// scanner's buffer without limit.
const maxDuLine = 1 << 20

// DuToJSONL reads du's "<bytes>\t<path>" lines from r and writes one JSON
// object per line to w. It processes a line at a time, so the size of the
// input has no bearing on memory use.
func DuToJSONL(r io.Reader, w io.Writer) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), maxDuLine)

	enc := json.NewEncoder(w)
	lineNum := 0
	for sc.Scan() {
		lineNum++
		line := sc.Text()
		if line == "" {
			continue
		}
		e, err := parseDuLine(line)
		if err != nil {
			return fmt.Errorf("line %d: %w", lineNum, err)
		}
		if err := enc.Encode(e); err != nil {
			return fmt.Errorf("line %d: writing json: %w", lineNum, err)
		}
	}
	return sc.Err()
}

// JSONLToDu reads one JSON entry per line from r and writes du-style
// "<bytes>\t<path>" lines to w, decoding one object at a time.
func JSONLToDu(r io.Reader, w io.Writer) error {
	dec := json.NewDecoder(r)
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	for {
		var e Entry
		err := dec.Decode(&e)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("decoding json: %w", err)
		}
		if _, err := fmt.Fprintf(bw, "%d\t%s\n", e.Bytes, e.Path); err != nil {
			return err
		}
	}
	return bw.Flush()
}

// parseDuLine splits on the first tab, matching the format du emits by
// default. It falls back to the first run of whitespace so output from
// `du -a` piped through tools that collapse tabs to spaces still parses.
func parseDuLine(line string) (Entry, error) {
	idx := strings.IndexByte(line, '\t')
	if idx < 0 {
		idx = strings.IndexByte(line, ' ')
	}
	if idx < 0 {
		return Entry{}, fmt.Errorf("no separator between size and path: %q", line)
	}
	size, err := strconv.ParseInt(line[:idx], 10, 64)
	if err != nil {
		return Entry{}, fmt.Errorf("bad size %q: %w", line[:idx], err)
	}
	path := strings.TrimLeft(line[idx+1:], " \t")
	return Entry{Bytes: size, Path: path}, nil
}
