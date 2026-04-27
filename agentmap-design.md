# agentmap: Navigation Maps for AI Agents

<!-- AGENT:NAV
purpose:design spec for agentmap CLI tool; nav block format; commands; agent workflow
nav[23]{s,n,name,about}:
37,12,##1. Problem,agents waste tokens navigating markdown
49,10,##2. Solution,compact nav block with line ranges at file top
59,158,##3. Format Specification,block structure; fields; depth; constraints; examples
61,13,###3.1 Nav Block Structure,HTML comment wrapper; field layout
74,28,###3.2 Field Definitions,purpose; nav header; entries; see block
102,21,###3.3 Heading Depth Convention,# count mirrors markdown; absolute depth
123,35,###3.4 Complete Example,full nav block with three-tier demo
158,7,###3.5 Placement Rules,after frontmatter; first 40 lines; 40-line budget rationale
164,7,###3.6 Constraints,no commas; no escaping; descriptions optional
171,12,###3.7 Purpose-Only Block,minimal block for small files
183,34,###3.8 Subsection Hints,> suffix; three-tier threshold logic
217,114,##4. CLI Commands,command behavior and flags>generate;update;check
331,64,##5. Description Authoring,tiered: empty; keywords; agent-written; LLM>empty;keywords;agent-written;LLM;preservation
395,46,##6. Git Integration,change detection via diff; pre-commit hook setup
441,82,##7. Agent Workflow,AGENTS.md instructions; reading and editing flows>instructions;workflow;skill
523,52,##8. Parser Specification,heading parser; nav block parser/writer rules>heading-parser;block-parser;block-writer
575,28,##9. Keyword Extraction (Tier 1),offline TF-IDF; stopwords; purpose generation
603,60,##10. Configuration,agentmap.yml; file discovery; ignore rules; defaults>config-file;discovery;defaults
663,33,##11. Edge Cases,duplicates; special chars; empty sections; corruption
696,45,##12. Implementation Notes,Python project structure; testing; performance
741,42,##13. Future Work (Out of Scope for v0.1),project index; cross-file see; staleness
783,40,##Appendix A: Token Budget Analysis,token cost; savings per navigation; break-even math
823,55,##Appendix B: Design Decisions and Rationale,HTML comments; # depth; no escaping; check in hook>HTML-comments;#-depth;no-escaping;check-in-hook;git-diff;subsection-hints
-->

## Design Document v0.1

### Status: Design complete, ready for implementation

---

## 1. Problem

AI coding agents waste significant tokens navigating documentation and markdown files. When an agent needs information from a 500-line markdown file, it typically reads the entire file (~2000 tokens) even if it only needs a 20-line section (~80 tokens).

Agents have tools to read file slices by line offset, but no efficient way to know *which* lines to read. The current options are:

- **Read the whole file.** Works, but wasteful. Compounds across a session touching many files.
- **Grep for a heading.** Costs a full tool-call round-trip (~150-270 tokens of overhead plus latency) to get a line number the agent could have been given directly.
- **Guess.** Unreliable.

Code files have LSPs, treesitter, go-to-definition, and symbol search. Markdown has none of these. This tool fills that gap.

## 2. Solution

**agentmap** is a CLI tool that generates and maintains a compact navigation block at the top of markdown files. The block gives agents:

1. A one-line file purpose (should I read this file at all?)
2. A section index with line ranges (which section do I need, and exactly which lines?)
3. Cross-references to related files (is the answer actually somewhere else?)

Agents are instructed (via AGENTS.md) to read the first 40 lines of any markdown file before reading the rest. This gives them the nav block, which collapses multi-step navigation into a single precise `Read(offset=s, limit=n)` call.

## 3. Format Specification

### 3.1 Nav Block Structure

```markdown
<!-- AGENT:NAV
purpose:one-line file description
nav[N]{s,n,name,about}:
s,n,#Heading,description
s,n,##Subheading,description
see[N]{path,why}:
relative/path.md,reason to read it
-->
```

### 3.2 Field Definitions

**Header line:** `<!-- AGENT:NAV`
- Exact string. Used by the parser to locate the block.
- Must be the first line of the block.

**purpose:** `purpose:one-line file description`
- No space after the colon.
- Single line. Describes what the file is and when an agent should read further.
- Written by `generate` (via keyword extraction or LLM). Preserved by `update`.

**lines:** `lines:N` *(optional)*
- Total line count of the file (`wc -l` equivalent — number of newlines). Matches what an editor displays.
- Written by `generate`; refreshed by `update` whenever it writes the block.
- Lets an agent decide how much to read before opening the file. Omitted for files with no nav block.

**nav:** `nav[N]{s,n,name,about}:`
- `N` = total number of entries (including subheadings).
- `s` = start line number (1-indexed, inclusive).
- `n` = number of lines in the section (length). The count of lines from `s` to the last line of content before the next heading or EOF.
- `name` = heading text prefixed with `#` characters matching the heading depth in the source. `#` = h1, `##` = h2, `###` = h3.
- `about` = short description of the section's content. No commas; use semicolons if needed. May be empty for new/unwritten descriptions. May include a `>` suffix with subsection hints (see section 3.8).
- One entry per line. No leading whitespace on entries.

**see:** `see[N]{path,why}:`
- `N` = number of entries.
- `path` = relative path from repo root to the related file.
- `why` = reason an agent working in this file might need that file. No commas.
- Optional block. Omit entirely if no related files.

