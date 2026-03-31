# agentmap Implementation Tasks

Vertical slices -- each phase delivers a working CLI command with tests.

## Phase 1: `generate` -- Headings to Nav Block (v0.1)

**Goal:** `agentmap generate ./docs` produces nav blocks with empty descriptions.

**What's already done:**
- `parser.ParseHeadings` -- extracts h1-h3, code-fence aware (14 tests)
- `navblock.ParseNavBlock` / `navblock.RenderNavBlock` -- round-trip (7 tests)
- Test fixtures in `testdata/`

**What's needed:**
1. **Section computation** -- given headings, compute `s,e` ranges for each section
   - Section ends at line before next heading at same or higher level, or EOF
   - Empty sections: `s == e`
2. **Nav block writer** -- insert/replace nav block in file content
   - After YAML frontmatter (if present), before first heading
   - Replace existing block in place
   - One blank line between nav block and first heading
3. **File discovery** -- find all `.md` files under a path (recursive)
   - Skip `.git/` directory
   - v0.1: no `.gitignore` parsing needed, just skip `node_modules`, `.git`
4. **`generate` command** -- wire it all together
   - For each file: parse headings → compute sections → render nav block → write file
   - `--dry-run` flag: print without writing
   - Output: `Generated: path (N sections)` / `Skipped: path (under N lines; purpose-only)`
   - Purpose-only blocks for files under `min_lines` (default 50)

**Tests:** section computation, nav block insertion, file discovery, generate command integration with testdata fixtures.

---

## Phase 2: `generate` -- Keyword Descriptions

**Goal:** `agentmap generate` produces nav blocks with keyword-based descriptions.

**What's needed:**
1. **Keyword extraction** -- TF-IDF per section
   - Tokenize, lowercase, remove stopwords, remove short tokens
   - Term frequency within section
   - Optional IDF against other sections in same document
   - Top 4-6 terms joined by semicolons
2. **Purpose generation** -- keywords from first paragraph or whole file
3. **Subsection hints** -- `>` suffix with abbreviated h3 names
   - Three-tier threshold: no hints (<50 lines), hints (50-150), full h3 entries (>150)
4. **Flags** -- `--min-lines`, `--sub-threshold`, `--expand-threshold`

**Tests:** keyword extraction (stopwords, TF, IDF, edge cases), purpose generation, threshold logic, integration with generate.

---

## Phase 3: `update` -- Line Number Refresh

**Goal:** `agentmap update ./docs` refreshes line numbers, preserves descriptions.

**What's needed:**
1. **Nav matching** -- match existing nav entries to current headings
   - Strip `#` prefixes from `name`, match by text
   - Duplicate headings: match in order of appearance
   - New headings: add with empty `about`
   - Deleted headings: remove entry
2. **Line number update** -- recompute `s,e` for matched headings
3. **Git diff integration** -- `git diff HEAD -- <file>` → changed line ranges
   - Map changed ranges to sections
   - Mark sections as `content-changed` or `new`
4. **Report output** -- the primary interface for agents
   - `OK`, `shifted`, `content-changed`, `new`, `deleted`
   - Current description text for changed sections
5. **Flags** -- `--quiet`, `--dry-run`
6. **Exit codes** -- 0 success, 1 error

**Tests:** nav matching (rename, new, delete, duplicate), line update, git diff mapping, report formatting, integration with modified testdata.

---

## Phase 4: `check` -- Validation

**Goal:** `agentmap check ./docs` validates nav blocks are in sync.

**What's needed:**
1. **Validation** -- parse nav block, reparse headings, compare
   - Mismatched line numbers
   - Headings in nav but not in document
   - Headings in document but not in nav
2. **Report** -- `FAIL: path` with details per mismatch
3. **Exit codes** -- 0 all sync, 1 one or more failures
4. **Never modifies files**

**Tests:** validation logic, all mismatch types, exit codes, integration.

---

## Phase 5: Configuration + File Discovery

**Goal:** `agentmap.yml` config + proper `.gitignore`-aware file discovery.

**What's needed:**
1. **Config loader** -- read `agentmap.yml`, apply defaults
   - `min_lines`, `sub_threshold`, `expand_threshold`, `max_depth`, `exclude`
   - Config overrides via CLI flags take precedence
2. **File discovery** -- `git ls-files` when in a git repo
   - Respects `.gitignore` automatically
   - Filter against `exclude` list from config
   - Fallback to `filepath.Walk` when not in a git repo
3. **Edge cases** -- nav block corruption handling, empty sections, special characters in headings, very large files (cap at ~20 entries)

**Tests:** config loading (missing file, partial config, overrides), file discovery (git repo, non-git repo, exclude patterns), edge cases.

---

## Phase 6: Polish + Pre-Commit Hook

**Goal:** Production-ready tool with pre-commit hook support.

**What's needed:**
1. **Pre-commit hook template** -- `.git/hooks/pre-commit` script
2. **Performance** -- `update` and `check` under 100ms for 50 files
3. **Error handling** -- clear error messages, `fmt.Errorf("context: %w", err)`
4. **`--llm` flag stub** -- flag exists but returns error if not configured (LLM integration is out of scope for v0.1)

**Tests:** hook template, performance benchmark, error paths.
