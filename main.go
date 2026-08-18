// Command mdtlint checks markdown files for broken pipe tables: rows whose
// column count doesn't match the header, and malformed separator rows.
package main

import (
	"fmt"
	"os"
)

func main() {
	args := os.Args[1:]

	if len(args) == 0 {
		findings, err := Lint(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "mdtlint:", err)
			os.Exit(2)
		}
		printFindings("stdin", findings)
		if len(findings) > 0 {
			os.Exit(1)
		}
		return
	}

	dirty := false
	for _, path := range args {
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "mdtlint:", err)
			dirty = true
			continue
		}
		findings, err := Lint(f)
		f.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "mdtlint: %s: %v\n", path, err)
			dirty = true
			continue
		}
		printFindings(path, findings)
		if len(findings) > 0 {
			dirty = true
		}
	}
	if dirty {
		os.Exit(1)
	}
}

func printFindings(source string, findings []Finding) {
	for _, f := range findings {
		fmt.Printf("%s:%d: %s\n", source, f.Line, f.Message)
	}
}
