package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"
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

// DuToJSONL reads du's "<bytes>\t<path>" records from r and writes one JSON
// object per line to w. It processes a record at a time, so the size of the
// input has no bearing on memory use.
//
// If nullDelim is true, records are read up to a NUL byte instead of a
// newline, matching `du -a0` output. That lets a path containing a literal
// newline pass through intact, since only NUL terminates a record.
func DuToJSONL(r io.Reader, w io.Writer, nullDelim bool) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), maxDuLine)
	if nullDelim {
		sc.Split(scanNullDelim)
	}

	enc := json.NewEncoder(w)
	recordNum := 0
	for sc.Scan() {
		recordNum++
		line := sc.Text()
		if line == "" {
			continue
		}
		e, err := parseDuLine(line)
		if err != nil {
			return fmt.Errorf("record %d: %w", recordNum, err)
		}
		if err := enc.Encode(e); err != nil {
			return fmt.Errorf("record %d: writing json: %w", recordNum, err)
		}
	}
	return sc.Err()
}

// scanNullDelim is a bufio.SplitFunc that splits on a NUL byte instead of
// a newline, mirroring bufio.ScanLines. du writes NUL-terminated records
// with the -0 flag so paths containing a literal newline are unambiguous.
func scanNullDelim(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, 0); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// JSONLToDu reads one JSON entry per line from r and writes du-style
// "<bytes>\t<path>" records to w, decoding one object at a time.
//
// If nullDelim is true, each record is terminated with a NUL byte instead
// of a newline, so a path containing a literal newline round-trips back to
// a form `du -a0`-style consumers expect.
func JSONLToDu(r io.Reader, w io.Writer, nullDelim bool) error {
	dec := json.NewDecoder(r)
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	term := "\n"
	if nullDelim {
		term = "\x00"
	}

	for {
		var e Entry
		err := dec.Decode(&e)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("decoding json: %w", err)
		}
		if _, err := fmt.Fprintf(bw, "%d\t%s%s", e.Bytes, e.Path, term); err != nil {
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
	size, err := parseSize(line[:idx])
	if err != nil {
		return Entry{}, fmt.Errorf("bad size %q: %w", line[:idx], err)
	}
	path := strings.TrimLeft(line[idx+1:], " \t")
	return Entry{Bytes: size, Path: path}, nil
}

// sizeUnits maps the single-letter suffixes `du -h` appends to its
// human-readable sizes to their byte multiplier. du uses binary (1024-based)
// units, not decimal ones, even though the letters look like SI prefixes.
var sizeUnits = map[rune]int64{
	'B': 1,
	'K': 1 << 10,
	'M': 1 << 20,
	'G': 1 << 30,
	'T': 1 << 40,
	'P': 1 << 50,
	'E': 1 << 60,
}

// parseSize accepts either a plain byte count (from `du -a -b`) or a
// human-readable size like "4.0K" or "1.5G" (from `du -a -h`). The
// human-readable form is lossy: du prints one decimal digit, so the
// resulting byte count is an approximation, not the exact original value.
func parseSize(s string) (int64, error) {
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		return n, nil
	}
	if s == "" {
		return 0, fmt.Errorf("empty size")
	}
	unit, ok := sizeUnits[unicode.ToUpper(rune(s[len(s)-1]))]
	if !ok {
		return 0, fmt.Errorf("unrecognized unit")
	}
	f, err := strconv.ParseFloat(s[:len(s)-1], 64)
	if err != nil {
		return 0, fmt.Errorf("bad numeric part: %w", err)
	}
	return int64(f * float64(unit)), nil
}
