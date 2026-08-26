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

## Building

```
go build -o duconv .
```

## Current limitations

- Only byte counts are supported as input (`du -a -b`, not `du -a -h`).
  Human-readable sizes like "4.0K" are ambiguous to parse back losslessly
  and aren't handled yet.
- Paths containing a literal newline break line-based du output; `du -a0`
  (null-separated) input isn't supported yet.
