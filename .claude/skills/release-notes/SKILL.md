---
name: release-notes
description: Turn a raw `just release-notes` draft into a published, house-style release note for an OP Stack component (op-node, op-batcher, op-supernode, kona-node, op-proposer, op-challenger, op-reth, op-deployer). Cuts PRs that never reach the component's binary, then replaces the raw PR list with a curated change list grouped by theme, under a standard Overview sentence giving the upgrade recommendation. Use when asked to write, polish, tidy, or finalize release notes, to clean up a draft GitHub release, or to prepare notes before publishing a release.
---

# Release notes

`just release-notes <component>` emits a git-cliff changelog: every PR whose diff hit the
component's include paths, in one flat list of PR titles. A published release note is a
different artifact — a **curated change list**, grouped by theme, where each entry explains
itself without relying on the PR title, and only changes that actually reach the binary
appear at all.

The raw list is replaced, not annotated. `op-challenger/v1.9.4` and
`op-contracts/v8.0.0-rc.2` are the reference for what to aim at.

This skill does that transformation. It does **not** create tags or finalize RCs; see the
`op-challenger-release` / `op-proposer-release` skills for the tagging flow.

Read `reference/triage.md` before pruning and `reference/house-style.md` before writing
prose. Both are short and carry the worked examples.

## 1. Establish the target

Ask which component(s) and which release, unless the user already said. Draft releases
awaiting notes are usually the answer:

```bash
gh release list --limit 20        # 'Draft' rows are the candidates
```

**Check for both an RC and a finalized draft of the same version.** Release automation
creates a fresh draft when an RC is finalized, and that finalized draft is the one that
gets published. Writing to the RC draft instead is wasted work. If both exist, target the
finalized one and say so.

Handle each component as a separate pass — the notices differ per component even when they
share a release train.

## 2. Learn from recent releases first

Release managers edit these notes after the skill drafts them, and those edits *are* the
house style. Read them before writing anything:

```bash
.claude/skills/release-notes/scripts/recent-style.sh <component> 4
```

Also diff against your own previous output when a draft you produced has since been
published or hand-edited — the delta is the correction, and it is the most valuable input
you get. Look for: how the Overview sentence is phrased, which headings the change list is
grouped under, how much detail an entry carries, what was judged too minor to mention, and
what the compare link's base is.

**When recent releases disagree with `reference/house-style.md`, the releases win.** Say
what drifted and offer to update the reference doc, so the next run starts from the
corrected convention instead of rediscovering it.

## 3. Get the draft

```bash
gh release view <tag> --json body -q .body > /tmp/<component>-draft.md
```

If there is no draft yet, generate one:

```bash
GITHUB_TOKEN=$(gh auth token) just release-notes <component>              # latest stable -> latest RC
GITHUB_TOKEN=$(gh auth token) just release-notes <component> latest develop
```

If the draft contains more than one `## What's Changed in ...` section, earlier RCs were
never published. Merge them into one section under the final tag and dedupe by PR number
(see `reference/triage.md`).

## 4. Find out what each PR actually touched

```bash
.claude/skills/release-notes/scripts/pr-facts.sh /tmp/<component>-draft.md <component>
```

Each PR is tagged `LINKED` (changed a package compiled into the binary, and which ones),
`DEPS` (manifests only), `--` (nothing the binary compiles), or `?` (unresolvable — judge
by hand). The tags come from `go list -deps` and `cargo tree`, so linkage is exact.

Linkage is a necessary condition, not a sufficient one: it proves the package is compiled
in, not that the changed *function* is on the component's runtime path. See
`reference/triage.md`.

## 5. Triage

Drop `--` and non-security `DEPS` rows. For every `LINKED` row apply the judgment pass in
`reference/triage.md`.

Read the intent of every surviving PR — you cannot write a self-contained entry from a PR
title. Batch the lookups rather than going one at a time; if there are more than ~15,
delegate the reading and ask for a one-line "what an operator would notice" per PR.

Keep the raw bullets for cut PRs as HTML comments at the bottom of the draft, each with a
short reason, so a reviewer can see what was considered and reinstate one in a single edit.

## 6. Curate the change list

This is the substance of the job. Replace the PR titles with grouped, self-contained
entries — see "Curating the change list" in `reference/house-style.md`:

