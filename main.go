// Command mdtlint checks markdown files for broken pipe tables: rows whose
// column count doesn't match the header, and malformed separator rows.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	fix := flag.Bool("fix", false, "auto-correct separator row problems in place")
	flag.Parse()
	args := flag.Args()

	if len(args) == 0 {
		os.Exit(runStdin(*fix))
	}

	dirty := false
	for _, path := range args {
		if runFile(path, *fix) {
			dirty = true
		}
	}
	if dirty {
		os.Exit(1)
	}
}

// runStdin lints or fixes stdin and returns the process exit code. In fix
// mode the corrected content is written to stdout; there's no file to
// rewrite in place.
func runStdin(fix bool) int {
	if fix {
		fixed, findings, err := Fix(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "mdtlint:", err)
			return 2
		}
		os.Stdout.Write(fixed)
		printFindings("stdin", findings)
		if len(findings) > 0 {
			return 1
		}
		return 0
	}

	findings, err := Lint(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mdtlint:", err)
		return 2
	}
	printFindings("stdin", findings)
	if len(findings) > 0 {
		return 1
	}
	return 0
}

// runFile lints or fixes the file at path, rewriting it in place in fix
// mode, and reports whether it has unresolved findings or hit an error.
func runFile(path string, fix bool) (dirty bool) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mdtlint:", err)
		return true
	}
	defer f.Close()

	if fix {
		fixed, findings, err := Fix(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mdtlint: %s: %v\n", path, err)
			return true
		}
		if err := os.WriteFile(path, fixed, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "mdtlint: %s: %v\n", path, err)
			return true
		}
		printFindings(path, findings)
		return len(findings) > 0
	}

	findings, err := Lint(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mdtlint: %s: %v\n", path, err)
		return true
	}
	printFindings(path, findings)
	return len(findings) > 0
}

func printFindings(source string, findings []Finding) {
	for _, f := range findings {
		fmt.Printf("%s:%d: %s\n", source, f.Line, f.Message)
	}
}
