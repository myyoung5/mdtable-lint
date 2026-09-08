package main

import (
	"fmt"
	"io"
	"strings"
)

// Fix reads markdown from r and returns the content with auto-correctable
// separator row problems fixed in place, plus any findings it couldn't fix.
//
// Only the separator row is rewritten: its column count is padded or
// truncated to match the header, and any cell that isn't valid dash syntax
// is replaced with "---". Data row column mismatches are left as findings
// rather than fixed, since there's no safe way to guess whether a row is
// missing a cell or has an extra one.
func Fix(r io.Reader) ([]byte, []Finding, error) {
	lines, err := readLines(r)
	if err != nil {
		return nil, nil, err
	}
	if len(lines) == 0 {
		return nil, nil, nil
	}

	var findings []Finding
	scanTables(lines, func(lines []string, i int) int {
		consumed, tableFindings := fixTable(lines, i)
		findings = append(findings, tableFindings...)
		return consumed
	})

	return []byte(strings.Join(lines, "\n") + "\n"), findings, nil
}

// fixTable corrects the separator row of the table at header index i,
// mutating lines in place, and returns how many lines the table occupies
// and any findings for problems it left unfixed.
func fixTable(lines []string, i int) (int, []Finding) {
	var findings []Finding

	headerCells := splitRow(lines[i])
	delimLine := i + 1
	delimCells := splitRow(lines[delimLine])

	changed := false
	for len(delimCells) < len(headerCells) {
		delimCells = append(delimCells, "---")
		changed = true
	}
	if len(delimCells) > len(headerCells) {
		delimCells = delimCells[:len(headerCells)]
		changed = true
	}
	for idx, cell := range delimCells {
		if !delimCellRe.MatchString(cell) {
			delimCells[idx] = "---"
			changed = true
		}
	}
	if changed {
		lines[delimLine] = "| " + strings.Join(delimCells, " | ") + " |"
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
