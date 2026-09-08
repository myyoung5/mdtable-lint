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

// delimCandidateRe matches a cell that could plausibly be someone's attempt
// at a separator cell: no letters or digits, just punctuation. It's
// deliberately looser than delimCellRe so that a typo like "==" still gets
// the row recognized as a table (and reported via lintTable's stricter
// check) instead of silently falling through as ordinary text.
var delimCandidateRe = regexp.MustCompile(`^[^\p{L}\p{N}]+$`)

// Lint reads markdown from r and returns every table problem it finds,
// in the order the problems appear in the input.
func Lint(r io.Reader) ([]Finding, error) {
	lines, err := readLines(r)
	if err != nil {
		return nil, err
	}

	var findings []Finding
	scanTables(lines, func(lines []string, i int) int {
		consumed, tableFindings := lintTable(lines, i)
		findings = append(findings, tableFindings...)
		return consumed
	})
	return findings, nil
}

// scanTables walks lines looking for pipe tables (a row with an unescaped
// pipe immediately followed by a delimiter row), skipping fenced code
// blocks, and invokes handle at each table's header line. handle returns
// how many lines the table occupies so the scan can skip past it.
func scanTables(lines []string, handle func(lines []string, headerIdx int) int) {
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
			i += handle(lines, i)
			continue
		}
		i++
	}
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

// isDelimiterRow reports whether line looks like an attempted table
// separator row, e.g. "| --- | :--- | ---: |" or a malformed one like
// "| --- | == |". It only rules out rows that clearly aren't a separator
// (empty, or containing ordinary text) — cell-by-cell validity is lintTable's
// job, so a malformed separator still gets the table recognized and reported
// instead of being mistaken for prose and skipped.
func isDelimiterRow(line string) bool {
	if !hasUnescapedPipe(line) {
		return false
	}
	cells := splitRow(line)
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		if !delimCandidateRe.MatchString(cell) {
			return false
		}
	}
	return true
}

// hasUnescapedPipe reports whether line contains a "|" that acts as a table
// cell delimiter, i.e. one that isn't backslash-escaped and isn't inside an
// inline code span.
func hasUnescapedPipe(line string) bool {
	if strings.TrimSpace(line) == "" {
		return false
	}
	return len(tablePipes([]rune(line))) > 0
}

// splitRow splits a table row into trimmed cell contents, dropping the
// optional leading and trailing pipe and honoring backslash-escaped pipes
// and pipes inside inline code spans (e.g. the pipe in `` `a\|b` `` is part
// of the code span's literal text, not a cell separator).
func splitRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")

	runes := []rune(trimmed)
	pipes := tablePipes(runes)

	var cells []string
	start := 0
	for _, p := range pipes {
		cells = append(cells, strings.TrimSpace(string(runes[start:p])))
		start = p + 1
	}
	cells = append(cells, strings.TrimSpace(string(runes[start:])))
	return cells
}

// tablePipes returns the indices, into runes, of the pipes that act as cell
// delimiters: neither backslash-escaped nor inside a run of backticks
// forming an inline code span. A code span opened by a run of n backticks is
// closed by the next run of exactly n backticks; until then its contents,
// including any pipes or backslashes, are literal.
func tablePipes(runes []rune) []int {
	var pipes []int
	escaped := false
	codeSpanTicks := 0
	for idx := 0; idx < len(runes); idx++ {
		r := runes[idx]
		if codeSpanTicks > 0 {
			if r == '`' {
				start := idx
				for idx < len(runes) && runes[idx] == '`' {
					idx++
				}
				if idx-start == codeSpanTicks {
					codeSpanTicks = 0
				}
				idx--
			}
			continue
		}
		switch {
		case escaped:
			escaped = false
		case r == '\\':
			escaped = true
		case r == '`':
			start := idx
			for idx < len(runes) && runes[idx] == '`' {
				idx++
			}
			codeSpanTicks = idx - start
			idx--
		case r == '|':
			pipes = append(pipes, idx)
		}
	}
	return pipes
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
