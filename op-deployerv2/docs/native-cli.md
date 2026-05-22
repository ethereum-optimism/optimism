# Native v2 CLI

This document describes the proposed native `op-deployerv2` CLI. It is separate
from the legacy compatibility CLI.

Native v2 commands should be source-backed and generated-input driven. They
should not expose release-specific OPCM or script fields as hardcoded CLI flags.

## Global shape

Every source-backed native command takes the same required global source:

```bash
op-deployerv2 --contracts-source <repo-or-path>@<commit-or-version> <command>
```

`--contracts-source` selects the contracts source that provides:

- ABIs
- generated Go packages
- script manifests
- legacy adapters, when compatibility commands are used

Core v2 resolves the source. The selected source defines the concrete OPCM ABI,
script input schemas, script output schemas, and adapters.

## Native commands

The proposed native command set is intentionally small:

```text
source
template
upgrade
bootstrap
script
gen
verify
clean
```

Legacy commands like `init`, `apply`, `inspect`, `manage`, and version-named
`upgrade vX.Y.Z` commands are compatibility commands, not native v2 commands.

## `source`

Inspect what a selected contracts source provides.

```bash
op-deployerv2 --contracts-source <source> source info
```

Example output should include the resolved commit, artifact set, default OPCMv2
artifact, known script IDs, generated packages, and available adapters.

```bash
op-deployerv2 --contracts-source <source> source list
```

`source list` should list generated callable surfaces from the selected source,
for example:

```text
opcm.upgrade
script.DeploySuperchain
script.DeployImplementations
adapter.op-deployer/v1
```

## `template`

Generate an input file scaffold from the selected ABI or script schema.

Upgrade template:

```bash
op-deployerv2 --contracts-source <source> template upgrade \
  --out upgrade.yml
```

Script template:

```bash
op-deployerv2 --contracts-source <source> template script \
  --script <script-id> \
  --out input.yml
```

Templates are generated from the selected source. If a future OPCM adds a field,
the generated template changes because the source changed, not because core
`op-deployerv2` changed.

## `upgrade`

Plan or broadcast an OPCMv2 upgrade.

```bash
op-deployerv2 --contracts-source <source> upgrade \
  --network <network-or-chain-id> \
  --l1-rpc-url <url> \
  --opcm <address> \
  --executor <address> \
  --input upgrade.yml \
  --out upgrade-plan.json
```

Broadcasting is an execution option, not a separate versioned command:

```bash
op-deployerv2 --contracts-source <source> upgrade \
  --network <network-or-chain-id> \
  --l1-rpc-url <url> \
  --opcm <address> \
  --executor <address> \
  --input upgrade.yml \
  --broadcast \
  --private-key <hex>
```

Inline typed overrides are allowed:

```bash
op-deployerv2 --contracts-source <source> upgrade \
  --network <network-or-chain-id> \
  --l1-rpc-url <url> \
  --opcm <address> \
  --executor <address> \
  --input upgrade.yml \
  --input.disputeGameConfigs='[
    {
      "enabled": true,
      "gameType": 0,
      "initBond": "0",
      "gameArgs": "0x..."
    }
  ]' \
  --input.extraInstructions='[]' \
  --out upgrade-plan.json
```

Stable `upgrade` flags are execution concerns:

```text
--network
--l1-rpc-url
--opcm
--executor
--input
--input.*
--out
--broadcast
--private-key
```

OPCM input fields are not stable first-class flags. They come from the selected
OPCM ABI and are supplied through `--input` or `--input.*`.

## `bootstrap`

Run a bootstrap script from the selected contracts source.

```bash
op-deployerv2 --contracts-source <source> bootstrap \
  --script <script-id> \
  --network <network-or-chain-id> \
  --l1-rpc-url <url> \
  --input bootstrap.yml \
  --out bootstrap-output.json
```

Broadcasting uses the same execution flags:

```bash
op-deployerv2 --contracts-source <source> bootstrap \
  --script <script-id> \
  --network <network-or-chain-id> \
  --l1-rpc-url <url> \
  --input bootstrap.yml \
  --broadcast \
  --private-key <hex>
```

The script ID comes from the selected contracts source, for example
`DeploySuperchain` or `DeployImplementations`. Core v2 should not hardcode those
script input structs.

## `script`

Run any generated script surface from the selected contracts source. This is the
generic form that `bootstrap` can use internally.

```bash
op-deployerv2 --contracts-source <source> script run \
  --script <script-id> \
  --network <network-or-chain-id> \
  --l1-rpc-url <url> \
  --input input.yml \
  --out output.json
```

Inline typed overrides use the same convention:

```bash
op-deployerv2 --contracts-source <source> script run \
  --script <script-id> \
  --input input.yml \
  --input.owner=0x... \
  --out output.json
```

Use `script run` for source-provided scripts that are not part of the user-level
bootstrap workflow.

## `gen`

Generate checked-in Go packages from source ABIs and scripts.

Generate upgrade bindings:

```bash
op-deployerv2 --contracts-source <source> gen go \
  --surface opcm.upgrade \
  --package opcmupgrade \
  --out ./internal/generated/opcmupgrade
```

Generate script bindings:

```bash
op-deployerv2 --contracts-source <source> gen go \
  --surface script.DeployImplementations \
  --package deployimplementations \
  --out ./internal/generated/deployimplementations
```

CI freshness check:

```bash
op-deployerv2 --contracts-source <source> gen check \
  --config op-deployerv2.gen.yaml
```

Programmatic callers use these generated packages. They should not handwrite
release-specific structs.

## `verify`

Verify contracts using artifacts from the selected source.

```bash
op-deployerv2 --contracts-source <source> verify \
  --network <network-or-chain-id> \
  --l1-rpc-url <url> \
  --input output.json \
  --verifier <etherscan|blockscout|custom[,..]> \
  --verifier-url <url> \
  --verifier-api-key <key>
```

If `--input` is a legacy output shape, this should route through the selected
legacy adapter. If it is a native generated output shape, core v2 can verify it
directly.

## `clean`

Clean local v2 cache data.

```bash
op-deployerv2 clean cache \
  --cache-dir <dir>
```

This command does not need a contracts source.

## Non-native compatibility commands

These commands can still exist for drop-in compatibility, but they should be
adapter-backed:

```text
init
apply
inspect
manage
validate auto
upgrade v2.0.0
upgrade v3.0.0
upgrade v4.0.0
upgrade v4.1.0
upgrade v5.0.0
upgrade v6.0.0-rc.2
```

Compatibility usage:

```bash
op-deployerv2 --contracts-source <source> apply --workdir .deployer
```

Native usage:

```bash
op-deployerv2 --contracts-source <source> upgrade \
  --opcm <address> \
  --executor <address> \
  --input upgrade.yml
```
