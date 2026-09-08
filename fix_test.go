package main

import (
	"strings"
	"testing"
)

func TestFixPadsShortSeparator(t *testing.T) {
	input := "| a | b | c |\n| --- | --- |\n| 1 | 2 | 3 |\n"
	got, findings, err := Fix(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Fix returned error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no remaining findings, got %v", findings)
	}
	want := "| a | b | c |\n| --- | --- | --- |\n| 1 | 2 | 3 |\n"
	if string(got) != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFixTruncatesLongSeparator(t *testing.T) {
	input := "| a | b |\n| --- | --- | --- |\n| 1 | 2 |\n"
	got, findings, err := Fix(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Fix returned error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no remaining findings, got %v", findings)
	}
	want := "| a | b |\n| --- | --- |\n| 1 | 2 |\n"
	if string(got) != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFixReplacesInvalidSeparatorCell(t *testing.T) {
	input := "| a | b |\n| --- | == |\n| 1 | 2 |\n"
	got, findings, err := Fix(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Fix returned error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no remaining findings, got %v", findings)
	}
	want := "| a | b |\n| --- | --- |\n| 1 | 2 |\n"
	if string(got) != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFixPreservesAlignmentColons(t *testing.T) {
	input := "| a | b |\n| :--- | ---: |\n| 1 | 2 |\n"
	got, _, err := Fix(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Fix returned error: %v", err)
	}
	want := input
	if string(got) != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestFixLeavesRowMismatchAsFinding(t *testing.T) {
	input := "| a | b |\n| --- | --- |\n| 1 | 2 | 3 |\n"
	got, findings, err := Fix(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Fix returned error: %v", err)
	}
	if string(got) != input {
		t.Errorf("expected content unchanged, got:\n%s", got)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 remaining finding, got %v", findings)
	}
	if findings[0].Line != 3 {
		t.Errorf("expected finding on line 3, got line %d", findings[0].Line)
	}
}

func TestFixSkipsFencedCodeBlock(t *testing.T) {
	input := "```\n| a | b | c |\n| --- | --- |\n```\n"
	got, findings, err := Fix(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Fix returned error: %v", err)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings inside fenced block, got %v", findings)
	}
	if string(got) != input {
		t.Errorf("expected fenced block left untouched, got:\n%s", got)
	}
}

func TestFixEmptyInput(t *testing.T) {
	got, findings, err := Fix(strings.NewReader(""))
	if err != nil {
		t.Fatalf("Fix returned error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty output, got %q", got)
	}
	if len(findings) != 0 {
		t.Fatalf("expected no findings, got %v", findings)
	}
}
