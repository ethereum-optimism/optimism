# eth-liquidity-pause

Inspects and drives the ETHLiquidity pause.

The L2 `ETHLiquidity` predeploy (`0x4200000000000000000000000000000000000025`) mirrors the
chain's L1 pause: the SuperchainConfig pause of the chain's `ETHLockbox` identifier, or the
global pause. While `ETHLiquidity.paused()` is true, `burn()` and `mint()` revert with
`ETHLiquidity_Paused()`. As a result, `SuperchainETHBridge`
(`0x4200000000000000000000000000000000000024`) `sendETH`, `relayETH` and `refundETH` also
revert.

The L1 pause reaches L2 through the permissionless `L1CrossDomainMessenger.syncETHLiquidityPause()`.
That call reads the L1 pause and sends `ETHLiquidity.setPaused(paused, l1Timestamp)` to L2 as
a cross-domain message with a 100k gas limit. Anyone can call it, so `sync` only mirrors state
and needs no privileged key. L2 ignores a state read earlier than the one it already applied.

Production pauses stay with the guardian's existing DeputyPauseModule flow. This tool reads
production state (`status`) and can mirror it (`sync`). It never pauses production: the
`devnet-*` helpers refuse OP Mainnet, Ink, Unichain, their Sepolia chains and an L1 of
Ethereum mainnet.

## Commands

### status

For each `--l2`, `status` reads:

- the chain ID;
- `ETHLiquidity.paused()` and `pausedL1Timestamp()` (the L1 timestamp of the last applied
  sync);
- whether an `eth_call` of `SuperchainETHBridge.sendETH(from, destination)` with 1 wei
  reverts, and with which error.

The call uses a state override to fund `--from`, so the default dummy sender works on any
chain. The destination defaults to the chain's own ID + 1.

With `--l1` and one `--l1-messenger` per `--l2` (in the same order), `status` also reads the L1
side. It reads `L1CrossDomainMessenger.paused()`, which is the ETHLockbox pause. It flags any
chain where the two sides differ, for example:

```
chain 901 (http://localhost:9545)
  L2 ETHLiquidity.paused(): false (never synced)
  sendETH eth_call:         succeeds
  L1 paused():              true (ETHLockbox 0x..., SuperchainConfig 0x...)
  OUT OF SYNC: L1 paused=true but L2 paused=false; run `eth-liquidity-pause sync`
```

`--expect paused` exits non-zero unless every chain meets all of these:

- `paused()` is true;
- `sendETH` reverts with `ETHLiquidity_Paused()`;
- if read, L1 is paused.

`--expect unpaused` is the reverse. It also fails when `paused()` itself reverts, which is
what a pre-pause ETHLiquidity does. `--json` prints machine-readable output.

```bash
# Devnet gate interop.eth-liquidity-pause, or production verification.
go run ./op-chain-ops/cmd/eth-liquidity-pause status \
  --l2 http://localhost:9545 --l2 http://localhost:9546 \
  --l1 http://localhost:8545 --l1-messenger 0x... --l1-messenger 0x... \
  --expect paused
```

### sync

`sync` mirrors the L1 pause to L2 on each chain. For each chain it:

1. checks that the messenger's `SystemConfig.l2ChainId()` matches the paired `--l2`;
2. prints both sides;
3. calls `syncETHLiquidityPause()` from `--key` (any funded L1 account) on every chain whose
   L2 differs from L1;
4. waits for the L1 receipt;
5. polls L2 until `paused()` equals the L1 pause as of the sync's block, or `--timeout`
   passes.

Chains that are already in sync are skipped unless you pass `--force`. A timeout usually means
that L2 has not derived the L1 block yet. It can also mean that the message failed on L2 and
is waiting in the L2CrossDomainMessenger's failed messages. In that case, run `sync` again.

```bash
go run ./op-chain-ops/cmd/eth-liquidity-pause sync \
  --l1 http://localhost:8545 \
  --l2 http://localhost:9545 --l1-messenger 0x... \
  --l2 http://localhost:9546 --l1-messenger 0x... \
  --key <funded-l1-key>
```

### devnet-pause / devnet-unpause

These commands are for devnets only. For each chain they resolve the pause identifier:
`L1CrossDomainMessenger` → `SystemConfig` → `OptimismPortal` → `ETHLockbox`, and the ETHLockbox
address is the identifier. Chains in one interop set share an ETHLockbox, so each identifier is
handled once.

The command checks that `--guardian-key` is the SuperchainConfig guardian. It then calls:

- `pause(lockbox)`, or `extend(lockbox)` if an earlier pause has expired;
- or `unpause(lockbox)`.

Then it runs `sync` from `--key`, or from the guardian key if `--key` is not set.
`devnet-unpause` warns when a global pause (`address(0)`) is active, because L1 stays paused
until that pause is cleared.

```bash
go run ./op-chain-ops/cmd/eth-liquidity-pause devnet-pause --config devnet.toml
go run ./op-chain-ops/cmd/eth-liquidity-pause status --config devnet.toml --expect paused
go run ./op-chain-ops/cmd/eth-liquidity-pause devnet-unpause --config devnet.toml
go run ./op-chain-ops/cmd/eth-liquidity-pause status --config devnet.toml --expect unpaused
```

## Config

Copy `config.example.toml`, fill it in, and pass it with `--config`. Keys are the flag names,
and repeatable flags are TOML arrays. Your copy may hold private keys, so every `*.toml` in this
directory except the example is gitignored. CLI flags and `ETH_LIQUIDITY_PAUSE_*` env vars
override the file.
