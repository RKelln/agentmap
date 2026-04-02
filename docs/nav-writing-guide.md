<!-- AGENT:NAV
purpose:practical guide for writing AGENT:NAV block descriptions; quality criteria; examples
nav[8]{s,n,name,about}:
14,156,#Nav Writing Guide,canonical how-to reference for nav block descriptions
23,14,##Quick Reference,nav block format at a glance
37,17,##1. Before You Write,read nav block first; use line ranges to skim sections
54,20,##2. Writing `purpose` Lines,one-line file summary; word limit; no commas; good vs bad examples
74,36,##3. Writing `about` Fields,one-line section summaries; > hints handling; good vs bad examples
110,23,##4. Adding `see` Entries,when and how to add cross-file links; format
133,20,##5. The `~` Prefix,auto-generated marker; when to remove it; index behavior
153,17,##6. Quality Checklist,four criteria for decision-quality descriptions
-->

# Nav Writing Guide

This is the canonical reference for writing AGENT:NAV block descriptions. It covers the practical
"how to write" workflow. For the format specification and parser rules, see `agentmap-design.md`.

**Scope:** `purpose`, `about`, and `see` fields. Not line numbers — those are managed by `agentmap update`.

---

## Quick Reference

```
<!-- AGENT:NAV
purpose:one-line file summary; no commas
nav[N]{s,n,name,about}:
s,n,#Heading,one-line section summary; no commas
see[N]{path,why}:
relative/path.md,why an agent here would need that file
-->
```

---

## 1. Before You Write

Read the AGENT:NAV block first (first 40 lines of any file). This gives you purpose, section names,
and line ranges before you commit to reading further.

Use line ranges from the nav block to read only what you need:

```
Read(offset=s, limit=n)    # read exactly the section flagged by update
```

Skim the full section content before writing its description. A description written from the heading
alone is usually wrong — the actual content often surprises. For sections with `>` hints (e.g.
`topic>sub1;sub2;sub3`), read the subsections too before writing the parent `about`.

---

## 2. Writing `purpose` Lines

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

## 3. Writing `about` Fields

`about` is a one-line summary of **one section**. An agent reading only this line should know
whether to read that section.

**Rules:** under 10 words; no commas; no `~` prefix after you write it.

| | Example |
|---|---|
| Bad | `~silent rotation expiry detection` |
| Bad | `Overview of token refresh` |
| Good | `silent rotation and sliding-window expiry` |

The bad keyword example still has `~` — it is auto-generated and unreviewed. The bad prose example
uses `overview` — a filler word that adds no information. The good example is specific: it names
the mechanisms.

**For sections with `>` hints:** The hints list child subsections (e.g. `>PKCE;implicit;device-code`).
Read those subsections, then write an `about` that covers the parent as a whole — don't just repeat
the hint names as the description.

```
# Before (auto-generated with hints):
51,80,##Token Exchange,~OAuth2 PKCE code token>PKCE;implicit;device-code

# After (agent-written):
51,80,##Token Exchange,OAuth2 code-for-token flow>PKCE;implicit;device-code
```

The `~` prefixes the entire `about` value (the text before `>`). The `>` hints survive unchanged;
you only rewrite the description before the `>`.

**Ask yourself:** if I read only this line, would I know whether to read this section vs. its siblings?

---

## 4. Adding `see` Entries

`see` lists files that are closely related — files an agent working here would likely need next.

**When to add:**
- The file you're writing links to another file in markdown.
- Another file defines types; configs; or constants used here.
- The two files must be read together to understand a feature.

**When not to add:** don't list every adjacent file. Only add entries that save a real search.

**Format:** `relative/path/from/repo/root.md,reason without commas`

```
see[2]{path,why}:
src/config.py,default timeout and token TTL values
docs/error-policy.md,error handling for auth failures
```

`why` must not contain commas. Use semicolons if listing multiple reasons.

---

## 5. The `~` Prefix

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

## 6. Quality Checklist

Before committing a description, check these four criteria:

**1. Decision support** — Would an agent reading only this description know whether to read the section?
If not, it's too vague.

**2. Disambiguation** — Is it specific enough to distinguish from sibling sections?
`token management` fails if there are three token-related sections; `silent rotation and expiry` does not.

**3. Precision** — Avoid generic words: `overview`; `introduction`; `details`; `information`.
Use the actual mechanism; policy; or concept the section covers.

**4. Format compliance** — Under 10 words; no commas; `~` removed.

A description that passes all four takes seconds to write and saves many minutes of agent navigation.
