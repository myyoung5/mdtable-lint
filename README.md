# mdtable-lint

Markdown pipe tables fall apart in small ways that are easy to miss in a
diff: a row picks up one extra `|` and its cells shift, someone deletes a
column from the header but forgets the separator row, or the separator row
gets typed as `|---|---|` with three columns while the header only has two.
Most renderers don't error on this, they just render something slightly
wrong, and it sits there until someone notices by eye.

`mdtlint` scans markdown for pipe tables and reports, with line numbers,
wherever a row's column count doesn't match the header or the separator row
is malformed.

## Usage

Lint one or more files:

```
$ mdtlint README.md docs/api.md
docs/api.md:14: row has 4 columns, expected 3 (from header)
```

Or pipe content in from stdin, using `-` conventions aren't needed — no
arguments means read stdin:

```
$ cat notes.md | mdtlint
stdin:7: invalid separator cell "==": must contain only dashes with optional leading/trailing colon
```

Exit status is `0` if no findings, `1` if any findings were reported, `2` on
an error reading input (missing file, etc).

## What it checks today

- separator row column count matches the header column count
- each separator cell is dashes with an optional leading/trailing colon
  (`---`, `:---`, `---:`, `:---:`)
- every data row has the same number of columns as the header
- fenced code blocks (\`\`\` or `~~~`) are skipped, so a markdown example
  containing pipes inside a fence isn't mistaken for a real table
- pipes inside inline code spans (`` `a|b` ``) are treated as literal text,
  not cell separators, so a cell like `` `a\|b` `` isn't split wrong

## What it doesn't do yet

- alignment consistency (e.g. warning if some rows clearly ignore the
  declared alignment) isn't checked

## Building

Standard library only, no dependencies:

```
go build -o mdtlint .
```

## License

MIT, see LICENSE.
