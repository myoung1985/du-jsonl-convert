# du-jsonl-convert

`du` is great at walking a filesystem and printing sizes, but its output is a
flat, whitespace-delimited text format that's annoying to filter or reshape
with normal tools. JSON Lines is easy to filter with `jq`, load into a
database, or diff between two runs, but nothing produces it directly.

`duconv` converts between the two, one entry at a time. It never builds a
tree or buffers the whole input, so it's fine to point at `du -a /` on a
disk with millions of files.

## Usage

Convert du output to JSON Lines:

```
du -a -b / 2>/dev/null | duconv -from du -to jsonl > usage.jsonl
```

Each line of `usage.jsonl` looks like:

```
{"bytes":4096,"path":"/home/alice/.bashrc"}
```

Filter with jq without ever loading the whole file:

```
duconv -from du -to jsonl -in du-output.txt | jq 'select(.bytes > 100000000)'
```

Convert back to du's format:

```
duconv -from jsonl -to du -in usage.jsonl -out usage.du
```

By default input is read from stdin and output written to stdout; `-in` and
`-out` accept a file path or `-` for a stream.

Paths containing a literal newline break line-based du output. `du -a0`
writes NUL-terminated records instead of newline-terminated ones to avoid
that ambiguity; pass `-null` to read or write that form:

```
du -a0 -b / 2>/dev/null | duconv -from du -to jsonl -null > usage.jsonl
duconv -from jsonl -to du -null -in usage.jsonl -out usage.du
```

## Building

```
go build -o duconv .
```

## Current limitations

- `du -a -h` input (e.g. "4.0K", "1.5G") is accepted, but converting it to
  bytes is lossy: du only prints one decimal digit, so the resulting byte
  count is an approximation of the original size, not an exact match.
  Prefer `du -a -b` when an exact round trip matters.
