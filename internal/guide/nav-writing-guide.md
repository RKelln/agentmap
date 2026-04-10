# Nav Writing Guide

This is the canonical reference for writing AGENT:NAV block descriptions.

**Scope:** `purpose`, `about`, and `see` fields.
Do not hand-edit metadata (`s`, `n`, `nav[N]`, `see[N]`) or line numbers — `agentmap update` manages that.

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
You may add and remove `see` entries as appropriate to document changes.

---

## 1. Before You Write

Read the AGENT:NAV block first. This gives you purpose, section names,
and line ranges before you commit to reading further.

Never hand-edit nav line metadata while doing description work. Keep your edits to `purpose`, `about`,
and `see`; then run `agentmap update` to refresh line numbers.

Use line ranges from the nav block to read only what you need:

```
Read(offset=s, limit=n)    # read exactly the section flagged by update
```

Skim the full section content before writing its description. A description written from the heading
alone is usually wrong — the actual content often surprises. For sections with `>` hints (e.g.
`topic>sub1;sub2;sub3`), read the subsections too before writing the parent `about`. Try to include
each subsection or decide what is most useful and informative if too many subsections.

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

`purpose` is a one-line summary of the **entire file**. An agent reading only this line should know
whether the file is relevant to their task.

**Rules:** under 10 words; no commas (use semicolons); no `~` prefix after you write it.

| | Example |
|---|---|
| Bad | `~authentication OAuth2 PKCE token flow redirect` |
| Bad | `Overview of the authentication system and how it works` |
| Good | `token lifecycle; OAuth2 exchange; refresh and revocation policies` |

The bad keyword example is auto-generated noise. The bad prose example is too long and vague.
The good example tells an agent exactly what topics are covered and whether to keep reading.

**Ask yourself:** if I read only this line, would I know whether to open this file?

---

## 4. Writing `about` Fields

`about` is a one-line summary of **one section**. An agent reading only this line should know
whether to read that section.

**Rules:** under 10 words; no commas; no `~` prefix after you write it.

### The anti-restatement rule

**`about` must never restate the heading.** The heading is already visible in the `name` field.
Repeating it wastes the description slot and gives the reader nothing new.

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

**Ask yourself:** would a reader learn anything from the `about` that they couldn't already infer
from the heading alone? If no — leave it blank.

Avoid filler words: `overview`, `introduction`, `details`, `information`, `description`. Use the
actual mechanism, policy, or concept the section covers.

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

**Ask yourself:** if I read only this line, would I know whether to read this section vs. its siblings?

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

**Why it matters:** The index task list only includes files and sections that still have `~`
descriptions. Removing `~` marks that section as done. Files where every description has been
reviewed drop off the work list.

Do not remove `~` from a description you haven't actually improved — that defeats the tracking.

---

## 7. Quality Checklist

Before committing a description, check these four criteria:

**1. Decision support** — Would an agent reading only this description know whether to read the section?
If not, it's too vague.

**2. Disambiguation** — Is it specific enough to distinguish from sibling sections?
`token management` fails if there are three token-related sections; `silent rotation and expiry` does not.

**3. Precision** — Avoid generic words: `overview`; `introduction`; `details`; `information`.
Use the actual mechanism; policy; or concept the section covers.

**4. Format compliance** — Under 10 words; no commas; `~` removed.

A description that passes all four takes seconds to write and saves many minutes of agent navigation.