**Footer line:** `-->`
- Closes the HTML comment.

### 3.3 Heading Depth Convention

The `#` count in the `name` field mirrors the heading depth in the source markdown:

```markdown
nav[6]{s,n,name,about}:
5,41,#Authentication,token lifecycle management
8,13,##Token Exchange,OAuth2 code-for-token flow
10,5,###PKCE,proof key for code exchange
15,6,###Implicit,legacy implicit grant flow
21,15,##Token Refresh,silent rotation and expiry detection
36,10,##Token Revocation,logout and forced invalidation
```

This provides **absolute depth** — an agent landing on any entry knows its exact position in the hierarchy without scanning upward. It directly mirrors the markdown heading syntax agents have seen extensively in training data.

**Rules:**
- Every entry has at least one `#`. There are no unmarked entries.
- Depth is absolute, not relative. `##` always means h2 regardless of surrounding entries.
- Maximum tracked depth: configurable via `max_depth` (default h6). This is a heuristic to keep nav entries within budget — in a shallow document all tracked headings get full entries regardless of depth. Headings deeper than `max_depth` are not included; the agent reads the parent section to find them.

### 3.4 Complete Example

Source file `docs/authentication.md` (350 lines):

```markdown
---
title: Authentication
---

<!-- AGENT:NAV
purpose:token lifecycle; OAuth2 exchange; refresh and revocation policies
nav[8]{s,n,name,about}:
12,339,#Authentication,token lifecycle management
14,37,##Overview,protocol selection; supported grant types
51,80,##Token Exchange,OAuth2 code-for-token flow>PKCE;implicit;device-code
131,200,##Token Lifecycle,rotation; expiry; revocation
135,66,###Refresh,silent rotation and sliding-window expiry
201,80,###Revocation,logout; forced invalidation; webhook notify
281,50,###Introspection,token validation endpoint; caching policy
331,20,##Migration Guide,upgrading from v1 tokens
see[2]{path,why}:
src/config.py,default timeout and token TTL values
docs/error-policy.md,error handling for auth failures
-->

# Authentication

Token lifecycle management for the platform...
```

This example shows all three tiers from section 3.8:
- `##Overview` (36 lines) and `##Migration Guide` (19 lines) — under `sub_threshold`; no subsection info even if they had children.
- `##Token Exchange` (79 lines) — between `sub_threshold` and `expand_threshold`; `>` hints list its three children without giving them their own entries.
- `##Token Lifecycle` (199 lines) — over `expand_threshold`; its children (`###Refresh`, `###Revocation`, `###Introspection`) appear as full nav entries with their own `s,n` ranges.

### 3.5 Placement Rules

1. After frontmatter (if present), before the first heading.
2. Always wrapped in an HTML comment (`<!-- AGENT:NAV ... -->`).
3. The nav block should appear immediately in the file (after frontmatter) and be limited in length. This allows AGENTS.md to instruct agents to "read the first 40 lines" and reliably get the nav block. The 40-line budget assumes frontmatter up to ~10 lines and a nav block up to ~20 lines (the default configurable limit), leaving headroom for both.

### 3.6 Constraints

- **No commas in nav block field values** — this applies to `about`, `why`, `purpose`, and subsection hints (the `>` suffix). These fields are parsed as CSV; a comma would be treated as a field delimiter and corrupt the entry. Use **semicolons** as the preferred separator — they read naturally in both lists ("rotation; expiry; revocation") and descriptions ("silent rotation and expiry detection"). The generator should warn (not error) if a nav field contains a comma. This constraint is scoped to nav block fields only; regular markdown content, headings, body text, and code examples are unaffected.
- **No quoting or escaping.** Simplicity over flexibility. The comma restriction eliminates the need for any escape mechanism.
- **Descriptions are optional.** A new or ungenerated entry may have an empty `about` field: `12,45,#Authentication,`
- **`see` block is optional.** Omit it entirely (not `see[0]:`) if there are no related files.

### 3.7 Purpose-Only Block

Files under the minimum line threshold (default: 50 lines) get a minimal block:

```markdown
<!-- AGENT:NAV
purpose:helper utilities for date formatting
-->
```

No `nav` or `see` sections. The `purpose` line is always useful regardless of file size — it lets an agent decide whether to read the file at all.

### 3.8 Subsection Hints

The `about` field may include a `>` suffix listing abbreviated names of subsections that have no
nav entry of their own. This trades horizontal space (longer lines) for vertical space (fewer nav
entries) and gives agents routing signals without expanding every subsection into its own entry.

**Format:** `description>sub1;sub2;sub3`

```
51,171,##3. Format Specification,block structure; fields; depth>structure;field-defs;depth-convention;placement;constraints;purpose-only
172,284,##4. CLI Commands,command behavior and flags>generate;update;check
528,555,##9. Keyword Extraction,offline TF-IDF; stopwords
```

**When to use `>` hints vs. full entries:**

The generator starts by including all tracked headings as full nav entries, then prunes back to
`max_nav_entries` (default 20) if needed. Pruning works depth-first: the deepest entries are
removed first. When a deep entry is removed, it either becomes a `>` hint on its parent or
disappears entirely, depending on parent section size:

