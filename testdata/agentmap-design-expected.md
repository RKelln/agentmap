<!-- AGENT:NAV
purpose:nav;line;block;file;lines;entries;markdown;threshold
nav[27]{s,n,name,about}:
32,338,#agentmap: Navigation Maps for AI Agents,nav;line;block;file;lines
35,40,##Design Document v0.1,design;complete;document;implementation;ready
37,40,##1. Problem,file;line;tokens;files;markdown
41,52,##2. Solution,agents;block;file;read;files
53,62,##3. Format Specification,line;nav;threshold;lines;entries
63,255,###3.1 Nav Block Structure,block;markdown;nav;structure
65,69,###3.2 Field Definitions,closes;comment;definitions;field;html
70,73,###3.3 Heading Depth Convention,depth;token;heading;markdown;absolute
74,94,###3.4 Complete Example,lines;threshold;authentication;children;token
95,114,###3.5 Placement Rules,line;threshold;nav;block;file
115,230,###13.2 Cross-File `see` Auto-Population,files;analyzing;auto;automatically;change
231,237,###13.3 Description Staleness Detection,changes;content;descriptions;updated;change
238,241,###13.4 Editor Integration,auto;block;clickable;code;editor
242,245,###13.5 Dedicated Ignore File,file;ignore;agentmap;blocks;control
246,249,###13.6 Non-Markdown Support,markdown;asciidoc;extension;formats;future
250,255,###3.1 Nav Block Structure,block;markdown;nav;structure
256,282,###3.2 Field Definitions,closes;comment;definitions;field;html
258,266,###3.3 Heading Depth Convention,depth;token;heading;markdown;absolute
267,276,###3.4 Complete Example,lines;threshold;authentication;children;token
277,282,###3.5 Placement Rules,line;threshold;nav;block;file
283,338,###13.2 Cross-File `see` Auto-Population,files;analyzing;auto;automatically;change
285,288,###13.3 Description Staleness Detection,changes;content;descriptions;updated;change
289,295,###13.4 Editor Integration,auto;block;clickable;code;editor
296,304,###13.5 Dedicated Ignore File,file;ignore;agentmap;blocks;control
305,313,###13.6 Non-Markdown Support,markdown;asciidoc;extension;formats;future
314,322,##Appendix A: Token Budget Analysis,tokens;nav;000;block;read
323,338,##Appendix B: Design Decisions and Rationale,nav;block;agent;hints;sections>Why HTML comments; not YAML frontmatter?;Why `#` for heading depth instead of indentation or relative markers?;Why no quoting or escaping?;Why `check` in the hook; not `update`?;Why git diff for change detection; not stored hashes?;Why `>` subsection hints instead of always expanding h3s?
-->

# agentmap: Navigation Maps for AI Agents


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

Agents are instructed (via AGENTS.md) to read the first ~20 lines of any markdown file before reading the rest. This gives them the nav block, which collapses multi-step navigation into a single precise `Read(offset=s, limit=n)` call.

## 3. Format Specification

### 3.1 Nav Block Structure

```markdown
```

### 3.2 Field Definitions

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
- Maximum tracked depth: `###` (h3). Deeper headings (h4+) are not included in the nav block — the agent reads the parent h3 section to find them.

### 3.4 Complete Example

Source file `docs/authentication.md` (350 lines):

```markdown
---
title: Authentication
---


# Authentication

Token lifecycle management for the platform...
```

This example shows all three tiers from section 3.8:
- `##Overview` (36 lines) and `##Migration Guide` (19 lines) — under `sub_threshold`; no subsection info even if they had h3 children.
- `##Token Exchange` (79 lines) — between `sub_threshold` and `expand_threshold`; `>` hints list its three h3 children without giving them their own entries.
- `##Token Lifecycle` (199 lines) — over `expand_threshold`; its h3 children (`###Refresh`, `###Revocation`, `###Introspection`) appear as full nav entries with their own `s,n` ranges.

### 3.5 Placement Rules

