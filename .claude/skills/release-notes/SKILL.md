---
name: release-notes
description: Turn a raw `just release-notes` draft into published, house-style release notes for an OP Stack component (op-node, op-batcher, op-supernode, kona-node, op-proposer, op-challenger, op-reth, op-deployer). Prunes PRs that never reach the component's binary and writes the operator-facing notices that go at the top. Use when asked to write, polish, tidy, or finalize release notes, to clean up a draft GitHub release, or to prepare notes before publishing a release.
---

# Release notes

`just release-notes <component>` emits a git-cliff changelog: every PR whose diff hit the
component's include paths, in one flat list. A published release note is a different
artifact — it opens with a verdict an operator can act on, and it lists only the changes
that actually reach the binary.

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
you get. Look for: how the verdict callout is phrased, how aggressively the
`### Breaking / behavior changes` section is trimmed, whether callouts were merged, and
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

**Comment dropped bullets out; do not delete them.** A commented bullet can be reinstated
by a reviewer in one edit, and it shows what was considered. Keep the reason short and put
it before the closing `-->`:

```markdown
<!--* op-core/fees: add Jovian DA-footprint calculation by @claude[bot] in [#22163](...) doesn't affect the batcher-->
```

Batch the PR lookups — read intent for all the `LINKED` rows in parallel rather than one
at a time. If there are more than ~15, delegate the reading to a sub-agent and ask for a
one-line "what an operator would notice" per PR, so the diffs stay out of context.

Never reword a surviving bullet.

## 6. Pick the verdict

One callout, set by the most serious thing that survived triage:
`[!CAUTION]` mandatory → `[!WARNING]` security or startup/config breakage → `[!NOTE]`
optional. `reference/house-style.md` has the ladder and the phrasing.

Write it in terms of the symptom an operator would notice, not the mechanism. If nothing
functional survived, say exactly that — do not manufacture significance.

## 7. Check the standing notices

Recurring boilerplate (the APKO migration block was one) is carried between releases and
is **not** safe to copy forward blindly — the APKO text said "in this release (and only in
this release)". Step 2's output shows whether the previous release carried one.

**[USER REVIEW]** If the previous release has a standing notice, show it and ask whether it
still applies. If it does, copy it verbatim, along with whatever image lines it implies.

## 8. Assemble

Order: verdict callout → topic callouts → standing notices → optional
`### Breaking / behavior changes` → `## What's Changed in <tag>` → `## New Contributors` →
`**Full Changelog**` → image line(s). Write to `/tmp/<component>-notes.md`.

## 9. Retarget RC references when finalizing

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

## 10. Review before applying

**[USER REVIEW]** Show the user both:

1. the proposed notes in full, and
2. **the drop list** — every pruned PR with its tag and a few words on why.

The drop list is the part that needs human eyes. A wrongly pruned PR is invisible in the
rendered note, so present it as a list to check.

## 11. Apply

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
