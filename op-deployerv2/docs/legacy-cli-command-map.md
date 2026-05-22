# Legacy CLI command map

This document maps the current `op-deployer` CLI surface to the proposed
`op-deployerv2` compatibility CLI.

Yes: compatibility mode should be literally the same CLI, with one new required
global parameter:

```bash
op-deployerv2 --contracts-source <repo-or-path>@<commit-or-version> <same-command> <same-flags>
```

The user-facing CLI does not need an explicit `adapter run` command. Internally,
core v2 still resolves `--contracts-source` and loads the commit-local
`op-deployer/v1` adapter from that source. The important boundary is that core
v2 may know the legacy command names for compatibility, but release-specific
intent/state/config translation lives in the selected contracts source.

For a drop-in binary named `op-deployer`, the only required caller change is:

```bash
op-deployer --contracts-source <repo-or-path>@<commit-or-version> <same-command> <same-flags>
```

## Global flags

Current global flags:

```text
--cache-dir
--log.level
--log.format
--log.color
--log.pid
```

V2 compatibility global flags:

```text
--contracts-source
--cache-dir
--log.level
--log.format
--log.color
--log.pid
```

`--contracts-source` is required for every compatibility command except help and
version. It selects the contracts commit/release that provides ABIs, generated
Go wrappers, and legacy adapters.

## `init`

Current:

```bash
op-deployer init \
  --l1-chain-id <uint64> \
  --l2-chain-ids <comma-separated-ids> \
  --workdir <dir> \
  --outdir <dir> \
  --intent-type <standard|custom|standard-overrides> \
  --intent-config-type <standard|custom|standard-overrides>
```

Aliases:

```text
--outdir              alias for --workdir
--intent-config-type  alias for --intent-type
```

V2 compatibility:

```bash
op-deployerv2 --contracts-source <source> init \
  --l1-chain-id <uint64> \
  --l2-chain-ids <comma-separated-ids> \
  --workdir <dir> \
  --intent-type <standard|custom|standard-overrides>
```

## `apply`

Current:

```bash
op-deployer apply \
  --l1-rpc-url <url> \
  --workdir <dir> \
  --outdir <dir> \
  --private-key <hex> \
  --deployment-target <live|genesis|calldata|noop> \
  --op-program-svc-url <url> \
  --verify \
  --verifier-api-key <key> \
  --etherscan-api-key <key> \
  --verifier <etherscan|blockscout|custom[,..]> \
  --verifier-url <url> \
  --use-forge \
  --validate <auto|version>
```

Aliases:

```text
--outdir            alias for --workdir
--etherscan-api-key deprecated alias for --verifier-api-key
```

V2 compatibility:

```bash
op-deployerv2 --contracts-source <source> apply \
  --l1-rpc-url <url> \
  --workdir <dir> \
  --private-key <hex> \
  --deployment-target <live|genesis|calldata|noop> \
  --op-program-svc-url <url> \
  --verify \
  --verifier-api-key <key> \
  --verifier <etherscan|blockscout|custom[,..]> \
  --verifier-url <url> \
  --use-forge \
  --validate <auto|version>
```

The selected adapter reads legacy `intent.toml` and `state.json`, translates
them to generated inputs for the selected contracts source, executes the
generated script wrappers, and writes legacy-compatible state.

## `upgrade`

Current parent:

```bash
op-deployer upgrade \
  --l1-rpc-url <url> \
  --private-key <hex> \
  --deployment-target <live|genesis|calldata|noop> \
  <subcommand>
```

Current subcommands:

```text
v2.0.0
v3.0.0
v4.0.0
v4.1.0
v5.0.0
v6.0.0-rc.2
```

Every current upgrade subcommand has the same flags:

```bash
op-deployer upgrade <version> \
  --l1-rpc-url <url> \
  --config <path> \
  --override-artifacts-url <url> \
  --outfile <path|->
```

V2 compatibility:

```bash
op-deployerv2 --contracts-source <source> upgrade <version> \
  --l1-rpc-url <url> \
  --config <path> \
  --override-artifacts-url <url> \
  --outfile <path|->
```

