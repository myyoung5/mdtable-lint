// Command mdtlint checks markdown files for broken pipe tables: rows whose
// column count doesn't match the header, and malformed separator rows.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

// sourceFinding is a Finding tagged with the file it came from, so findings
// from multiple files can be merged into one JSON report.
type sourceFinding struct {
	Source  string `json:"source"`
	Line    int    `json:"line"`
	Message string `json:"message"`
}

func main() {
	fix := flag.Bool("fix", false, "auto-correct separator row problems in place")
	jsonOut := flag.Bool("json", false, "report findings as a JSON array instead of plain text")
	flag.Parse()
	args := flag.Args()

	var all []sourceFinding
	dirty := false

	if len(args) == 0 {
		findings, code := runStdin(*fix)
		if code == 2 {
			os.Exit(2)
		}
		all = append(all, findings...)
		dirty = len(findings) > 0
	} else {
		for _, path := range args {
			findings, fileDirty := runFile(path, *fix)
			all = append(all, findings...)
			if fileDirty {
				dirty = true
			}
		}
	}

	if *jsonOut {
		printJSON(all)
	} else {
		for _, f := range all {
			fmt.Printf("%s:%d: %s\n", f.Source, f.Line, f.Message)
		}
	}

	if dirty {
		os.Exit(1)
	}
}

// runStdin lints or fixes stdin, returning its findings and an exit code of
// 2 if reading failed. In fix mode the corrected content is written to
// stdout; there's no file to rewrite in place.
func runStdin(fix bool) ([]sourceFinding, int) {
	if fix {
		fixed, findings, err := Fix(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "mdtlint:", err)
			return nil, 2
		}
		os.Stdout.Write(fixed)
		return tagFindings("stdin", findings), 0
	}

	findings, err := Lint(os.Stdin)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mdtlint:", err)
		return nil, 2
	}
	return tagFindings("stdin", findings), 0
}

// runFile lints or fixes the file at path, rewriting it in place in fix
// mode, and reports whether it has unresolved findings or hit an error (in
// which case findings is nil).
func runFile(path string, fix bool) (findings []sourceFinding, dirty bool) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mdtlint:", err)
		return nil, true
	}
	defer f.Close()

	if fix {
		fixed, tableFindings, err := Fix(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "mdtlint: %s: %v\n", path, err)
			return nil, true
		}
		if err := os.WriteFile(path, fixed, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "mdtlint: %s: %v\n", path, err)
			return nil, true
		}
		tagged := tagFindings(path, tableFindings)
		return tagged, len(tagged) > 0
	}

	tableFindings, err := Lint(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mdtlint: %s: %v\n", path, err)
		return nil, true
	}
	tagged := tagFindings(path, tableFindings)
	return tagged, len(tagged) > 0
}

func tagFindings(source string, findings []Finding) []sourceFinding {
	tagged := make([]sourceFinding, len(findings))
	for i, f := range findings {
		tagged[i] = sourceFinding{Source: source, Line: f.Line, Message: f.Message}
	}
	return tagged
}

// printJSON writes findings as a JSON array to stdout, or "[]" if there are
// none, so editor integrations always get a parseable document.
func printJSON(findings []sourceFinding) {
	if findings == nil {
		findings = []sourceFinding{}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(findings); err != nil {
		fmt.Fprintln(os.Stderr, "mdtlint:", err)
	}
}
