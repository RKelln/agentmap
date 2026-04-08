# testdata/

Static markdown fixtures used by the test suite. These files are read-only
from the perspective of tests — never modified in place.

**Rule:** always write generated output to a temp directory (`t.TempDir()`,
`--output /tmp/`, or `--dry-run`) when testing against files here.

---

## Files

### `authentication.md`
Realistic multi-section doc (~200 lines, 10 headings) with a pre-existing
`AGENT:NAV` block and YAML frontmatter. Used as a benchmark fixture in
`generate_test.go` and as a keyword-extraction fixture in
`keywords/keywords_test.go`.

### `code-fences.md`
Small file with headings inside fenced code blocks. Exercises the parser's
code-fence awareness — headings inside fences must not be treated as real
section boundaries.

### `design-clean.md`
Large doc (~88 headings, the project's own design document). Triggers pruning
under default config. Used in the idempotency benchmark
(`TestFile_IdempotentDesignClean`) and as a benchmark fixture.

### `error-policy.md`
Tiny doc (3 headings, below `min_lines` default of 50). Gets a purpose-only
nav block. Used to verify the short-file path in `generate`.

### `tiny.md`
Single-heading file, also below `min_lines`. Similar role to `error-policy.md`.

### `pruning-over-budget.md`
Purpose-built fixture that exercises all three branches of `PruneNavEntries`
under default config (`sub_threshold=50`, `expand_threshold=150`,
`max_nav_entries=20`):

| Section | N (lines) | Classification | Result |
|---------|-----------|----------------|--------|
| Large Section | ≥ 150 | unkillable | all 9 h3s kept |
| Medium Section | 50–149 | hintable | h3s removed; names appended as `>` hints on parent |
| Small Section | < 50 | droppable | h3s silently removed; no hints |
| Filler One–Six | ~7 each | — | h2-only; inflate entry count |

Total: 35 entries → budget 20. After pruning: 20 entries survive.

To inspect the output manually:
```bash
cp -r testdata/ /tmp/demo
./agentmap index /tmp/demo
head -30 /tmp/demo/pruning-over-budget.md
```

---

## `index-fixture/`

A self-contained subdirectory used by `index/index_test.go` to test the
`agentmap index` command — specifically the generation of `AGENTMAP.md` (the
cross-file index). It contains its own `README.md` and `docs/` tree, simulating
a small project. Tests copy it to `t.TempDir()` before each run so the
originals are never modified.

---

## Adding a new fixture

1. Create the `.md` file here with a descriptive name.
2. Add a row to this README describing its purpose and which tests use it.
3. In tests, read via the path helper (see `fixtureDir()` in
   `internal/index/index_test.go` or the `mustReadBenchmarkFixture` pattern in
   `generate_test.go`) — never `os.Chdir` into this directory.
