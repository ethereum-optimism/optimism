# Withdrawal component proofs

`WithdrawalAuthorizationKontrol` checks inherited production methods using a small fixture.

| Proof | Scope |
| --- | --- |
| `prove_checkWithdrawal_equivalence` | Acceptance and rejection for symbolic authorization inputs. |
| `prove_deleteProvenWithdrawal_preservesOtherRecord` | Deletion eligibility and preservation of a distinct proof record, finalized state, and submitter counts. |

The fixture uses a seven-day maturity delay and `uint64` timestamps. Registry and game responses
are modeled. Setup helpers install proof records directly and are not protocol transitions.
These checks do not establish proof-record provenance, proxy behavior, or an invariant over
arbitrary protocol histories. Destination execution and gas availability are separate obligations.

## Running locally

Use the repository's pinned Foundry and Kontrol versions. Start with a clean `kprove` output
directory when limiting the compilation scope:

```sh
mise x -- just build-kontrol test/kontrol/proofs/WithdrawalAuthorization.k.sol
FOUNDRY_PROFILE=kprove kontrol build --no-forge-build --regen --rekompile
FOUNDRY_PROFILE=kprove kontrol prove \
  --match-test 'WithdrawalAuthorizationKontrol.prove_' \
  --reinit --workers 1 --max-depth 10000 --max-iterations 10000 \
  --smt-timeout 16000 --smt-retry-limit 1 --schedule CANCUN \
  --no-gas --xml-test-report --hide-status-bar --no-counterexample-information
```

This standalone invocation uses the fixture's `setUp` without a deployment state diff or additional
rewrite rules. `--no-gas` abstracts gas accounting. A proof stopped by a resource limit is incomplete.
Kontrol 1.0.90 reads its output directory from the selected Foundry TOML profile; keep compiled
artifacts in that directory rather than relying on `FOUNDRY_OUT` when invoking Kontrol.
