package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Golden files pair a realistic markdown document under testdata/*.md with
// the exact findings Lint should report for it, one "line: message" per
// line, in the matching testdata/*.golden. That keeps the fixtures readable
// as plain markdown instead of escaped Go string literals.
func TestLintGolden(t *testing.T) {
	inputs, err := filepath.Glob("testdata/*.md")
	if err != nil {
		t.Fatalf("glob testdata: %v", err)
	}
	if len(inputs) == 0 {
		t.Fatal("no golden inputs found under testdata/")
	}

	for _, inputPath := range inputs {
		name := strings.TrimSuffix(filepath.Base(inputPath), ".md")
		t.Run(name, func(t *testing.T) {
			f, err := os.Open(inputPath)
			if err != nil {
				t.Fatalf("open %s: %v", inputPath, err)
			}
			defer f.Close()

			findings, err := Lint(f)
			if err != nil {
				t.Fatalf("Lint(%s): %v", inputPath, err)
			}

			var got strings.Builder
			for _, finding := range findings {
				fmt.Fprintf(&got, "%d: %s\n", finding.Line, finding.Message)
			}

			goldenPath := filepath.Join("testdata", name+".golden")
			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read %s: %v", goldenPath, err)
			}

			if got.String() != string(want) {
				t.Errorf("Lint(%s) mismatch\ngot:\n%s\nwant:\n%s", inputPath, got.String(), string(want))
			}
		})
	}
}