The old version-named commands can remain for compatibility. The actual ABI,
input type, and output type come from `--contracts-source`.

## `bootstrap implementations`

Current:

```bash
op-deployer bootstrap implementations \
  --l1-rpc-url <url> \
  --private-key <hex> \
  --outfile <path|-> \
  --artifacts-locator <locator> \
  --mips-version <uint64> \
  --dev-feature-bitmap <hash> \
  --withdrawal-delay-seconds <uint64> \
  --min-proposal-size-bytes <uint64> \
  --challenge-period-seconds <uint64> \
  --proof-maturity-delay-seconds <uint64> \
  --dispute-game-finality-delay-seconds <uint64> \
  --dispute-max-game-depth <uint64> \
  --dispute-split-depth <uint64> \
  --dispute-clock-extension <uint64> \
  --dispute-max-clock-duration <uint64> \
  --superchain-config-proxy <address> \
  --l1-proxy-admin-owner <address> \
  --upgrade-controller <address> \
  --superchain-proxy-admin <address> \
  --challenger <address> \
  --verify \
  --verifier <etherscan|blockscout|custom[,..]> \
  --verifier-url <url> \
  --verifier-api-key <key> \
  --etherscan-api-key <key> \
  --use-forge
```

Aliases:

```text
--upgrade-controller alias for --l1-proxy-admin-owner
--etherscan-api-key  deprecated alias for --verifier-api-key
```

V2 compatibility:

```bash
op-deployerv2 --contracts-source <source> bootstrap implementations \
  --l1-rpc-url <url> \
  --private-key <hex> \
  --outfile <path|-> \
  --artifacts-locator <locator> \
  --mips-version <uint64> \
  --dev-feature-bitmap <hash> \
  --withdrawal-delay-seconds <uint64> \
  --min-proposal-size-bytes <uint64> \
  --challenge-period-seconds <uint64> \
  --proof-maturity-delay-seconds <uint64> \
  --dispute-game-finality-delay-seconds <uint64> \
  --dispute-max-game-depth <uint64> \
  --dispute-split-depth <uint64> \
  --dispute-clock-extension <uint64> \
  --dispute-max-clock-duration <uint64> \
  --superchain-config-proxy <address> \
  --l1-proxy-admin-owner <address> \
  --superchain-proxy-admin <address> \
  --challenger <address> \
  --verify \
  --verifier <etherscan|blockscout|custom[,..]> \
  --verifier-url <url> \
  --verifier-api-key <key> \
  --use-forge
```

## `bootstrap superchain`

Current:

```bash
op-deployer bootstrap superchain \
  --l1-rpc-url <url> \
  --private-key <hex> \
  --outfile <path|-> \
  --artifacts-locator <locator> \
  --superchain-proxy-admin-owner <address> \
  --guardian <address> \
  --paused \
  --verify \
  --verifier <etherscan|blockscout|custom[,..]> \
  --verifier-url <url> \
  --verifier-api-key <key> \
  --etherscan-api-key <key> \
  --use-forge
```

Alias:

```text
--etherscan-api-key deprecated alias for --verifier-api-key
```

V2 compatibility:

```bash
op-deployerv2 --contracts-source <source> bootstrap superchain \
  --l1-rpc-url <url> \
  --private-key <hex> \
  --outfile <path|-> \
  --artifacts-locator <locator> \
  --superchain-proxy-admin-owner <address> \
  --guardian <address> \
  --paused \
  --verify \
  --verifier <etherscan|blockscout|custom[,..]> \
  --verifier-url <url> \
  --verifier-api-key <key> \
  --use-forge
```

## `inspect`

Current subcommands:

```text
l1
genesis
rollup
deploy-config
l2-semvers
```

Each current inspect subcommand has the same positional argument and flags:

```bash
op-deployer inspect <l1|genesis|rollup|deploy-config|l2-semvers> \
  --workdir <dir> \
  --outdir <dir> \
  --outfile <path|-> \
  <l2-chain-id>
```

