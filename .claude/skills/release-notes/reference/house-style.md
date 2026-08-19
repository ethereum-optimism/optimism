# House style for OP Stack release notes

Derived from published releases: `op-node/v1.19.3`, `op-node/v1.19.4`,
`op-batcher/v1.16.12`, `kona-node/v1.6.3`, `op-supernode/v1.0.0`.

## Structure

A published release note is the raw git-cliff draft with a header bolted on top and the
PR list pruned. Everything below `## What's Changed` keeps git-cliff's exact formatting.

1. **Verdict callout** — exactly one, always first. Tells an operator whether to upgrade.
2. **Topic callouts** — zero or more, for specific operator-facing changes.
3. **Standing notices** — recurring boilerplate (e.g. the APKO migration block).
4. **`### Breaking / behavior changes`** — optional bulleted section, when there are
   several distinct changes that each need a sentence of explanation.
5. **`## What's Changed in <tag>`** — the pruned PR list, git-cliff bullets verbatim.
6. **`## New Contributors`** — optional.
7. **`**Full Changelog**`** line — never edit.
8. **Docker image line(s)** — never edit.

## The verdict callout

Exactly one, chosen by the worst thing in the release:

| Callout | Lead-in | Use when |
| --- | --- | --- |
| `> [!CAUTION]` | `**Mandatory upgrade for ...**` | Not upgrading risks a halt, consensus divergence, or stalled safe-head progression |
| `> [!WARNING]` | `**Security fix — upgrade recommended.**` | Closes a vulnerability, or behaviour changes that can break startup/config |
| `> [!NOTE]` | `**Optional upgrade.**` | Routine fixes and improvements |
| `> [!NOTE]` | `**Optional upgrade — no functional changes.**` | Only refactors/alignment ship |
| `> [!IMPORTANT]` | topic sentence | Milestone releases (e.g. first stable) that are not risk-ranked |

Keep it to one to three sentences. Scope it: name the affected chains or configurations
and say who is *not* affected, as `op-node/v1.19.3` does with "Mode, Metal, and Zora ...
OP Mainnet is unaffected."

## Voice

- **Bold lead-in fragment, then plain prose.** The bold part is the verdict; the rest is
  the reason.
- **Every technical claim carries an inline PR link** in the form
  `([#21674](https://github.com/ethereum-optimism/optimism/pull/21674))`.
- **Write the operator consequence, not the code change.** "an op-node on v1.19.2 or
  earlier can derive a different post-Karst gas limit and halt safe-head progression"
  beats "syncs the embedded registry configs."
- **Say what the operator must do** when anything is required of them.
- Use `**bold**` inside callouts for the thing that changed; keep sentences short.

## Worked examples

Mandatory, tightly scoped, with the consequence spelled out:

```markdown
> [!CAUTION]
> **Mandatory upgrade for operators of Mode, Metal, and Zora using the built-in `--network` configs.** The previously embedded registry data set `keep_karst_upgrade_gas=true` for these chains, but their sequencers activated Karst with `false` — an op-node on v1.19.2 or earlier can derive a different post-Karst gas limit and halt safe-head progression. This release syncs the embedded configs to the canonical behavior ([#21674](https://github.com/ethereum-optimism/optimism/pull/21674)), so explicit overrides are no longer needed. OP Mainnet is unaffected (`true` remains correct). Recommended for all other operators.
```

Security, with the fixes enumerated inline:

```markdown
> [!WARNING]
> **Security fix — upgrade recommended.** This release closes three unauthenticated attack surfaces ([#21753](https://github.com/ethereum-optimism/optimism/pull/21753))
> and hardens gossip by binding the message-id to the topic and metering invalid messages ([#21804](https://github.com/ethereum-optimism/optimism/pull/21804)).
```

A release that carries nothing functional — say so plainly rather than padding:

```markdown
> [!NOTE]
> **Optional upgrade — no functional changes.** This release contains no behavioural changes to op-batcher.
> It is published to keep op-batcher aligned with the rest of the op-stack release, and carries only a
> refactor moving shared `DepositTx` and transaction helpers into `op-core`.
```

Several distinct behaviour changes — bundle them as a bulleted `[!WARNING]`:

```markdown
> [!WARNING]
> **Behaviour changes that can affect startup and configuration:**
> - op-supernode now **fails startup** when the deny list cannot be opened, instead of continuing silently ([#21997](https://github.com/ethereum-optimism/optimism/pull/21997)).
> - Interop log backfill now defaults to **7 days** ([#20577](https://github.com/ethereum-optimism/optimism/pull/20577)); depth is also settable via env var ([#20566](https://github.com/ethereum-optimism/optimism/pull/20566)).
```

The `### Breaking / behavior changes` section, when each item needs more than a clause
(from `op-node/v1.19.3`) — note the third entry, which is explicitly flagged as affecting
downstream Go importers but *not* operators:

```markdown
### Breaking / behavior changes

* **Strict registry decoding** ([#21648](https://github.com/ethereum-optimism/optimism/pull/21648)): unknown keys in superchain-registry TOML now fail loudly at startup instead of being silently dropped. No change with the embedded registry, but custom registry data carrying unmodeled keys will no longer load.
* **Go API move** ([#21650](https://github.com/ethereum-optimism/optimism/pull/21650)): `LoadOPStackRollupConfig` moved from `op-node/rollup` to `op-node/superchain`. Downstream Go importers must update their import path; no operator-facing change.
```

## Standing notices

Recurring boilerplate that spans several releases, carried verbatim from the previous
release of the same component. The APKO migration block is the current example, and its
text is self-limiting ("In this release (and only in this release) ..."), so it is not
safe to carry forward blindly.

**Always ask** whether a standing notice still applies — see step 6 of `SKILL.md`. Pull
the exact text from the previous published release rather than retyping it:

```bash
gh release view <previous-published-tag> --json body -q .body
```

Note that releases carrying the APKO notice also publish two image lines, which git-cliff
does not generate:

```markdown
🚢 Docker built Image: https://us-docker.pkg.dev/oplabs-tools-artifacts/images/<component>:<version>
🚢 Apko built Image: https://us-docker.pkg.dev/oplabs-tools-artifacts/images/nexus/<component>:<version>
```

If the notice is dropped, leave git-cliff's single `🚢 Docker Image:` line alone.

## New Contributors

git-cliff generates this section. Keep it, but:

- **Drop bot rows** — `@dependabot[bot]`, `@oplabs-renovate[bot]`, `@claude[bot]`.
  `op-node/v1.19.4` dropped the section entirely because its only entry was a bot.
- **Drop rows whose PR was pruned.** Crediting a first contribution against a PR that is
  not in the list reads as an error, and the contributor is credited in whichever
  component's release actually ships their work. The same contributor can therefore
  appear in one component's notes and not another's from the same train.
- Drop the whole section if nothing survives.

## Things never to touch

- The `**Full Changelog**` compare link. It points at the previous *RC* tag, not the
  previous stable one. That is git-cliff's output and published notes leave it as-is. The
  one exception is a draft whose sections you merged: widen the link's base to the
  earliest merged section's base, or the note will claim a narrower range than it covers.
- The bullet format inside `## What's Changed`: `* <title> by @<author> in [#<n>](<url>)`.
  Prune whole lines; never reword them. The prose explaining a change belongs in the
  callouts, where it can carry the operator consequence.
- The trailing `<!-- generated by git-cliff -->` marker.
