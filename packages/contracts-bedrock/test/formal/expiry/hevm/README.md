# L2ToL2CrossDomainMessenger equivalence: develop vs this branch

This directory checks that the new `L2ToL2CrossDomainMessenger` behaves exactly like develop's
(`c2fe2a991b`) on the surface the two share, except for the differences the expiry design
introduces on purpose. The code under test is the runtime bytecode of each version, built with
the repo's `foundry.toml` default profile (solc 0.8.25, 999999 optimizer runs, cancun,
`bytecode_hash = "none"`, no immutables).

| | sha256 of the runtime hex |
|---|---|
| develop `c2fe2a991b` | `eaecd8fff0fa2cfe03f4c14b83379c04ccf28ec07defc6437f55f8f819c6f556` |
| this branch at `448d31ad19` (semver 2.0.0) | `aa912696762430ae5013b842fe075ecaa7e95610b7be4737fc85a6d9f305aeb9` |

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
      the new-only `sentMessageTimestamps[q]` (slot 3) and `expiredMessages[q]` (slot 4) pinned
      exactly in NEW and zero in OLD. Slots 0 to 4 are every storage variable of both versions
      (the cross-domain context is transient), so any single entry of any mapping is compared.
    - **(S-all)**: a symbolic raw slot outside the pinned new-only slots is equal. Under Halmos
      0.3.3 (`check_allSlots_*`, generic layout) this covers **non-hash-derived slots only**:
      Halmos keeps keccak-derived slots in separate arrays, so the symbolic slot never aliases a
      mapping entry, and a changed mapping entry passes S-all (mutants M5 and M6 in `run.sh`
      record this; S-map catches both). S-all catches stray writes to plain slots (M1, M2).
- **Excluded on purpose**: an unsafe target (L2CrossDomainMessenger or L2ToL1MessagePasser), the
  new `sentMessageTimestamps` write, the new-only mappings, the removed/added/changed selectors,
  a relay to the messenger itself (unreachable through a valid identifier), and errors renamed
  with the contract-name prefix (revert data is compared with the selector mapped back to
  develop's; `gen-bytecodes.sh` generates the map from both sources).
- **Not compared**: events (neither engine can assert on logs; covered by a concrete differential
  fuzz instead) and gas (gas use and out-of-gas outcomes; NEW's extra SSTORE costs more).
- **Domain**: message lengths 0/4/37/100/128 (send) and 0/4/37/100 (relay); canonical relay
  payloads, plus arbitrary payloads of 0/100/128 bytes and 288-byte payloads with non-canonical
  `bytes` offsets and lengths; structured mock return data; one fixed relay caller.

## Engines and why both

- **hevm 0.58** (`prove_*`): `sendMessage` with the empty message, (O) and (S-all). hevm cannot
  handle develop's `sendMessage` with a non-empty message ("CopySlice with a symbolically sized
  region not currently implemented"), and its `relayMessage` run exceeds 16 GB.
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
| hevm `prove_sendMessage_len0` | 1 | PASS, 2 to 3 min |
| hevm non-vacuity | 3 | validated counterexamples |
| Halmos send (S-map, S-all), lengths 0/4/37/100/128 | 10 | PASS, 1 to 6 s each |
| Halmos relay (S-map, S-all), lengths 0/4/37/100 | 8 | PASS, 6 to 12 s each |
| Halmos relay, arbitrary and non-canonical payloads | 9 | PASS, 3 to 13 s each |
| Halmos non-vacuity | 6 | counterexamples, each replayed concretely |
| Mutants M1 to M6 (stray SSTORE in relay / send, wrong inbox argument, no ETH forwarded, relay does not set `successfulMessages[H]`, send does not set `sentMessages[nonce]`) | 6 | each caught by the check named for it (M5, M6: by S-map) |
| Known blind spot: M5, M6 under S-all (`check_allSlots_*`) | 2 | PASS, as expected (S-all does not see mapping entries) |
| Mutant M0 (unmutated source, same compile) | 4 | PASS |
| Concrete tests: event fuzz (4 tests) and witness replays (6 tests) | 10 | PASS |

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
- **Retarget to `448d31ad19`** (tip of `karl/message-expiry-refunds`): regenerated with
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