| Pruned entry's parent section size | Outcome |
|---|---|
| Under `sub_threshold` (default 50 lines) | Entry dropped; no hint — parent is cheap to read in full |
| `sub_threshold` to `expand_threshold` (default 150 lines) | Entry collapsed to `>` hint on parent |
| Over `expand_threshold` | Entry kept as full entry — section too large to scan without offsets |

A document with 1 h1, 2 h2s, and 15 h3s has 18 total entries — under the default cap of 20 — so
all get full entries regardless of section size. Thresholds only apply when the budget is exceeded.

**Authoring:**

`agentmap generate` initially populates hints with verbatim heading text as a scaffold (marked `~`
on the full `about` value). Agents reviewing `~` entries should rewrite the whole value — both the
description and the hints — choosing 1-2 words per hint that best encapsulate that subsection.
Hints may overlap with heading words but should reflect agent judgment about the most useful label.

The `~` prefix covers the entire `about` value including hints. Removing `~` signals that both
the description and the hints have been reviewed.

**Rules:**
- `>` hints are appended directly to the `about` text with no space before `>`.
- Subsection names are separated by semicolons.
- No commas in subsection names (same constraint as `about`).
- Subsection names should be 1-2 words — abbreviate to the most useful label.
- One hint per subsection that has no nav entry of its own — count matches the unrepresented subsections.
- `agentmap update` preserves the full `about` value (description and hints) unchanged.
- If a section has no unrepresented subsections, no `>` suffix appears.

## 4. CLI Commands

### 4.1 `agentmap generate [path]`

**Purpose:** Initial creation of nav blocks. Parses markdown headings, generates descriptions, writes the full nav block.

**Behavior:**
1. Find all markdown files under `path` (recursive). Respect `.gitignore` (see section 10.3).
2. For each file:
   a. Parse all headings (h1-h3) with their line numbers.
   b. Compute end-line for each section (line before next heading at same or higher level, or EOF).
   c. Apply subheading threshold: only include h2+ if parent section exceeds configured line count (default: 50).
   d. If file is under minimum line threshold (default: 50 lines), emit purpose-only block.
   e. Generate `about` descriptions using Tier 1 keyword extraction (default) or LLM (with `--llm` flag). All keyword-generated descriptions are prefixed with `~` to signal auto-generated content.
   f. Generate `purpose` from first paragraph of file or keyword extraction. Keyword-generated purpose is also prefixed with `~`.
   g. Write nav block at correct position (after frontmatter, before first heading).
3. If a nav block already exists, **skip the file** (safe default). Use `--force` to overwrite.

**Flags:**
- `--force` — Overwrite existing nav blocks. Without this flag, files that already have a nav block are skipped.
- `--llm` — Use an LLM to generate descriptions instead of keyword extraction. Requires LLM configuration (see section 7).
- `--min-lines N` — Override minimum file size threshold (default: 50).
- `--sub-threshold N` — Override subheading inclusion threshold (default: 50 lines). Sections under this size get no subsection info.
- `--drop-subtitles` — Exclude structural headings (lone subtitles with no same-depth siblings) from the nav block entirely, regardless of budget. Without this flag, structural headings are still pruned first when budget is exceeded but included when budget allows.
- `--expand-threshold N` — Override full-expansion threshold (default: 150 lines). These thresholds are pruning heuristics that only apply when the `max_nav_entries` budget is exceeded: sections with parent N >= expand-threshold are kept as full entries (unkillable); sections with parent N in [sub-threshold, expand-threshold) become `>` hints on the parent entry; sections with parent N < sub-threshold are dropped silently (see section 3.8 and §11.4).
- `--dry-run` — Print what would be generated without writing files.

**Output:**
```
Generated: docs/authentication.md (6 sections)
Generated: docs/error-policy.md (3 sections)
Skipped: docs/changelog.md (under 50 lines; purpose-only)
Skipped: README.md (under 50 lines; purpose-only)
Skipped: docs/existing.md (already has nav block; use --force to overwrite)
```

### 4.2 `agentmap update [path]`

**Purpose:** Fast line-number refresh. Preserves all descriptions. This is the pre-commit and agent-facing command.

**Behavior:**
1. Find all markdown files under `path`.
2. For each file:
   - If the file has **no nav block**: delegate to `generate` (same as running `agentmap generate` on that file). This makes `update <dir>` work correctly on a mixed directory of new and already-indexed files.
   - If the file has an existing nav block:
   a. Parse the existing nav block (extract entries with their `name` and `about` fields).
   b. Reparse the markdown to get current headings and line numbers.
   c. Match existing nav entries to current headings by heading text (`name` field, ignoring `#` prefixes).
   d. **Matched headings:** Update `s,n` values. Preserve `about` description unchanged.
   e. **New headings** (in document but not in nav): Add entry with empty `about`.
   f. **Deleted headings** (in nav but not in document): Remove entry.
   g. **Renamed headings:** Appear as a deletion + a new entry. This is correct — a renamed section likely needs a new description.
   h. Recompute `[N]` counts.
   i. Write updated nav block.
3. Run `git diff` (staged + unstaged) on the file to identify which sections have content changes.
4. Output a report (to stdout) detailing what changed.

**Report format:**
```
Updated: docs/authentication.md
  OK: #Authentication (12-65)
  shifted: ##Token Exchange (14-35 -> 16-37)
  content-changed: ##Token Refresh (lines 36-50; 14 lines modified)
    current description: "silent rotation and expiry detection"
  new: ##Token Revocation (51-65; no description)
  deleted: ##Legacy Auth (removed from document)

Updated: docs/error-policy.md
  OK: #Error Handling (8-95)
  OK: ##Retry Policy (10-45)
  OK: ##Circuit Breakers (46-95)

No changes: docs/glossary.md
```

