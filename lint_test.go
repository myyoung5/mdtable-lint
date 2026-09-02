package main

import (
	"strings"
	"testing"
)

func TestLintValidTable(t *testing.T) {
	input := "| a | b |\n| --- | --- |\n| 1 | 2 |\n"
	findings, err := Lint(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %v", findings)
	}
}

func TestLintRowColumnMismatch(t *testing.T) {
	input := "| a | b |\n| --- | --- |\n| 1 | 2 | 3 |\n"
	findings, err := Lint(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %v", findings)
	}
	if findings[0].Line != 3 {
		t.Errorf("expected finding on line 3, got line %d", findings[0].Line)
	}
}

func TestLintInvalidSeparatorCell(t *testing.T) {
	input := "| a | b |\n| --- | == |\n| 1 | 2 |\n"
	findings, err := Lint(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %v", findings)
	}
}

func TestLintSkipsFencedCodeBlock(t *testing.T) {
	input := "```\n| a | b | c |\n| --- | --- |\n```\n"
	findings, err := Lint(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings inside fenced block, got %v", findings)
	}
}

func TestLintSkipsTildeFence(t *testing.T) {
	input := "~~~\n| a | b | c |\n| --- | --- |\n~~~\n"
	findings, err := Lint(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings inside tilde fence, got %v", findings)
	}
}

func TestLintChecksTableAfterFencedBlock(t *testing.T) {
	input := "```\ncode\n```\n\n| a | b |\n| --- | --- |\n| 1 | 2 | 3 |\n"
	findings, err := Lint(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding after fenced block, got %v", findings)
	}
	if findings[0].Line != 7 {
		t.Errorf("expected finding on line 7, got line %d", findings[0].Line)
	}
}

func TestLintUnclosedFenceSkipsRestOfInput(t *testing.T) {
	input := "```\n| a | b | c |\n| --- | --- |\n"
	findings, err := Lint(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings in unclosed fence, got %v", findings)
	}
}

func TestFenceMarkerShortRunDoesNotCloseLongerFence(t *testing.T) {
	// A closing fence must be at least as long as the opening one, so a
	// 3-backtick line can't close a 4-backtick fence.
	char, count, rest, ok := fenceMarker("```")
	if !ok || char != '`' || count != 3 || rest != "" {
		t.Fatalf("fenceMarker(\"```\") = %q, %d, %q, %v", char, count, rest, ok)
	}
	if count >= 4 {
		t.Fatalf("expected count 3 to be less than an opening fence of 4, got %d", count)
	}
}

func TestSplitRowIgnoresPipeInCodeSpan(t *testing.T) {
	cells := splitRow("| `a|b` | c |")
	if len(cells) != 2 {
		t.Fatalf("expected 2 cells, got %v", cells)
	}
	if cells[0] != "`a|b`" {
		t.Errorf("expected first cell to keep the code span intact, got %q", cells[0])
	}
	if cells[1] != "c" {
		t.Errorf("expected second cell %q, got %q", "c", cells[1])
	}
}

func TestSplitRowIgnoresEscapedPipeInCodeSpan(t *testing.T) {
	cells := splitRow("| `a\\|b` | c |")
	if len(cells) != 2 {
		t.Fatalf("expected 2 cells, got %v", cells)
	}
	if cells[0] != "`a\\|b`" {
		t.Errorf("expected first cell %q, got %q", "`a\\|b`", cells[0])
	}
}

func TestSplitRowHandlesUnmatchedBacktickRun(t *testing.T) {
	// A single backtick with no closing run of the same length never closes
	// the code span, so every remaining pipe is swallowed as literal text.
	cells := splitRow("| `a|b|c |")
	if len(cells) != 1 {
		t.Fatalf("expected 1 cell, got %v", cells)
	}
}

func TestLintTableWithCodeSpanCellDoesNotMisreport(t *testing.T) {
	input := "| a | b |\n| --- | --- |\n| `x|y` | 2 |\n"
	findings, err := Lint(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Lint returned error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %v", findings)
	}
}

func TestIsDelimiterRowAcceptsMalformedCell(t *testing.T) {
	// The row must still be recognized as an attempted separator so
	// lintTable's stricter per-cell check gets a chance to report it;
	// otherwise a typo like "==" makes the whole table invisible.
	if !isDelimiterRow("| --- | == |") {
		t.Fatal("expected row with a malformed separator cell to still be recognized as a delimiter row")
	}
}

func TestIsDelimiterRowRejectsOrdinaryTextRow(t *testing.T) {
	if isDelimiterRow("| foo | bar |") {
		t.Fatal("expected a row of ordinary words not to be mistaken for a delimiter row")
	}
}

func TestFenceMarkerRejectsIndentedFence(t *testing.T) {
	if _, _, _, ok := fenceMarker("    ```"); ok {
		t.Fatalf("expected a 4-space indented fence to not be recognized")
	}
}