Alias:

```text
--outdir alias for --workdir
```

V2 compatibility:

```bash
op-deployerv2 --contracts-source <source> inspect <l1|genesis|rollup|deploy-config|l2-semvers> \
  --workdir <dir> \
  --outfile <path|-> \
  <l2-chain-id>
```

## `clean cache`

Current:

```bash
op-deployer \
  --cache-dir <dir> \
  clean cache
```

V2 compatibility:

```bash
op-deployerv2 \
  --contracts-source <source> \
  --cache-dir <dir> \
  clean cache
```

This command can be implemented directly by core v2 because cache cleanup does
not require release-specific contract translation.

## `verify`

Current:

```bash
op-deployer verify \
  --l1-rpc-url <url> \
  --artifacts-locator <locator> \
  --verifier-api-key <key> \
  --etherscan-api-key <key> \
  --input-file <path> \
  --contract-name <name> \
  --verifier <etherscan|blockscout|custom[,..]> \
  --verifier-url <url>
```

Alias:

```text
--etherscan-api-key deprecated alias for --verifier-api-key
```

V2 compatibility:

```bash
op-deployerv2 --contracts-source <source> verify \
  --l1-rpc-url <url> \
  --artifacts-locator <locator> \
  --verifier-api-key <key> \
  --input-file <path> \
  --contract-name <name> \
  --verifier <etherscan|blockscout|custom[,..]> \
  --verifier-url <url>
```

The adapter is needed when `--input-file` is in a legacy output shape. Pure
artifact verification can also be a core v2 capability.

## `manage add-game-type-v2`

Current:

```bash
op-deployer manage add-game-type-v2 \
  --l1-rpc-url <url> \
  --config <path> \
  --override-artifacts-url <url> \
  --outfile <path|-> \
  --cache-dir <dir>
```

V2 compatibility:

```bash
op-deployerv2 --contracts-source <source> manage add-game-type-v2 \
  --l1-rpc-url <url> \
  --config <path> \
  --override-artifacts-url <url> \
  --outfile <path|-> \
  --cache-dir <dir>
```

The adapter maps the legacy game-type config to the selected generated OPCM
upgrade input.

## `manage migrate`

Current:

```bash
op-deployer manage migrate \
  --cache-dir <dir> \
  --l1-rpc-url <url> \
  --private-key <hex> \
  --artifacts-locator <locator> \
  --l1-proxy-admin-owner-address <address> \
  --opcm-impl-address <address> \
  --starting-anchor-root <hash> \
  --starting-anchor-l2-sequence-number <uint64> \
  --initial-bond <wei-string> \
  --system-config-proxy-address <address> \
  --starting-respected-game-type <uint64> \
  --dispute-game-enabled \
  --dispute-game-type <uint64> \
  --dispute-absolute-prestate <hash>
```

V2 compatibility:

```bash
op-deployerv2 --contracts-source <source> manage migrate \
  --cache-dir <dir> \
  --l1-rpc-url <url> \
  --private-key <hex> \
  --artifacts-locator <locator> \
  --l1-proxy-admin-owner-address <address> \
  --opcm-impl-address <address> \
  --starting-anchor-root <hash> \
  --starting-anchor-l2-sequence-number <uint64> \
  --initial-bond <wei-string> \
  --system-config-proxy-address <address> \
  --starting-respected-game-type <uint64> \
  --dispute-game-enabled \
  --dispute-game-type <uint64> \
  --dispute-absolute-prestate <hash>
```

This is adapter-owned because it is migration semantics, not generic ABI
execution.

## `validate auto`

Current:

```bash
op-deployer validate auto \
  --l1-rpc-url <url> \
  --workdir <dir> \
  --fail \
  [chain-id]
```

V2 compatibility:

```bash
op-deployerv2 --contracts-source <source> validate auto \
  --l1-rpc-url <url> \
  --workdir <dir> \
  --fail \
  [chain-id]
```

The adapter decides whether validation should read legacy `state.json`, a v2
state shape, or both.
