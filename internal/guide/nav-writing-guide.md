# Nav Writing Guide

This is the canonical reference for writing AGENT:NAV block descriptions.

**Scope:** `purpose`, `about`, and `see` fields.
Do not hand-edit metadata (`s`, `n`, `nav[N]`, `see[N]`) or line numbers — `agentmap update` manages that.

---

## The Mental Model

An agent navigating a codebase opens a file based on a guess — a filename, an index entry, a
search result. The nav block is read immediately to validate that guess:

- **`purpose`** — serves two roles: (1) scanned in the AGENTMAP index to find the right file
  among many, and (2) read after opening to confirm the guess was correct. Must be specific enough
  to rule in *and* rule out in both contexts.
- **`about`** — routes within the file once confirmed. The agent reads only the sections that
  answer its question, skipping the rest.
- **`see`** — exit ramp when the guess was wrong. Points directly to the right file without a
  new search.

All three decisions happen under token pressure with no additional context. Every word costs
tokens. A vague or redundant description forces the agent to read sections it shouldn't have
to — defeating the nav block entirely.

**The goal of every description:** enable a confirm/route/exit decision with no further reading.

Two properties matter above all others:

1. **Disambiguation** — distinguishes this entry from its siblings. An agent can only navigate if
   descriptions differ meaningfully. `token management` fails when three sections exist; `sliding-window expiry` does not.
2. **Decision support** — contains enough information to act on. If the agent still needs to read
   the section to know whether it's relevant, the description failed.

Everything else — word count, filler avoidance, format — is in service of these two.

---

## Quick Reference

```
<!-- AGENT:NAV
purpose:one-line file summary; no commas
nav[N]{s,n,name,about}:
s,n,#Heading,one-line section summary; don't repeat heading; no commas>subsection
see[N]{path,why}:
relative/path.md,linked file purpose relative to this file
-->
```

**Order is strict:** `purpose` first, then `nav[...]` and its entries, then optional `see[...]` and its entries, then closing `-->`.

---

## 1. Before You Write

Read the AGENT:NAV block first. This gives you purpose, section names,
and line ranges before you commit to reading further.

**Read the entire file before writing any descriptions.** You need the full picture to write
descriptions that disambiguate correctly — a section's `about` should be written knowing what
its siblings and parent sections contain, not in isolation.

Never hand-edit nav line metadata while doing description work. Keep your edits to `purpose`, `about`,
and `see`; then run `agentmap update` to refresh line numbers.

Use line ranges from the nav block to read only what you need when re-reading for detail:

```
Read(offset=s, limit=n)    # read exactly the section flagged by update
```

---

## 2. Starting from Scratch

Don't craft a nav block by hand and then run `agentmap generate` — it will overwrite your work.

The correct workflow for a new file:
1. Write the file content (no nav block yet).
2. Run `agentmap generate <file>` — writes a skeleton with `~`-prefixed descriptions.
3. Rewrite each `~` description with human-quality text.
4. Run `agentmap update <file>` to confirm line numbers are current.
5. Commit.

---

<!-- rules-start -->
## 3. Writing `purpose` Lines

`purpose` is a one-line summary of the **entire file**. It appears in the AGENTMAP index (where
an agent scans many files to find the right one) and at the top of the file itself (where it
confirms or rejects the guess that brought the agent here).

**Rules:** under 10 words; no commas (use semicolons); no `~` prefix after you write it.

**Write for the sibling question:** a file doesn't exist alone — it sits next to others on the
same topic. `purpose` must distinguish *this* file from its neighbours in both contexts. Prefer
concrete nouns and mechanisms over category labels.

| | Example |
|---|---|
| Bad — auto noise | `~authentication OAuth2 PKCE token flow redirect` |
| Bad — too vague | `Overview of the authentication system and how it works` |
| Bad — category label | `authentication system` |
| Good | `token lifecycle; OAuth2 exchange; refresh and revocation policies` |

**Ask yourself:** if I read only this line, would I know whether I'm in the right file *vs.* a neighbour on the same topic?

---

## 4. Writing `about` Fields

`about` is a one-line summary of **one section**. An agent already reading this file uses it to
jump directly to the right section without reading each one.

**Rules:** under 10 words; no commas; no `~` prefix after you write it.

**Prefer noun phrases over sentences.** `sliding-window expiry and silent rotation` is faster to
scan and more precise than `how token expiry and rotation work`. Concrete terms beat category
labels: `PKCE code exchange` beats `authorization flow`.

### The anti-restatement rule

**`about` must never restate the heading.** The heading is already visible in the `name` field.
Repeating it wastes the slot and gives the reader nothing to act on.

**If you cannot add new information, leave `about` empty** — a trailing comma with nothing after it
(`43,9,#Heading,`) is valid and preferable to noise. Self-explanatory headings like `##Summary` or
`##Open Questions` often need no `about`.

