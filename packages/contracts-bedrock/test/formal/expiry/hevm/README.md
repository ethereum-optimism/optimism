# L2ToL2CrossDomainMessenger equivalence: develop vs this branch

This directory checks that the new `L2ToL2CrossDomainMessenger` behaves exactly like develop's
(`c9b441bac5`, the stack's merge base with develop; its messenger is the same as at `c2fe2a991b`)
on the surface the two share, except for the differences the expiry design introduces on purpose.
The code under test is the runtime bytecode of each version, built with the repo's `foundry.toml`
default profile (solc 0.8.25, 999999 optimizer runs, cancun, `bytecode_hash = "none"`). Neither
messenger has immutables (`gen-bytecodes.sh` asserts it), so both runtimes are exactly what a
production deployment runs. The current messenger keeps its expiry period in storage (slot 5),
set by `initialize`; the harness stores the production `Constants.L2_TO_L2_MESSAGE_EXPIRY_PERIOD`
there for NEW, as a proxy initialized by `L2ContractsManager` holds it (E7 in the harness header).

| | sha256 of the runtime hex |
|---|---|
| develop `c9b441bac5` | `eaecd8fff0fa2cfe03f4c14b83379c04ccf28ec07defc6437f55f8f819c6f556` |
| this branch at `0a88e080e6` (semver 2.0.0, period set by `initialize`) | `d75090e109f1373ee5b8d3002ac3f072de16bf7a6a766e165904a0931e6c4acd` |

## What is checked

The full statement, with every exclusion and bound, is the header of
[`L2ToL2Equivalence.t.sol`](L2ToL2Equivalence.t.sol). In short:

- **Shared getters** (`successfulMessages`, `sentMessages`, `messageNonce`, `messageVersion`,
  `crossDomainMessageSender/Source/Context`): proved equivalent at bytecode level with
  `hevm equivalence`, over the same abstract storage for both codes.
- **`sendMessage` and `relayMessage`**: both versions are etched at two addresses and called with
  the same symbolic inputs from the same symbolic state. Each check asserts:
  - **(O)** the same success or revert, with byte-identical return or revert data;
  - **(X)** for relay, the same interaction with the outside: ETH movements; calls to the
    CrossL2Inbox and to the target (count and exact arguments, also on reverting paths, because
    the mocks revert on a predicate of their arguments and return or revert with their argument
    hash); the cross-domain context the target reads back; a reentrant relay attempt; the
    entered flag afterwards;
  - **(S)** the same storage, in two forms:
    - **(S-map)**, the only check of mapping entries: for one symbolic key `q`, equal
      `msgNonce` (slot 1), `successfulMessages[q]` (slot 0) and `sentMessages[q]` (slot 2), with
      the new-only `sentMessageTimestamps[q]` (slot 3), `expiredMessages[q]` (slot 4) and
      `expiryPeriod` (slot 5) pinned exactly in NEW and zero in OLD. Slots 0 to 5 are every storage
      variable of both versions (the cross-domain context is transient; send and relay never touch
      the current version's Initializable word), so any single entry of any mapping is compared.
    - **(S-all)**: a symbolic raw slot outside the pinned new-only slots (E3, and slot 5 for E7)
      is equal, and NEW's slot 5 still holds the period. Under Halmos
      0.3.3 (`check_allSlots_*`, generic layout) this covers **non-hash-derived slots only**:
      Halmos keeps keccak-derived slots in separate arrays, so the symbolic slot never aliases a
      mapping entry, and a changed mapping entry passes S-all (mutants M5 and M6 in `run.sh`
      record this; S-map catches both). S-all catches stray writes to plain slots (M1, M2).
- **Excluded on purpose**: an unsafe target (L2CrossDomainMessenger or L2ToL1MessagePasser), the
  new `sentMessageTimestamps` write, the new-only mappings and `expiryPeriod` slot (pinned, not
  compared), the removed/added/changed selectors (`initialize`, `proxyAdmin` and
  `proxyAdminOwner` are new),
  a relay to the messenger itself (unreachable through a valid identifier), and errors renamed
  with the contract-name prefix (revert data is compared with the selector mapped back to
  develop's; `gen-bytecodes.sh` generates the map from both sources).
- **Not compared**: events (neither engine can assert on logs; covered by a concrete differential
  fuzz instead) and gas (gas use and out-of-gas outcomes; NEW's extra SSTORE costs more).
- **Domain**: message lengths 0/4/37/100/128 (send) and 0/4/37/100 (relay); canonical relay
  payloads, plus arbitrary payloads of 0/100/128 bytes and 288-byte payloads with non-canonical
  `bytes` offsets and lengths; structured mock return data; one fixed relay caller.

## Engines and why both

- **hevm 0.58** (`prove_*`): `sendMessage` with the empty message, (O) and (S-all), from a
  pre-state with a symbolic `msgNonce` (and NEW's period, E7). hevm cannot handle develop's
  `sendMessage` with a non-empty message ("CopySlice with a symbolically sized region not
  currently implemented"), and its `relayMessage` run exceeds 16 GB. With the current messenger's
  bytecode two more limits show: once storage has an entry at a symbolic key, it holds the size of
  the messenger's revert data only symbolically (so the hevm engine copies return and revert data
  with a concrete size, `_sendCall`: 32 bytes on success, 0, 4 or 36 on revert, any other size a
  counterexample), and `prove_sendMessage_len0` exceeds 16 GB within two minutes, with the earlier
  harness too. So the hevm engine no longer seeds the arbitrary raw slot k := v or the E3 entries;
  the Halmos checks keep them.
- **Halmos 0.3.3** (`check_*`): every listed length, (O), (X), (S-map) with the solidity storage
  layout and (S-all) with the generic layout (`check_allSlots_*`). The two are complementary, not
  ordered: S-all sees plain slots at a symbolic index but not mapping entries; S-map sees one
  symbolic entry of every mapping. hevm's storage model for `prove_sendMessage_len0` (S-all) is
  not shown to share the Halmos limitation, nor shown free of it.

## Results (`run.sh`, all as expected)

| Group | Checks | Result |
|---|---|---|
| `hevm equivalence`, shared getters | 7 | PASS, under 1 s each |
| `hevm equivalence`, `version()` / whole `sendMessage` | 2 | counterexamples (witnesses) |
| hevm log-blindness witness | 1 | PASS (event-only difference is "equivalent") |
| hevm `prove_sendMessage_len0` | 1 | PASS, 8 s |
| hevm non-vacuity | 3 | validated counterexamples (15 to 905 s; `unsafeTarget` also had one solver timeout on another path, which does not affect its validated counterexample) |
| Halmos send (S-map, S-all), lengths 0/4/37/100/128 | 10 | PASS, 2 to 8 s each |
| Halmos relay (S-map, S-all), lengths 0/4/37/100 | 8 | PASS, 7 to 13 s each |
| Halmos relay, arbitrary and non-canonical payloads | 9 | PASS, 4 to 14 s each |
| Halmos non-vacuity | 6 | counterexamples, each replayed concretely |
| Mutants M1 to M7 (stray SSTORE to slot 6 in relay / send, wrong inbox argument, no ETH forwarded, relay does not set `successfulMessages[H]`, send does not set `sentMessages[nonce]`, send overwrites the period in slot 5) | 8 | each caught by the check named for it (M5, M6: by S-map; M7: by S-map and by the S-all period pin) |
| Known blind spot: M5, M6 under S-all (`check_allSlots_*`) | 2 | PASS, as expected (S-all does not see mapping entries) |
| Mutant M0 (unmutated source, same compile) | 4 | PASS |
| Concrete tests: event fuzz (4 tests) and witness replays (6 tests) | 10 | PASS |

Last run: `0a88e080e6`, `run.sh` "overall: all as expected" (62 checks `ok`, none unexpected), 27 min on a
loaded 32-core host.

A PASS means: complete exploration, no counterexample, no timeout or unknown result, no bounded
loop (from Halmos's JSON output). Non-vacuity checks and mutants need a real counterexample:
hevm's must be validated; Halmos's must come with a model (models that mention keccak are
reported "potentially invalid" by Halmos, which is why each is also replayed in
[`L2ToL2EquivalenceConcrete.t.sol`](L2ToL2EquivalenceConcrete.t.sol)).

## Files

- `L2ToL2Equivalence.t.sol`: the harness (shared base, hevm contract, Halmos contract, mocks).
- `L2ToL2EquivalenceConcrete.t.sol`: event fuzz and witness replays (runs in the repo's
  `forge test`; fuzz budget capped for `ciheavy`).
- `L2ToL2Bytecodes.sol`, `develop.runtime.hex`, `current.runtime.hex`: generated by
  `gen-bytecodes.sh` from the two checkouts.
- `foundry.toml`: standalone project, so the engines load only these artifacts.
- `run.sh`: runs everything and writes `results/summary.txt` (gitignored).

## Running

```sh
# Regenerate (or check) the bytecode; the develop checkout needs the same lib/ submodules.
./gen-bytecodes.sh <develop packages/contracts-bedrock dir> --check
# Everything; every engine call runs in a systemd scope with MemoryMax=16G and no swap.
./run.sh
```

Needs hevm 0.58, z3 4.13.3, halmos 0.3.3, forge, python3, solc 0.8.25 (`SOLC=`), and a systemd
user manager. `run.sh` takes about 15 to 25 minutes. Mutants are generated into `mutants/` while the
mutant step runs and deleted afterwards.

## Review log

- **v1** (first hand-off): bytecode-level getters, a Halmos harness for send/relay checking only
  slot 1 and mapping entries, hevm for the empty-message send, and an event fuzz.
- **v2**, after a three-way review (R1 fresh-context reviewer; R2, R3 independent model-based reviewers; verdict: useful, needs
  tightening, bytecode reproduces exactly, nothing critical):
  - The runner could certify incomplete runs. Halmos verdicts now come from `--json-output`
    (bounded loops, timeouts, all-revert and errors are failures); witnesses need an actual
    counterexample (hevm: validated; bare `[FAIL]` is rejected); the whole-`sendMessage`
    bytecode witness is relabelled (E2 also separates safe-target sends) and records the targets
    of the counterexamples hevm reports instead of asserting them.
  - Storage was compared only at slot 1 and mapping entries under Halmos, so a stray SSTORE could
    survive. Added `check_allSlots_*` (a symbolic raw slot, generic layout; see v4 for its limit) for every send and relay
    length, and mutants M1/M2 that a stray `sstore(5, 1)` is caught.
  - Call equality covered only committed effects. The mocks now revert on a predicate of their
    arguments and return or revert with their argument hash, so arguments are compared on
    reverting paths too; mutant M3 (another hash to the inbox) is caught.
  - Gas is now stated as excluded.
  - The domain is documented (payload shape, structured return data, relay lengths, concrete
    chainid in both engines), and arbitrary or non-canonical relay payloads are now checked.
  - The relay event fuzz fixes the origin to the messenger, has a variant that hits the success
    path on every run, and tests invalid origins separately; `run.sh` runs it.
  - The post-call context check is relabelled (it compares the entered flag only); the hevm
    transient-storage caveat is stated; `gen-bytecodes.sh --check` compares all three generated
    files.
  - Repo conventions: `Test` from `test/setup/Test.sol`, NatSpec, named returns, shellcheck,
    semgrep and `forge fmt` clean, `deny = "warnings"`.
  - Re-verified against `52ff613e14` (exporter at 0x...0030): the messenger bytecode is
    unchanged.
  - Ready for the error renames of the guideline pass (E6): revert data is compared modulo a
    selector map that `gen-bytecodes.sh` derives from both sources (empty at `52ff613e14`).
- **Retarget to `448d31ad19`** (tip of the PR #23259 branch): regenerated with
  `gen-bytecodes.sh`. Relative to the committed `52ff613e14` runtime (sha256 `61e81194…c541`), only
  four `PUSH32` operands changed: the error selectors renamed with the
  `L2ToL2CrossDomainMessenger_` prefix (pcs 1296 and 3038 `MessageTargetUnsafe`, 2057
  `NotOtherMessenger`, 2198 `MessageNotExpired`); the version string is `"2.0.0"` again at this
  commit. None of the renamed errors exists in develop's messenger, so the E6 selector map stays
  empty (they are reached only on excluded paths: unsafe targets and `expireMessage`). The new
  runtime is byte-identical to `../evm-lean/bytecode/L2ToL2CrossDomainMessenger.runtime.hex` (same
  commit; its init code matches `semver-lock.json`). `gen-bytecodes.sh --check` passes; the three
  hevm files compile under `FOUNDRY_PROFILE=liteci` with forge 1.8.3 in the repo build (no
  warnings); `run.sh`: all as expected (every check `ok`, none unexpected; 25 min 26 s on a
  loaded 32-core host, the slowest check `prove_nonvacuity_sendMessage_unsafeTarget` at 932 s).
  Two fixes to the runner setup: `run.sh` is now executable (it was committed as `100644`), and
  `foundry.toml` sets `[lint] lint_on_build = false` as the repo's own `foundry.toml` does (with
  `deny = 'warnings'`, forge 1.8.1's post-build lint failed the harness build on lint findings).
- **Retarget to `0a88e080e6`** (the period set by `initialize`): the current messenger has no
  immutables, so `gen-bytecodes.sh` asserts that for both codes and takes the plain deployed
  bytecode (sha256 above; byte-identical to `../evm-lean/bytecode/L2ToL2CrossDomainMessenger.runtime.hex`).
  The period lives in slot 5, a new-only variable: the harness stores the production period there
  for NEW (E7), S-map pins it exactly in NEW and zero in OLD, and S-all skips slot 5 in the OLD/NEW
  comparison and asserts NEW's period unchanged instead, as E3 does for slots 3 and 4. The stray
  write of mutants M1/M2 moved from slot 5 to slot 6, and M7 (send overwrites slot 5) is new. The
  mutants deploy without a constructor argument. hevm needed two changes (see "Engines"): a
  concrete-size copy of sendMessage's output, and a pre-state without symbolic-key entries. The
  `foundry.toml` gains the OpenZeppelin v5 remapping the messenger now imports (for the mutants).
  The witness verdict now accepts a validated counterexample even when another path was explored
  only partially; a PASS still needs complete exploration.
- **v4** (cross-layer mutation campaign, `../mutation/README.md` "Findings"): the Halmos S-all
  checks cannot see a changed mapping entry. Mutant K07 there (relay does not set
  `successfulMessages[H]`) passed all nine `check_allSlots_*` and was caught only by S-map.
  Cause: Halmos 0.3.3's generic layout decodes a keccak-derived slot into its preimage and keeps a
  separate array per preimage width, so the free raw slot never aliases a mapping entry.
  - The README and the harness header no longer say "every raw slot" or "a PASS there is at
    least as strong": S-all covers non-hash-derived slots; S-map is the only check of mapping
    entries, and it already compares every mapping both versions have (slots 0 to 4: one
    symbolic key each, with the new-only ones pinned), so no new assertion was needed.
  - `run.sh` gains mutants M5 (relay does not set `successfulMessages[H]`, i.e. K07) and M6
    (send does not set `sentMessages[nonce]`): each must be caught by its S-map check, and each
    must PASS its S-all check, recorded as a known blind spot so a change in either direction
    shows up as UNEXPECTED. M0 now also runs `check_sendMessage_len37`.
  - hevm's `prove_sendMessage_len0` (S-all under hevm's storage model) is not claimed to share
    or avoid the limitation.