**The report is the primary interface for agents.** It tells the agent:
- Which sections need description updates (`content-changed` with modified content)
- Which sections need new descriptions (`new` with no description)
- The current description text (so the agent can decide if it still fits)
- Line ranges for changed sections (so the agent can read exactly the changed content)

**Flags:**
- `--quiet` — Suppress the report. For use in pre-commit hooks where only the file modifications matter.
- `--dry-run` — Print report without writing files.

**Exit codes:**
- 0: All nav blocks updated successfully.
- 1: Error (file not found, parse failure, etc.).

### 4.3 `agentmap check [path]`

**Purpose:** Validation. Verifies nav blocks are in sync with document headings. Used in CI and pre-commit hooks.

**Behavior:**
1. Find all markdown files under `path` that have an existing `<!-- AGENT:NAV` block.
2. For each file:
   a. Parse the nav block.
   b. Reparse the markdown headings and line numbers.
   c. Compare: do nav entries match current headings (by text) with correct line numbers?
3. Report mismatches.

**Output on failure:**
```
FAIL: docs/authentication.md
  ##Token Exchange: nav says 14-35, actual 16-37
  ##Token Revocation: in document but not in nav

FAIL: docs/error-policy.md
  ##Deprecated Section: in nav but not in document

2 files failed validation.
```

**Exit codes:**
- 0: All nav blocks are in sync.
- 1: One or more nav blocks are out of sync.

**This is the pre-commit hook command.** It never modifies files. It only validates and fails, forcing the agent or human to run `update` explicitly.

## 5. Description Authoring

Descriptions are the most valuable part of the nav block (they tell agents *what* a section contains) and the hardest to automate. agentmap uses a tiered approach:

### 5.1 Tier 0: Empty

New headings added by `update` get an empty description:

```
51,65,##Token Revocation,
```

The nav block still provides line numbers and hierarchy, which is the core mechanical value. A nav block with empty descriptions is still better than no nav block.

### 5.2 Tier 1: Keyword Extraction (default for `generate`)

The generator reads section content, extracts distinctive terms. No LLM required, pure text processing.

**Algorithm:**
1. Tokenize section content into words (split on whitespace and punctuation).
2. Lowercase, remove stopwords (common English words + markdown syntax).
3. Compute term frequency within the section.
4. Optionally compute TF-IDF against the full document (terms distinctive to *this* section vs. others).
5. Take top 4-6 terms, join with semicolons.

**The `~` prefix convention:**

All keyword-generated descriptions (both `purpose` and `about` fields) are prefixed with `~`. This signals: auto-generated; never reviewed by an agent or human.

```
purpose:~token OAuth2 authentication flow
nav[3]{s,n,name,about}:
12,65,#Authentication,~OAuth2 PKCE redirect token lifecycle
36,50,##Token Refresh,~silent rotation expiry detection
```

When an agent rewrites a description it removes the `~`:

```
purpose:token lifecycle; OAuth2 exchange flow
nav[3]{s,n,name,about}:
12,65,#Authentication,token refresh; expiry; rotation
36,50,##Token Refresh,silent rotation and expiry detection
```

The `~` prefix is used by downstream commands (index, check) to detect unreviewed descriptions. The `navblock.IsAutoGenerated(s)` helper checks for the prefix; `navblock.TrimAutoGenerated(s)` removes it.

**Example output:**
```
12,65,#Authentication,~OAuth2;PKCE;redirect;token;lifecycle
36,50,##Token Refresh,~silent;rotation;expiry;detection
```

Crude but useful — the agent sees topical keywords and can decide relevance without reading the section.

### 5.3 Tier 2: Agent-Written

The highest quality descriptions are written by agents or humans who understand the content. The workflow:

1. Agent runs `agentmap update` and sees the report.
2. Report flags sections as `content-changed` or `new`.
3. Agent reads the flagged sections using the line ranges from the report.
4. Agent writes/updates descriptions directly in the nav block.
5. Agent commits.

This is where most description quality comes from in practice. The `update` report is designed to give the agent exactly the information it needs to write good descriptions with minimal token spend.

### 5.4 Tier 3: LLM-Generated (optional, via `generate --llm`)

For initial bulk generation or periodic refresh. The generator sends each section's content to an LLM with a prompt like:

```
Write a one-line description (under 10 words; no commas) of this
markdown section for use in a file navigation index. Focus on what
an agent would need to know to decide whether to read this section.
```

This requires LLM configuration (API key, model selection). It's optional and never runs implicitly.

### 5.5 Description Preservation Rules

- `generate` writes descriptions (overwrites existing).
- `update` never writes descriptions. Never. It only modifies `s,n` values and `[N]` counts.
- `check` never writes anything.
- Descriptions are anchored to heading text. If a heading is renamed, the old description is lost and the new heading gets an empty description.
- The `~` prefix survives `update` unchanged since update preserves descriptions as-is.

## 6. Git Integration

### 6.1 Content Change Detection

`agentmap update` uses git to determine which sections have modified content:

1. Run `git diff HEAD -- <file>` to get changed line ranges.
2. Map each changed line range to the section(s) it falls within (using the newly computed `s,n` values).
3. Mark those sections as `content-changed` in the report.

