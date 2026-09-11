package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseDuLine(t *testing.T) {
	cases := []struct {
		name    string
		line    string
		want    Entry
		wantErr bool
	}{
		{
			name: "tab separated",
			line: "4096\t/var/log",
			want: Entry{Bytes: 4096, Path: "/var/log"},
		},
		{
			name: "space separated fallback",
			line: "4096 /var/log",
			want: Entry{Bytes: 4096, Path: "/var/log"},
		},
		{
			name: "human readable kilobytes",
			line: "4.0K\t/var/log",
			want: Entry{Bytes: 4096, Path: "/var/log"},
		},
		{
			name: "human readable gigabytes",
			line: "1.5G\t/home/data",
			want: Entry{Bytes: int64(1.5 * (1 << 30)), Path: "/home/data"},
		},
		{
			name: "plain bytes unit suffix",
			line: "512B\t/tmp/x",
			want: Entry{Bytes: 512, Path: "/tmp/x"},
		},
		{
			name: "path with leading whitespace collapsed after separator",
			line: "10\t\t/weird/path",
			want: Entry{Bytes: 10, Path: "/weird/path"},
		},
		{
			name: "path containing spaces preserved",
			line: "10\t/has space/in it",
			want: Entry{Bytes: 10, Path: "/has space/in it"},
		},
		{
			name:    "no separator",
			line:    "4096",
			wantErr: true,
		},
		{
			name:    "unrecognized unit",
			line:    "4.0X\t/var/log",
			wantErr: true,
		},
		{
			name:    "empty size",
			line:    "\t/var/log",
			wantErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := parseDuLine(c.line)
			if c.wantErr {
				if err == nil {
					t.Fatalf("parseDuLine(%q) = %+v, nil; want error", c.line, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseDuLine(%q) returned unexpected error: %v", c.line, err)
			}
			if got != c.want {
				t.Fatalf("parseDuLine(%q) = %+v, want %+v", c.line, got, c.want)
			}
		})
	}
}

func TestDuToJSONLRoundTrip(t *testing.T) {
	input := "4096\t/var/log\n8192\t/var/log/syslog\n"

	var jsonl bytes.Buffer
	if err := DuToJSONL(strings.NewReader(input), &jsonl, false); err != nil {
		t.Fatalf("DuToJSONL: %v", err)
	}

	var du bytes.Buffer
	if err := JSONLToDu(&jsonl, &du, false); err != nil {
		t.Fatalf("JSONLToDu: %v", err)
	}

	if du.String() != input {
		t.Fatalf("round trip mismatch:\n got: %q\nwant: %q", du.String(), input)
	}
}

func TestDuToJSONLRoundTripNullDelim(t *testing.T) {
	input := "4096\t/var/log/with\na newline\x008192\t/other/path\x00"

	var jsonl bytes.Buffer
	if err := DuToJSONL(strings.NewReader(input), &jsonl, true); err != nil {
		t.Fatalf("DuToJSONL: %v", err)
	}

	var du bytes.Buffer
	if err := JSONLToDu(&jsonl, &du, true); err != nil {
		t.Fatalf("JSONLToDu: %v", err)
	}

	if du.String() != input {
		t.Fatalf("round trip mismatch:\n got: %q\nwant: %q", du.String(), input)
	}
}

func TestDuToJSONLBadRecord(t *testing.T) {
	input := "not-a-number\t/var/log\n"
	var out bytes.Buffer
	err := DuToJSONL(strings.NewReader(input), &out, false)
	if err == nil {
		t.Fatal("expected error for malformed du line, got nil")
	}
}
