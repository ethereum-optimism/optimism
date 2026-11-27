# interop-safety-check

Independently checks the cross-safe chains that op-supernodes report. It does not trust
any supernode's decision. For each supernode, per chain, it:

- reads the supernode's cross-safe head and confirms the supernode's EL holds that block;
- reads every `CrossL2Inbox` `ExecutingMessage` log from `--from` up to that head;
- re-validates each message against the initiating log on the supernode's own ELs:
  - the source chain is in the dependency set;
  - the initiating timestamp is not after the executing block;
  - the initiating block's timestamp matches the identifier;
  - a log exists at that block and log index, from the identifier's origin, with the
    message's payload hash.

It also:

- checks that all supernodes report the same block at their lowest common cross-safe height;
- checks that `--gone-tx` transactions are not canonical, and that `--deposit-only` blocks
  hold only deposit transactions;
- reports a supernode whose cross-safe head has not advanced for `--stall-after`.

With `--interval`, later passes only check blocks that became cross-safe since the last pass.
If cross-safe moves back, the checker re-checks from the lower head.

## Example

Two chains, three supernodes, each supernode with its own EL per chain:

```sh
go run ./op-chain-ops/cmd/interop-safety-check \
  --chains 420120290 --chains 420120291 \
  --from 420120290:600 --from 420120291:300 \
  --supernode sn0=http://sn0:9545 --supernode sn1=http://sn1:9545 --supernode sn2=http://sn2:9545 \
  --el sn0:420120290=http://sn0-el-a:8545 --el sn0:420120291=http://sn0-el-b:8545 \
  --el sn1:420120290=http://sn1-el-a:8545 --el sn1:420120291=http://sn1-el-b:8545 \
  --el sn2:420120290=http://sn2-el-a:8545 --el sn2:420120291=http://sn2-el-b:8545 \
  --interval 30s --out safety.jsonl
```

A single pass (no `--interval`) exits non-zero if it finds any failure.
