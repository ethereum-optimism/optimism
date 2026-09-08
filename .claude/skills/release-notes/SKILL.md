---
name: release-notes
description: Write the release notes for an OP Stack component using the house style. Use when asked to write, polish, tidy, or finalize release notes, to clean up a draft GitHub release, or to prepare notes before publishing a release.
---

# Release notes

`just release-notes <component>` emits a git-cliff changelog: every PR whose diff hit the
component's include paths, as a flat list of PR titles. A published note is a different
artifact — a **curated change list**, grouped by theme, where each entry explains itself and
only changes that reach the binary appear at all.

The raw list is replaced, not annotated. Read `reference/house-style.md` before writing and
`reference/triage.md` before pruning.

This skill does not create or finalize tags.

## 1. Establish the target

Ask which component and which release, unless the user already said.

```bash
gh release list --limit 20        # 'Draft' rows are the candidates
```

**Check for both an RC and a finalized draft of the same version.** Release automation
creates a fresh draft when an RC is finalized, and that finalized draft is the one that gets
published. If both exist, target the finalized one and say so.

**If the newest tag is an RC with no finalized counterpart, ask before writing anything.**
Tags may not be local yet, so fetch first:

```bash
git fetch --tags
git tag -l '<component>/v*' --sort=-v:refname | head -5
```

Notes written against an RC carry `-rc.N` in the heading, compare link and image tag, and
have to be retargeted later (step 8) — a retarget that refuses because the finalized tag
sits on a different commit means regenerating from scratch. So ask the release manager
whether they want to finalize with `just release` first. It is their call: this skill does
not create or finalize tags, and `just release` is slow, so do not stall on it — write the
RC notes if they say no.

## 2. Get the draft

```bash
gh release view <tag> --json body -q .body > /tmp/<component>-draft.md
```

If there is no draft yet:

```bash
GITHUB_TOKEN=$(gh auth token) mise exec -- just release-notes <component>              # latest stable -> latest RC
GITHUB_TOKEN=$(gh auth token) mise exec -- just release-notes <component> latest develop
```

git-cliff is a mise-pinned tool and is not on `PATH`, so without `mise exec --` the recipe
fails with `git: 'cliff' is not a git command`, which names neither git-cliff nor mise.

More than one `## What's Changed in ...` section means earlier RCs were never published.
Merge them under the final tag and dedupe by PR number.

## 3. Find out what each PR actually touched

```bash
.claude/skills/release-notes/scripts/pr-facts.sh /tmp/<component>-draft.md <component>
```

Each PR is tagged `LINKED` (changed a package compiled into the binary, and which ones),
`CONFIG` (moved the embedded superchain registry — `LINKED+CONFIG` when it did both),
`DEPS` (manifests, no compiled package), `--` (nothing the binary compiles) or `?`
(unresolvable, or the PR could not be fetched). The tags come from `go list -deps` and
`cargo tree`, so linkage is exact — but it proves the package is compiled in, not that the
changed function is on the component's runtime path.

## 4. Triage

Drop `--` rows, then apply the judgment pass in `reference/triage.md` to every `LINKED` row.
Never drop a `CONFIG` row without reading it, and check a `DEPS` row that lists paths — the
lock diff may carry a security fix for a library this binary links.

Read the intent of each surviving PR — you cannot write a self-contained entry from a title.
Batch the lookups; if there are more than ~15, delegate the reading and ask for a one-line
"what an operator would notice" per PR.

Keep the raw bullets for cut PRs as HTML comments at the bottom of the draft, each with a
short reason, so a reviewer can reinstate one in a single edit.

## 5. Curate the change list

The substance of the job. Replace PR titles with grouped, self-contained entries.

First lift out the two entries that get their own top-level section: anything an operator
must act on before upgrading (`## Breaking changes`) and anything that moved a chain's
embedded config (`## Chain Configuration`) — see step 6. Everything else:

- under `## Other changes`, group into `### Features` / `### Bug fixes` /
  `### Miscellaneous`, or by domain; drop empty headings, and use a flat list for a short
  release
