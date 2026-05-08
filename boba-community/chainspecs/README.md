# Boba Chain Specs for op-reth

These JSON files are op-reth chain specifications for Boba networks, in the format consumed by `op-reth --chain=<path>`.

| File | Chain ID | Network |
|---|---|---|
| `boba.json` | 288 | Boba Mainnet |
| `boba-sepolia.json` | 28882 | Boba Sepolia Testnet |

Each file is a complete `Genesis` JSON containing:
- Chain config: `chainId`, all hardfork activation times (canyon/delta/ecotone/fjord/granite/holocene), and the OP Stack `optimism` config (EIP-1559 params)
- OVM-era genesis block 0 metadata (`extraData`, `gasLimit`, etc.)

## Why these files exist

`op-reth` was originally built and released by Paradigm at `ghcr.io/paradigmxyz/op-reth`, which embedded all superchain-registry chains (including Boba) at compile time. Starting with op-reth v1.11.0, ownership of op-reth moved to OP Labs (`ethereum-optimism/op-reth`), and the upstream image now only embeds OP Mainnet, OP Sepolia, Base Mainnet, Base Sepolia, and a dev chain — Boba is no longer built in.

To run op-reth on Boba with the OP Labs image, the chain spec must be supplied at runtime via `--chain=<path-to-json>`.

## Regenerating these files

The files in this directory were produced by extracting the in-memory chain spec from the last paradigmxyz build that included Boba (`ghcr.io/paradigmxyz/op-reth:v1.10.2`). They should not need to change unless Boba activates a new hardfork.

If you do need to regenerate them (e.g. after a hardfork is added to the superchain-registry), run:

```bash
docker run --rm ghcr.io/paradigmxyz/op-reth:v1.10.2 dump-genesis --chain boba 2>/dev/null \
  | tail -n +2 > boba.json
docker run --rm ghcr.io/paradigmxyz/op-reth:v1.10.2 dump-genesis --chain boba-sepolia 2>/dev/null \
  | tail -n +2 > boba-sepolia.json
```

(`tail -n +2` strips the leading log line that `op-reth` writes to stdout before the JSON.)

Note: this approach is frozen at v1.10.2 of paradigm's chain spec definitions. For new hardforks added after v1.10.2, the chain spec JSON will need to be edited manually or extracted from a newer source.

## Verifying

To confirm a chain spec produces an identical OpChainSpec to the paradigm built-in:

```bash
docker run --rm ghcr.io/paradigmxyz/op-reth:v1.10.2 dump-genesis --chain boba | tail -n +2 > /tmp/a.json
docker run --rm -v "$PWD:/cs" us-docker.pkg.dev/oplabs-tools-artifacts/images/op-reth:v2.2.1 \
  dump-genesis --chain /cs/boba.json | tail -n +2 > /tmp/b.json
diff /tmp/a.json /tmp/b.json && echo "OK"
```