1. After YAML frontmatter (if present), before the first heading.
```

No `nav` or `see` sections. The `purpose` line is always useful regardless of file size — it lets an agent decide whether to read the file at all.

### 3.8 Subsection Hints

The `about` field may include a `>` suffix listing abbreviated subsection names. This trades horizontal space (longer lines) for vertical space (fewer nav entries) and gives agents routing signals without expanding every subsection into its own entry.

**Format:** `description>sub1;sub2;sub3`

```
51,171,##3. Format Specification,block structure; fields; depth>structure;field-defs;depth-convention;placement;constraints;purpose-only
172,284,##4. CLI Commands,command behavior and flags>generate;update;check
528,555,##9. Keyword Extraction,offline TF-IDF; stopwords
```

Subsection names are 1-2 word abbreviations of the h3 heading text. They don't need to match the heading exactly — they need to be recognizable enough for the agent to decide "this section contains what I need."

**When to use `>` hints vs. full h3 entries:**

The generator uses a three-tier threshold based on parent section line count:

| Parent section size | Strategy | Rationale |
|---|---|---|
| Under `sub_threshold` (default 50 lines) | No subsection info | Section is cheap to read in full |
| `sub_threshold` to `expand_threshold` (default 150 lines) | `>` hints only | Agent reads parent section; hints help scan |
| Over `expand_threshold` | Full h3 entries with own `s,n` ranges | Section too large to scan; agent needs precise offsets |

This keeps the nav block compact for most files. A file with ten 40-line sections produces 10 nav entries with no hints. A file with three 80-line sections produces 3 entries with `>` hints. A file with one 200-line section produces 1 parent entry plus its h3 children as full entries.

**Rules:**
- `>` hints are appended directly to the `about` text with no space before `>`.
- Subsection names are separated by semicolons.
- No commas in subsection names (same constraint as `about`).
- Subsection names should be 1-2 words — abbreviate freely.
- `>` hints are generated by the tool and preserved by `update` (same as `about` descriptions).
- If a section has no subsections (no h3+ headings), no `>` suffix appears.

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
   e. Generate `about` descriptions using Tier 1 keyword extraction (default) or LLM (with `--llm` flag).
   f. Generate `purpose` from first paragraph of file or keyword extraction.
   g. Write nav block at correct position (after frontmatter, before first heading).
3. If a nav block already exists, **overwrite it entirely**. This is a full regeneration command.

**Flags:**
- `--llm` — Use an LLM to generate descriptions instead of keyword extraction. Requires LLM configuration (see section 7).
- `--min-lines N` — Override minimum file size threshold (default: 50).
- `--sub-threshold N` — Override subheading inclusion threshold (default: 50 lines). Sections under this size get no subsection info.
- `--expand-threshold N` — Override full-expansion threshold (default: 150 lines). Sections over this size get full h3 entries with own `s,n` ranges. Sections between `sub-threshold` and `expand-threshold` get `>` hints instead (see section 3.8).
- `--dry-run` — Print what would be generated without writing files.

**Output:**
```
Generated: docs/authentication.md (6 sections)
Generated: docs/error-policy.md (3 sections)
Skipped: docs/changelog.md (under 50 lines; purpose-only)
Skipped: README.md (under 50 lines; purpose-only)
```

### 4.2 `agentmap update [path]`

**Purpose:** Fast line-number refresh. Preserves all descriptions. This is the pre-commit and agent-facing command.

**Behavior:**
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

Banning commas in descriptions and using semicolons instead is a simpler constraint that eliminates the problem entirely. If a description naturally contains a comma, rewrite it with a semicolon. The generator can warn about this.

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

### Why `>` subsection hints instead of always expanding h3s?

Full h3 entries are precise but expensive: each adds a line to the nav block (~8-12 tokens). For a file with many sections, expanding every h3 could double the nav block size — burning the token budget the tool exists to save.

The three-tier threshold balances precision against cost:
- Small sections (under 50 lines) need no subsection info — reading the whole section is cheap.
- Medium sections (50-150 lines) get `>` hints — a few extra tokens per line that tell the agent *which* subsection exists without adding full entries. The agent reads the parent section and scans for the hinted heading.
- Large sections (over 150 lines) get full h3 entries — the section is too large to scan comfortably, so precise offsets pay for themselves.

This was validated by A/B testing with two subagents. The version with `>` hints reduced every test task to a single targeted read. The biggest win: finding a specific rule buried in a long section went from 4 reads (wrong section first, backtracked) to 1 read. The hints cost ~30 extra tokens across the whole nav block and saved ~200-400 tokens per avoided misroute.

Alternatives considered:
- **Always expand h3s:** Too many nav lines for large files. A 20-section file with 3 h3s each produces 80 entries.
- **Never show subsections:** Agents misroute into wrong sections when descriptions are ambiguous. The hints disambiguate.
- **Line numbers in hints** (`>sub1:42-48`): Both test agents requested this, but it adds too many tokens per hint line and once you're inside the right section, scanning for a heading is cheap.