| | Example (heading: `##Token Refresh`) |
|---|---|
| Bad — restates heading | `token refresh` |
| Bad — restates with filler | `overview of token refresh` |
| Bad — still has `~` | `~silent rotation expiry detection` |
| Good — adds new information | `silent rotation and sliding-window expiry` |
| Good — nothing to add | _(empty)_ |

**Ask yourself:** would an agent skip the right section based on this description alone?
If it could plausibly apply to a sibling section — rewrite it.

### Nested sections

When a section has a parent in the nav block, the parent heading and its `about` are already
visible context. A child `about` should only say what distinguishes it from its siblings — not
repeat concepts the parent already covers.

```
## Authentication,token lifecycle; OAuth2 exchange
### Token Refresh,silent rotation; sliding-window expiry   ← good: specific to refresh
### Token Refresh,token refresh and OAuth2 mechanism       ← bad: repeats parent context
### Token Revocation,immediate invalidation; blocklist     ← good: specific to revocation
```

The reader already knows they're inside `##Authentication`. The child only needs to differentiate
itself from `###Token Revocation` and `###Token Introspection` — not re-explain the auth context.

### Sections with `>` hints

Some sections are large enough to need subsection routing but not large enough to give each
subsection its own nav entry. For those, `agentmap generate` appends a `>` suffix to the `about`
field — one hint per subsection that would otherwise have no nav representation. This keeps the
nav block compact while still letting an agent identify which subsection to jump to.

**Format:** `description>sub1;sub2;sub3` — the `>` is appended directly to the description text
with **no space** before it.

```
# The full field value (everything after the comma):
token lifecycle; OAuth2 exchange>token-exchange;token-refresh;revocation
│                              ││                                       │
└─── about description ────────┘└─── subsection hints ──────────────────┘
```

`agentmap generate` produces hints as verbatim subsection heading text — a scaffold, not a final
answer. When you review a `~` entry that has hints, rewrite the whole value: description and hints
together. Choose 1-2 words per hint that best encapsulate that subsection. They may be the most
important words from the heading, but use judgment — the goal is the shortest label an agent could
scan to decide which subsection to jump to.

```
# Before (auto-generated scaffold):
51,80,##Token Exchange,~OAuth2 PKCE authorization code token exchange>PKCE Authorization Code;Client Credentials;Device Authorization

# After (agent-written) — full value rewritten, ~ removed:
51,80,##Token Exchange,OAuth2 code-for-token flow>PKCE;client-credentials;device-flow
```

**Rules for `>` hints:**
- One hint per subsection that has no nav entry of its own — don't add or drop entries, just rewrite the text.
- 1-2 words per hint; no commas (use hyphens for compound terms).
- The description before `>` must still pass the anti-restatement rule on its own.
- `agentmap update` preserves the full `about` value (description and hints) unchanged.

---

## 5. Adding `see` Entries

`see` lists files that are closely related — files an agent working here would likely need next.

**When to add:**
- The file you're writing links to another file in markdown.
- Another file defines types; configs; or constants used here.
- The two files must be read together to understand a feature.

**When not to add:** don't list every adjacent file. Only add entries that save a real search.

**Placement rule:** add `see` entries after the nav entries inside the same AGENT:NAV block.
Do not insert `see` above `purpose` or above `nav[...]`.

**Format:** `relative/path/from/repo/root.md,linked file purpose relative to this file`

```
see[2]{path,why}:
src/config.py,default timeout and token TTL values
docs/error-policy.md,error handling for auth failures
```

`why` should describe the linked file's purpose in this context. Keep it short, specific,
and free of filler words. `why` must not contain commas; use semicolons if needed.

---

## 6. The `~` Prefix

`~` at the start of a `purpose` or `about` value means: **auto-generated; never reviewed**.

```
purpose:~token OAuth2 authentication flow       # auto-generated
purpose:token lifecycle; OAuth2 exchange flow   # agent-reviewed
```

When you rewrite a description, remove the `~`. This signals that a human or agent has reviewed it.
`agentmap update` preserves `~` unchanged — only a deliberate rewrite removes it.

**Why it matters:** The index task list only includes files that still have `~` descriptions.
Removing `~` marks the section done. Files where every description has been reviewed drop off the
work list.

Do not remove `~` from a description you haven't actually improved — that defeats the tracking.

---

## 7. Quality Checklist

Before committing a description, check:

**1. Disambiguation** — Does it distinguish this entry from its siblings? If a sibling description
could be swapped in without loss — rewrite it.

**2. Decision support** — Could an agent identify the right section from this description
alone, without reading it? If not — it's too vague.

**3. No restatement** — Does it add information the heading doesn't already give? If not — leave it blank.

**4. Concrete terms** — Does it name a mechanism; policy; or concept rather than a category?
`sliding-window expiry` over `expiry details`; `PKCE code exchange` over `authorization flow`.

**5. Format** — Under 10 words; no commas; `~` removed.