- group by `### Features` / `### Bug fixes` / `### Other`, or by domain where that carries
  more meaning; drop headings that would be empty, and use a flat list for a short release
- fold PRs that are one logical change into one entry with all their numbers
- write each entry so it stands alone, saying what changed and why an operator cares
- reference PRs as bare `(#NNNNN)`; no `by @author`
- mention each PR exactly once

## 7. Write the Overview and check proportionality

Open `## Overview` with the standard recommendation sentence in a callout: release type,
what it contains, and one of `optional` / `optional but recommended` / `recommended` /
`required`. The block is always `> [!NOTE]`, except `required`, which uses `> [!CAUTION]` —
severity is carried by the sentence's fixed vocabulary, not by the block type.

Then apply the proportionality checks from `reference/house-style.md` before highlighting
anything:

- **Is the feature live in production?** Interop/Lagoon, ZK dispute games and super dispute
  games are not. Changes to dormant paths go under a `### <Feature> (not yet in production)`
  heading with the standard Note paragraph in the Overview — they cannot affect operators
  today, however alarming the description sounds.
- **Would a reader shrug?** New metrics, wrong version strings and rare corner cases are
  bullets, not callouts.

Add a `## Breaking changes` section only when an operator must *do* something before
upgrading. Go-API-only changes do not qualify.

The Overview block is the **only** callout in the note. Anything reaching for a second one
is an entry in the change list, or a breaking change.

## 8. Check the standing notices

Recurring boilerplate (the APKO migration block was one) is carried between releases and
is **not** safe to copy forward blindly — the APKO text said "in this release (and only in
this release)". Step 2's output shows whether the previous release carried one.

**[USER REVIEW]** If the previous release has a standing notice, show it and ask whether it
still applies. If it does, copy it verbatim, along with whatever image lines it implies.

## 9. Assemble

Order: optional `[!CAUTION]` → `## Overview` → optional `## Breaking changes` →
`## What's Changed` with its subheadings → `## New Contributors` → `**Full Changelog**` →
image line(s) → commented-out working notes. Write to `/tmp/<component>-notes.md`.

## 10. Retarget RC references when finalizing

A draft generated against an RC carries `-rc.N` in its heading, its compare link and its
image tag. A finalized release publishes those as the plain version — pointing operators
at an RC image is a real defect, not a cosmetic one.

```bash
.claude/skills/release-notes/scripts/retarget-tag.sh /tmp/<component>-notes.md <component>
```

The script rewrites the heading, the compare link's right side and the image tag, and
**refuses** — leaving the file untouched and exiting non-zero — if the finalized tag does
not exist or points at a different commit than the RC. If it refuses, surface the warning
to the release manager rather than editing by hand; a note whose PR list was generated for
a different commit needs regenerating, not retagging.

Then set the compare link's base to the previous **finalized** tag (see
`reference/house-style.md`).

## 11. Review before applying

Check that no PR is described twice — the curated style has no raw list to fall back on, so
a double mention is a real defect. Only `## New Contributors` rows may repeat a number:

```bash
grep -oE '#2[0-9]{4}' /tmp/<component>-notes.md | sort | uniq -d
```

**[USER REVIEW]** Show the user both:

1. the proposed notes in full, and
2. **the drop list** — every pruned PR with its tag and a few words on why.

The drop list is the part that needs human eyes. A wrongly pruned PR is invisible in the
rendered note, so present it as a list to check.

## 12. Apply

Only after explicit approval:

```bash
gh release edit <tag> --notes-file /tmp/<component>-notes.md
```

This leaves the release in its current draft/published state and changes only the body.
Confirm with `gh release view <tag>` and report what changed.

## Notes

- A change under `op-service/` or `op-core/` can absolutely change a component's
  behaviour. Never prune on directory name alone; that is what step 4 is for.
- Verify a judgment call against history by regenerating a published release's raw draft:
  `GITHUB_TOKEN=$(gh auth token) just release-notes op-node v1.19.3 v1.19.4`.
- Never overwrite a draft a human has already curated. If the body carries hand-written
  prose or commented-out bullets, treat it as the authority: propose specific edits
  instead of replacing it.
