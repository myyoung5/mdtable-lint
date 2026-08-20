package main

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Finding is a single problem reported at a specific line of the input.
type Finding struct {
	Line    int
	Message string
}

var delimCellRe = regexp.MustCompile(`^:?-+:?$`)

// Lint reads markdown from r and returns every table problem it finds,
// in the order the problems appear in the input.
func Lint(r io.Reader) ([]Finding, error) {
	lines, err := readLines(r)
	if err != nil {
		return nil, err
	}

	var findings []Finding
	i := 0
	var inFence bool
	var fenceChar rune
	var fenceLen int
	for i < len(lines) {
		if inFence {
			if char, count, rest, ok := fenceMarker(lines[i]); ok && char == fenceChar && rest == "" && count >= fenceLen {
				inFence = false
			}
			i++
			continue
		}
		if char, count, _, ok := fenceMarker(lines[i]); ok {
			inFence = true
			fenceChar = char
			fenceLen = count
			i++
			continue
		}
		if i+1 < len(lines) && hasUnescapedPipe(lines[i]) && isDelimiterRow(lines[i+1]) {
			consumed, tableFindings := lintTable(lines, i)
			findings = append(findings, tableFindings...)
			i += consumed
			continue
		}
		i++
	}
	return findings, nil
}

// fenceMarker reports whether line opens or closes a fenced code block, per
// CommonMark: up to 3 leading spaces followed by a run of 3+ backticks or
// tildes. rest is whatever follows that run (an info string for an opening
// fence, or trailing whitespace for a closing one); the caller compares it
// against the character and length that opened the fence.
func fenceMarker(line string) (char rune, count int, rest string, ok bool) {
	stripped := strings.TrimLeft(line, " ")
	if len(line)-len(stripped) > 3 {
		return 0, 0, "", false
	}
	if stripped == "" {
		return 0, 0, "", false
	}
	char = rune(stripped[0])
	if char != '`' && char != '~' {
		return 0, 0, "", false
	}
	runeStripped := []rune(stripped)
	for count < len(runeStripped) && runeStripped[count] == char {
		count++
	}
	if count < 3 {
		return 0, 0, "", false
	}
	rest = strings.TrimSpace(string(runeStripped[count:]))
	return char, count, rest, true
}

// lintTable checks one table starting at header index i (0-based) and
// returns how many lines it consumed and the findings within it.
func lintTable(lines []string, i int) (int, []Finding) {
	var findings []Finding

	headerCells := splitRow(lines[i])
	delimLine := i + 1
	delimCells := splitRow(lines[delimLine])

	if len(delimCells) != len(headerCells) {
		findings = append(findings, Finding{
			Line: delimLine + 1,
			Message: fmt.Sprintf(
				"header/separator column count mismatch: header has %d columns, separator has %d",
				len(headerCells), len(delimCells)),
		})
	}
	for _, cell := range delimCells {
		if !delimCellRe.MatchString(cell) {
			findings = append(findings, Finding{
				Line: delimLine + 1,
				Message: fmt.Sprintf(
					"invalid separator cell %q: must contain only dashes with optional leading/trailing colon",
					cell),
			})
		}
	}

	j := i + 2
	for j < len(lines) && hasUnescapedPipe(lines[j]) {
		rowCells := splitRow(lines[j])
		if len(rowCells) != len(headerCells) {
			findings = append(findings, Finding{
				Line: j + 1,
				Message: fmt.Sprintf(
					"row has %d columns, expected %d (from header)",
					len(rowCells), len(headerCells)),
			})
		}
		j++
	}

	return j - i, findings
}

// isDelimiterRow reports whether line is a valid GFM table separator row,
// e.g. "| --- | :--- | ---: |".
func isDelimiterRow(line string) bool {
	if !hasUnescapedPipe(line) {
		return false
	}
	cells := splitRow(line)
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		if !delimCellRe.MatchString(cell) {
			return false
		}
	}
	return true
}

// hasUnescapedPipe reports whether line contains a "|" that isn't preceded
// by a backslash.
func hasUnescapedPipe(line string) bool {
	if strings.TrimSpace(line) == "" {
		return false
	}
	escaped := false
	for _, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == '|' {
			return true
		}
	}
	return false
}

// splitRow splits a table row into trimmed cell contents, dropping the
// optional leading and trailing pipe and honoring backslash-escaped pipes.
func splitRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")

	var cells []string
	var current strings.Builder
	escaped := false
	for _, r := range trimmed {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			current.WriteRune(r)
			escaped = true
		case r == '|':
			cells = append(cells, strings.TrimSpace(current.String()))
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	cells = append(cells, strings.TrimSpace(current.String()))
	return cells
}

func readLines(r io.Reader) ([]string, error) {
	var lines []string
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}
