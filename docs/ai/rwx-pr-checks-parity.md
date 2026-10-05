# Module preparation and contract fast checks

Complete original-report parity passes at
`3aff9e64f8a797b8cfe92b5c5b2d30ffe91c0628` on
[RWX run 017f7c70](https://cloud.rwx.com/optimism/runs/017f7c703c884cdca57087885d8188ce).
Circle's Main prep job 5634033, contract prep job 5634049 and complete contract-fast
job 5634053 passed. The shared module producer serves both consumers.

Both original Circle module graphs match all 463 native modules, including
versions, content hashes, replacements and other module identity fields.
Download retry policy remains five attempts with the original exponential
backoff; all observed runs succeeded on the first attempt. The runner additionally
executes `go mod verify`. Circle omits two zero-byte logs per prep job: each was
restored only from its original empty-file digest. No nonempty original is missing.

All sixteen `checks.yaml` checks execute through the original
`just check-fast -verbose` runner. Phase builds, commands, dependencies and clean
retry behavior remain authoritative. Both providers report sixteen initial
passes and zero retries. Foundry configuration, pinned tools, complete check
configuration, JUnit identities and original command histories agree.
All nineteen recursive submodule paths and commit hashes match. Eight descriptive
Git labels differ because the providers retain different tags/remote refs;
these labels do not change the checked-out commits. Both original texts are
retained. Uninitialized/mismatched status prefixes, missing/extra/duplicate paths
and changed commit hashes fail validation.

The original target-branch resolver selects `develop`. Both providers bind
target and merge-base revision `c8e4ba855d79ca56463909ef5a2c5830a1189401`.
The shared runner fetches that protected ref before semver checks and verifies
that it does not change during execution. Native module preparation uses
4 CPU / 8 GiB; the fast check worker uses 16 CPU / 32 GiB. Go compiler and
Foundry caches remain reusable, while the fast verdict includes native run and
attempt identity in its cache key. Reports and check-runner result caches are
excluded from reusable outputs. Native warming prepares modules/tools with
zero fast-check verdicts.

[First-failure evidence](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-pr-checks-evidence/first-failure.json) retains the
missing pinned uv bootstrap and missing protected Git ref. The bootstrap-only
diagnostic passed without running checks. The second native run passed fifteen
checks and failed semver comparison while Circle passed on the same SHA; the
original failure remains unchanged. A real single-branch Git clone fixture
reproduces the absent ref and proves the fetch correction. Actual Go fixtures
exercise dependency execution, a configured clean retry, and the relative Circle
module adapter. Ten execution/adapter fixtures and fourteen comparison fixtures
pass; the shared stage's 27 Rust workspace regressions also pass (one separate
opt-in live Rust fixture skipped).

[Parity index](https://github.com/ethereum-optimism/optimism/blob/a1aa49aaf3713a8172f3f615f094488fd8e39c3d/docs/ai/rwx-pr-checks-evidence/parity.json) retains complete selections,
commands, module graphs, all original report hashes, provider identities and
comparison tool hashes. Complete originals remain under
`.ci/rwx-pr-checks-stage-evidence/3aff/` and immutable provider artifacts. This
closes two additional contract occurrences. Main module preparation was already
counted; it receives stronger evidence without being counted twice. Overall
coverage is **49/86 (57%)**. All four required Circle gates, dependency review and all seven optional RWX
checks passed at this exact benchmark SHA: 145 successful checks, one neutral,
no unfinished or failed checks. This checkpoint does not establish readiness
for a later revision.