- derivation changes, in op-node or kona, get their own `### Derivation` heading, first
- fold PRs that are one logical change into one entry carrying all their numbers
- **write impact, never implementation** — the symptom that appears or disappears, the
  flag/metric/config names, what the reader must do. Not goroutines, event loops, call
  paths, internal type names, or which PR was stacked on which. This is the correction made
  most often, and the PR descriptions you just read will pull you the wrong way
- **a change with no user-visible impact does not appear at all** — not under
  `### Miscellaneous`, not as a summarising line. Importability of the monorepo as a Go
  module is not user impact: we do not maintain releases of it as a Go module, so a change
  that only helps downstream importers is cut like any other internal churn
- reference PRs as bare `(#NNNNN)`; no `by @author`
- only mention a PR more than once if it included multiple logical changes which are worth describing separately

## 6. Write the Overview

One callout at the top of `## Overview`: which of `optional` / `recommended` / `strongly
recommended` / `required` the release is, scopable to a role, and what it contains. Never a
semver release type — our tags are not strict semver, so "this is a minor release" says
nothing. `> [!NOTE]` for `optional`, `> [!IMPORTANT]` for either recommended level,
`> [!CAUTION]` for `required`.

Then check proportionality:

- **Is the feature live?** Verify with the registry check in `reference/house-style.md`
  rather than assuming; ask the release manager for anything not expressed as a hardfork. A
  change confined to an unreleased feature is cut entirely. It stays only if it also reaches
  a live path, and is then described by that live effect — read what the change does, not
  the feature name in its PR title. Either way, do not describe an attack the live system
  cannot suffer.
- **Would a reader shrug?** New metrics, a wrong version string and rare corner cases are
  bullets, not callouts.

Add `## Breaking changes` only when an operator must *do* something before upgrading;
Go-API-only changes do not qualify. A release that moves a chain's embedded config — a new
hardfork activation time, most often — gets a `## Chain Configuration` section of its own.
The Overview block is the note's only callout.

## 7. Assemble

Order: `## Overview` → optional `## Breaking changes` → optional `## Chain Configuration` →
`## Other changes` with its subheadings → `**Full Changelog**` → image line → commented-out
working notes. Write to `/tmp/<component>-notes.md`.

## 8. Retarget RC references when finalizing

A draft generated against an RC carries `-rc.N` in its heading, compare link and image tag.
Publishing that finalized points operators at an RC image.

```bash
.claude/skills/release-notes/scripts/retarget-tag.sh /tmp/<component>-notes.md <component>
```

It **refuses**, leaving the file untouched, if the finalized tag is missing or sits on a
different commit than the RC. Surface that warning rather than editing by hand — a note
generated for a different commit needs regenerating, not retagging.

Then set the compare link's base to the previous **finalized** tag.

## 9. Review before applying

**[USER REVIEW]** Show the proposed notes in full **and the drop list** — every pruned PR
with its tag and a few words on why. A wrongly pruned PR is invisible in the rendered note,
so present it as a list to check.

## 10. Apply

Only after explicit approval:

```bash
gh release edit <tag> --notes-file /tmp/<component>-notes.md
```

This changes only the body, leaving draft/published state alone. `edit` needs a release
object to exist; when the tag has no draft, create one instead:

```bash
gh release create <tag> --draft --title '<tag>' --notes-file /tmp/<component>-notes.md
```

Add `--prerelease` for an RC. A draft's URL is an `untagged-<hash>` link until it is
published — that is normal, and `gh release view <tag>` still resolves it.

Confirm with `gh release view <tag>` and report what changed.

## Notes

- A change under `op-service/` or `op-core/` can change a component's behaviour. Never prune
  on directory name alone; that is what step 3 is for.
- Check a judgment call against history by examining recent releases for the same component.
- Never overwrite a draft a human has curated. If the body carries hand-written prose or
  commented-out bullets, propose specific edits instead of replacing it.
