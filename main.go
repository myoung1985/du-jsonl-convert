// Command duconv converts between du's text output and JSON Lines, one
// entry at a time, so it can sit in a pipeline behind `du -a` on a large
// filesystem without buffering the whole tree.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "duconv:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("duconv", flag.ContinueOnError)
	from := fs.String("from", "", "input format: du or jsonl")
	to := fs.String("to", "", "output format: du or jsonl")
	in := fs.String("in", "-", "input file, - for stdin")
	out := fs.String("out", "-", "output file, - for stdout")
	null := fs.Bool("null", false, "use NUL instead of newline to separate du records (matches `du -a0`); needed for paths containing a literal newline")
	if err := fs.Parse(args); err != nil {
		return err
	}

	r, closeIn, err := openInput(*in)
	if err != nil {
		return err
	}
	defer closeIn()

	w, closeOut, err := openOutput(*out)
	if err != nil {
		return err
	}
	defer closeOut()

	switch {
	case *from == "du" && *to == "jsonl":
		return DuToJSONL(r, w, *null)
	case *from == "jsonl" && *to == "du":
		return JSONLToDu(r, w, *null)
	case *from == "" || *to == "":
		return fmt.Errorf("both -from and -to are required (du or jsonl)")
	case *from == *to:
		return fmt.Errorf("-from and -to are the same format (%s), nothing to convert", *from)
	default:
		return fmt.Errorf("unsupported conversion: %s -> %s", *from, *to)
	}
}

func openInput(path string) (io.Reader, func(), error) {
	if path == "-" {
		return os.Stdin, func() {}, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	return f, func() { f.Close() }, nil
}

func openOutput(path string) (io.Writer, func(), error) {
	if path == "-" {
		return os.Stdout, func() {}, nil
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	return f, func() { f.Close() }, nil
}