If the file is untracked (new file), all sections are marked as `new`.

If git is not available (not a git repo), skip change detection and only report structural changes (shifted, new, deleted headings).

### 6.2 Pre-Commit Hook

The recommended pre-commit hook runs `check`, not `update`:

```yaml
# .pre-commit-config.yaml
repos:
  - repo: local
    hooks:
      - id: agentmap-check
        name: Validate AGENT:NAV blocks
        entry: agentmap check
        language: python
        types: [markdown]
```

Or as a simple git hook:

```bash
#!/bin/sh
# .git/hooks/pre-commit
agentmap check . || {
  echo "AGENT:NAV blocks are out of sync. Run: agentmap update ."
  exit 1
}
```

The hook is a gatekeeper. It validates and fails, never modifies. This ensures:
- The agent always runs `update` explicitly and sees the report.
- Humans are prompted to run `update` and can review changes.
- No silent file modifications during commit.

## 7. Agent Workflow

### 7.1 AGENTS.md Instructions

Add this to the project's AGENTS.md:

```markdown
## Reading Markdown Files

Use AGENTMAP.md first for file search/discovery.
Flow: read AGENTMAP.md -> identify file -> read AGENT:NAV -> jump to section.

AGENT:NAV appears immediately after frontmatter so you can 
read a files first 50 lines then use AGENT:NAV to target reads.

- If purpose does not match your task stop reading.
- Use s;n ranges: Read(offset=s; limit=n).
- `>` is a hint for subsections that are not listed directly in the nav.

## Before Committing Markdown Changes

1. Run: `agentmap update <changed files>` — refreshes heading line numbers and flags content-changed or new sections.
2. Edit the nav block directly: update `purpose`; `about`; and `see` descriptions for any flagged sections.
    - Do not edit s;n counts; nav[N]; or see[N] by hand.
    - Keep nav block format stable; add a `see` block after nav entries if needed.
    - Run `agentmap guide` for full instructions on writing nav descriptions.
3. Run: `agentmap update <changed files>` again — syncs AGENTMAP.md index with the updated purposes.
4. Commit.
```

### 7.2 Detailed Agent Workflow

**Reading a file:**
```
1. Agent needs information about authentication.
2. Agent reads first 40 lines of docs/authentication.md.
3. Sees nav block:
   purpose:token lifecycle; OAuth2 exchange; refresh and revocation
   nav[6]{s,n,name,about}:
   12,54,#Authentication,token lifecycle management
   14,22,##Token Exchange,OAuth2 code-for-token flow
   ...
   36,15,##Token Refresh,silent rotation and expiry detection
4. Agent needs token refresh info.
5. Agent calls Read(offset=36, limit=14).
6. Done. Read 14 lines instead of 200+.
```

**After editing a file:**
```
1. Agent modifies content in docs/authentication.md.
2. Agent runs: agentmap update docs/authentication.md
3. Sees output:
   Updated: docs/authentication.md
     shifted: ##Token Exchange (14-35 -> 16-37)
     content-changed: ##Token Refresh (38-52; 8 lines modified)
       current description: "silent rotation and expiry detection"
     new: ##Token Revocation (53-67; no description)
4. Agent reads lines 38-52 (changed section) and lines 53-67 (new section).
5. Agent edits nav block:
   - Updates ##Token Refresh description if needed.
   - Writes description for ##Token Revocation.
6. Agent commits. Pre-commit hook runs agentmap check. Passes.
```

### 7.3 Agent Skill (Optional Enhancement)

An agent skill can wrap this workflow:

```
Skill: agentmap-maintain
Trigger: agent has edited markdown files and is about to commit

Steps:
1. Run agentmap update on all modified .md files.
2. Parse the report output.
3. For each content-changed section:
   a. Read the section content using the reported line range.
   b. Evaluate if the current description still fits.
   c. If not, write an updated description (under 10 words, no commas).
4. For each new section:
   a. Read the section content.
   b. Write a description.
5. Commit.
```

## 8. Parser Specification

### 8.1 Markdown Heading Parser

The parser extracts headings from markdown files to build the nav block.

**Rules:**
- A heading is a line starting with 1-6 `#` characters followed by a space: `^(#{1,6}) (.+)$`
- Only track h1-h3 (1-3 `#` characters). Ignore h4+.
- Headings inside fenced code blocks (``` ``` ```) are not headings. Track code fence state.
- Headings inside HTML comments (`<!-- -->`) are not headings.
- The heading text is everything after `# ` (strip leading/trailing whitespace).

**Section boundaries:**
- A section starts at the heading line.
- A section ends at the line before the next heading at the same or higher (fewer `#`) level, or at EOF.
- Subsections are contained within their parent section's line range.

**Example:**
```
Line 12: # Authentication        -> section 12-65
Line 14: ## Token Exchange       -> section 14-35
Line 36: ## Token Refresh        -> section 36-50
Line 51: ## Token Revocation     -> section 51-65
Line 66: # Error Handling        -> section 66-95
```

### 8.2 Nav Block Parser

To parse an existing nav block:

1. Find `<!-- AGENT:NAV` in the file (must be on its own line or start of line).
2. Read lines until `-->`.
3. Parse `purpose:` line (everything after `purpose:`).
4. Parse `nav[N]{fields}:` header to get field names and expected count.
5. Parse subsequent lines as CSV with the declared fields until a blank line, another header (`see[N]...`), or `-->`.
6. Parse `see[N]{fields}:` header and its entries the same way.

**Matching existing entries to current headings:**
- Strip `#` prefixes from `name` field to get plain heading text.
- Match by exact text comparison (case-sensitive).
- If a heading appears multiple times in the document (duplicate headings), match in order of appearance.

### 8.3 Nav Block Writer

When writing/updating the nav block:

1. If YAML frontmatter exists (file starts with `---`), find the closing `---` and insert after it.
2. If an existing `<!-- AGENT:NAV ... -->` block exists, replace it in place.
3. If no existing block and no frontmatter, insert at line 1.
4. Ensure exactly one blank line between the nav block and the first heading.

## 9. Keyword Extraction (Tier 1)

Simple, offline, no-LLM keyword extraction for `about` descriptions.

### 9.1 Algorithm

```
function extractKeywords(sectionText, documentText, maxKeywords=5):
  1. Tokenize sectionText: split on whitespace and punctuation.
  2. Lowercase all tokens.
  3. Remove stopwords (English stopwords + markdown syntax tokens).
  4. Remove tokens shorter than 3 characters.
  5. Count term frequency in section.
  6. (Optional) Compute IDF against other sections in the same document.
  7. Score = TF * IDF (or just TF if single-section).
  8. Return top maxKeywords terms joined by semicolons.
```

### 9.2 Stopwords

Include standard English stopwords plus:
- Markdown syntax: `the`, `this`, `that`, `which`, `from`, `with`, `for`, `and`, `but`, `not`, `are`, `was`, `were`, `been`, `will`, `can`, `may`, `should`, `must`, `shall`, etc.
- Common markdown/doc words: `section`, `following`, `example`, `note`, `see`, `also`, `above`, `below`, `describes`, `provides`, `contains`, `overview`

### 9.3 Purpose Generation

For the `purpose:` line, run keyword extraction across the entire file (not per-section) and take the top 6-8 terms. Alternatively, extract from the first non-heading paragraph.

## 10. Configuration

### 10.1 Config File

`agentmap.yml` in the repo root (optional):

```yaml
# Minimum file size to generate full nav block (lines)
# Files under this threshold get purpose-only block
min_lines: 50

# Minimum section size to include subsection info (lines)
# Sections under this get no subsection hints or entries
sub_threshold: 50

# Section size at which full subsection entries replace inline > hints (lines)
# Between sub_threshold and expand_threshold: > hints only
# Above expand_threshold: full subsection entries with own line ranges
# Note: sub_threshold and expand_threshold are heuristics — the real constraint
# is max_nav_entries. If all tracked headings fit in the budget, all get entries.
expand_threshold: 150

# Maximum heading depth to track (heuristic to stay near max_nav_entries budget)
max_depth: 6

# Files/directories to exclude (in addition to .gitignore)
exclude:
  - "dist/**"
  - "CHANGELOG.md"

# LLM configuration (for generate --llm)
llm:
  model: "gpt-4o-mini"
  # API key should be in environment variable, not config
```

### 10.2 File Discovery and Ignore Rules

All commands that accept a `[path]` argument discover markdown files recursively. Files are excluded in this order:

**v0.1: `.gitignore` only.**
- All `.gitignore` rules are respected (root and nested `.gitignore` files).
- The `exclude` list in `agentmap.yml` provides additional glob patterns for files that are tracked by git but should not get nav blocks (e.g., generated changelogs, vendored docs, files that are templated or machine-written).
- Implementation: use `git ls-files` for file discovery when inside a git repo. This inherently respects `.gitignore`. Then filter against the `exclude` list from config.

**v0.2: dedicated `.agentmap-ignore` file.**
- A `.agentmap-ignore` file in the repo root, using the same glob syntax as `.gitignore`.
- For projects that need fine-grained control beyond the config `exclude` list (e.g., ignoring specific subdirectories of docs, per-directory overrides).
- Evaluated after `.gitignore` — a file must pass both filters.

### 10.3 Defaults

All configuration is optional. Sensible defaults:

| Setting | Default | Notes |
|---|---|---|
| `min_lines` | 50 | Purpose-only below this |
| `max_nav_entries` | 20 | Soft cap on nav entries; the primary budget constraint |
| `sub_threshold` | 50 | No subsection hints/entries below this (heuristic) |
| `expand_threshold` | 150 | Full subsection entries above this; `>` hints between (heuristic) |
| `max_depth` | 6 | Heading depth limit (heuristic to stay near budget) |
| `exclude` | `[]` | Additional excludes beyond .gitignore |

## 11. Edge Cases

### 11.1 Duplicate Headings

If a document has multiple headings with the same text (e.g., two `## Examples` sections), match by order of appearance. The nav block lists them in document order, and the `s,n` line ranges disambiguate.

### 11.2 Headings with Special Characters

Heading text is stored as-is in the `name` field, with these exceptions:
- Commas are stripped (they would break CSV parsing).
- Leading/trailing whitespace is stripped.
- Markdown attribute syntax (`{: .class-name}`) is stripped.
- Bracketed content (`[link]`, `[note]`) is stripped.
- Parenthetical content (`(deprecated)`, `(draft)`) is stripped.

### 11.3 Empty Sections and Heading Clusters

**Consecutive same-depth heading drop (always applied in `generate`):**

A run of consecutive headings at the same depth with no lines between them (not even blank lines) represents a metadata block or heading cluster. Only the first heading in the run is kept; the rest are dropped before nav generation.

Examples:
```
# Title
# Key: value    ← dropped (same depth, no lines between)
# Key: value    ← dropped
```
```
# H1
## Sub1
## Sub2         ← dropped (same depth as Sub1, no lines between)
```
```
# H1
## Sub1
               ← blank line breaks the run
## Sub2         ← kept
## Sub3         ← dropped (same depth as Sub2, no lines between)
```

This rule operates in `generate` (not the parser). The parser reports all headings faithfully; `generate` filters before building the nav entry list.

**Structural (lone subtitle) tagging:**

A heading is tagged `structural` if it is the only heading at its depth within its parent section — i.e., it has no siblings at the same depth under the same parent. Lone subtitles are low-information: they cannot be navigated to independently since there is no second sibling to distinguish them from.

Structural headings are treated as lowest-priority during budget pruning (pruned before any depth-based pruning). With `--drop-subtitles` they are excluded from the nav block entirely regardless of budget.

### 11.4 Very Large Files

Files with many headings would produce a nav block that itself costs significant tokens. A soft cap (`max_nav_entries`, default 20) limits nav entries. `max_depth`, `sub_threshold`, and `expand_threshold` are all heuristics in service of this budget.

**Algorithm:**

1. **Include all** — start with all headings up to `max_depth` as full nav entries, after applying the consecutive same-depth drop (§11.3).
2. **Check budget** — if total entries ≤ `max_nav_entries`, done.
3. **Prune structural first** — remove entries tagged `structural` (lone subtitles). Apply `sub_threshold`/`expand_threshold` rules to each removed entry as in step 4.
4. **Prune depth-first** — remove the deepest remaining entries first. For each removed entry:
   - If its parent section is under `sub_threshold`: drop entirely (cheap to read in full).
   - If its parent section is between `sub_threshold` and `expand_threshold`: collapse to a `>` hint on the parent entry.
   - If its parent section is over `expand_threshold`: keep as a full entry (too large to scan without offsets).
5. **Repeat** until within budget or no more prunable entries remain.

Shallow entries (those with no parent in the nav block) are never pruned. If shallow entries alone exceed the cap, the overrun is accepted.

### 11.5 Files Without Headings

Markdown files with no headings (e.g., prose documents) get a purpose-only block. No `nav` section.

### 11.6 Nav Block Corruption

If the parser cannot parse an existing nav block (malformed syntax), `update` should:
1. Warn to stderr.
2. Treat it as if no nav block exists.
3. Offer to regenerate with `generate`.

Do not silently overwrite a corrupted block — the corruption may contain manually edited content worth preserving.

## 12. Implementation Notes

### 12.1 Language

Python. No heavy dependencies. Standard library for markdown parsing and git integration. The tool should be installable via pip and runnable as a CLI.

### 12.2 Project Structure

```
agentmap/
  agentmap/
    __init__.py
    cli.py          # CLI entry point (argparse or click)
    parser.py       # Markdown heading parser
    navblock.py     # Nav block parser/writer
    keywords.py     # Tier 1 keyword extraction
    git.py          # Git diff integration
    config.py       # Configuration loading
  tests/
    test_parser.py
    test_navblock.py
    test_keywords.py
    test_git.py
    test_integration.py
    fixtures/       # Sample markdown files for testing
  agentmap.yml      # Example config
  pyproject.toml
  README.md
```

### 12.3 Testing Strategy

- **Unit tests** for heading parser (various markdown edge cases: code fences, HTML comments, duplicate headings, special characters).
- **Unit tests** for nav block parser/writer (round-trip: parse then write should be identity).
- **Unit tests** for keyword extraction.
- **Integration tests** with sample markdown files: generate, modify file, update, check.
- **Git integration tests** using a temporary git repo.

### 12.4 Performance Considerations

- `update` and `check` should be fast enough for pre-commit hooks. Target: <100ms for a repo with 50 markdown files.
- No LLM calls in `update` or `check`. Ever.
- File I/O: read each file once, write only if changed.
- Git: one `git diff` call per file (or batch with `git diff -- file1 file2 ...`).

## 13. Future Work (Out of Scope for v0.1)

### 13.1 Project-Level Index

A root-level nav block (in AGENTS.md or a dedicated file) that maps the entire repository. This solves the "cold start" problem where an agent doesn't know which files to open first.

Format TBD. Possibly:
```
<!-- AGENT:NAV
purpose:project-level file index
files[N]{path,about}:
docs/authentication.md,token lifecycle and OAuth2
docs/error-policy.md,retry and circuit-breaker rules
docs/glossary.md,term definitions
-->
```

### 13.2 Cross-File `see` Auto-Population

Automatically populate `see` entries by analyzing:
- Markdown links between files
- Frontmatter `depends_on` fields
- Git co-change frequency (files frequently committed together)

### 13.3 Description Staleness Detection

Track when descriptions were last updated relative to content changes. Flag descriptions that haven't been updated after significant content changes (not just any change).

### 13.4 Editor Integration

VS Code / Neovim extensions that render the nav block as a clickable sidebar, auto-update on save, etc.

### 13.5 Dedicated Ignore File

A `.agentmap-ignore` file using `.gitignore` syntax for fine-grained control over which markdown files get nav blocks. Per-directory overrides. See section 10.2 for rationale.

### 13.6 Non-Markdown Support

Possible future extension to other structured text formats (RST, AsciiDoc, MDX). Out of scope for v0.1 — markdown only.

---

## Appendix A: Token Budget Analysis

### Nav block cost

A typical nav block for a file with 6 sections and 2 see entries:

```
<!-- AGENT:NAV
purpose:token lifecycle; OAuth2 exchange; refresh and revocation
   nav[6]{s,n,name,about}:
   12,54,#Authentication,token lifecycle management
   14,22,##Token Exchange,OAuth2 code-for-token flow
   ...
   36,15,##Token Refresh,silent rotation and expiry detection
51,65,##Token Revocation,logout and forced invalidation
see[2]{path,why}:
src/config.py,default timeout and token TTL values
docs/error-policy.md,error handling for auth failures
-->
```

Approximate token count: ~120-150 tokens (varies by tokenizer).

### Savings per navigation

| Approach | Tokens | Tool calls |
|---|---|---|
| Read entire 500-line file | ~2000 | 1 |
| Grep then read section | ~300-400 | 2 |
| Nav block then read section | ~200-250 | 1* |

*The first read (nav block) can be part of reading the first 20 lines, which agents often do by default.

### Break-even

The nav block costs ~130 tokens to read. It saves ~1700 tokens vs. full-file reads, ~100-200 tokens vs. grep. It pays for itself on the first use.

Over a session touching 10 files, the savings compound: ~17,000 tokens saved vs. full reads, ~1,000-2,000 vs. grep, minus ~1,300 tokens for reading 10 nav blocks. Net savings: ~14,000-16,000 tokens per session.

## Appendix B: Design Decisions and Rationale

### Why HTML comments, not YAML frontmatter?

Many markdown files already use YAML frontmatter for other purposes (static site generators, documentation tools). Putting the nav block in a separate HTML comment avoids conflicts and keeps concerns separated. The nav block is for agent navigation, not document metadata.

### Why `#` for heading depth instead of indentation or relative markers?

- Absolute depth: `###` means h3 everywhere, no context needed.
- Direct mapping to source: the `#` count matches the markdown heading exactly.
- Training data: models have seen `#`/`##`/`###` trillions of times in markdown.
- Alternatives considered: `.`/`..` (relative path semantics are backwards — `.` means current not child), `>`/`>>` (blockquote semantics), indentation (whitespace tokens), explicit depth column (less readable).

### Why no quoting or escaping?

The nav block is small and constrained. Adding quoting rules (even simple ones) means:
- The parser is more complex.
- Agents need to understand the quoting convention.
- Edge cases multiply.

Banning commas in nav field values and using semicolons instead is a simpler constraint that eliminates the problem entirely. Semicolons work well as description separators — they're visually distinct from the CSV delimiters, read naturally in prose ("rotation; expiry; revocation"), and are unambiguous. If a description naturally contains a comma, rewrite it with a semicolon. The generator can warn about this.

### Why `check` in the hook, not `update`?

If the hook ran `update`, it would silently modify files during commit. This means:
- The agent never sees the update report (it runs in git's process).
- Humans don't review the changes.
- The commit includes modifications the committer didn't explicitly make.

Making the hook a gatekeeper (`check` fails, human/agent runs `update` explicitly) keeps the process transparent and gives agents the report they need to update descriptions.

### Why git diff for change detection, not stored hashes?

Git already tracks every change. Storing content hashes in the nav block adds:
- Extra fields the agent doesn't use.
- State that can get out of sync.
- Complexity in the parser and writer.

`git diff HEAD -- file.md` gives exact changed line ranges, which map directly to sections. It's simpler, more reliable, and leverages infrastructure that's already present.

### Why `>` subsection hints instead of always expanding?

Full subsection entries are precise but expensive: each adds a line to the nav block (~8-12 tokens). For a file with many sections, expanding every subsection could double the nav block size — burning the token budget the tool exists to save.

The three-tier threshold balances precision against cost:
- Small sections (under 50 lines) need no subsection info — reading the whole section is cheap.
- Medium sections (50-150 lines) get `>` hints — a few extra tokens per line that tell the agent *which* subsection exists without adding full entries. The agent reads the parent section and scans for the hinted heading.
- Large sections (over 150 lines) get full entries with own `s,n` ranges — the section is too large to scan comfortably, so precise offsets pay for themselves.

Hints apply to any subsection depth (h3 under h2, h4 under h3, etc.) as long as that subsection has no nav entry of its own. The tool initially populates hints with verbatim heading text as a scaffold; agents reviewing `~` entries rewrite both the description and hints to produce 1-2 word labels chosen for navigational usefulness, not just heading verbatim.

This was validated by A/B testing with two subagents. The version with `>` hints reduced every test task to a single targeted read. The biggest win: finding a specific rule buried in a long section went from 4 reads (wrong section first, backtracked) to 1 read. The hints cost ~30 extra tokens across the whole nav block and saved ~200-400 tokens per avoided misroute.

Alternatives considered:
- **Always expand subsections:** Too many nav lines for large files. A 20-section file with 3 subsections each produces 80 entries.
- **Never show subsections:** Agents misroute into wrong sections when descriptions are ambiguous. The hints disambiguate.
- **Line numbers in hints** (`>sub1:42-48`): Both test agents requested this, but it adds too many tokens per hint line and once you're inside the right section, scanning for a heading is cheap.
