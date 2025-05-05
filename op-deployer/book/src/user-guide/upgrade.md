# The Upgrade Command

The `upgrade` command allows you to upgrade a chain from one version to another. It consists of several subcommands, one
for each upgrade version. Think of it like a database migration: each upgrade command upgrades a chain from exactly
one previous version to the next. A chain that is several versions behind can be upgrade to the latest version by
running multiple `upgrade` commands in sequence.

Unlike the `bootstrap` or `apply` commands, `upgrade` does not directly interact with the chain. Instead, it generates
calldata. You can then use this calldate with `cast`, Gnosis SAFE, [`superchain-ops`][superchain-ops], or whatever
tooling you use to manage your L1.

Your chain **must** be owned by a smart contract for `upgrade` to work because the OPCM can only be called via
`DELEGATECALL`.

[superchain-ops]: https://github.com/ethereum-optimism/superchain-ops

## Usage

Use the `upgrade` command like this:

```shell
op-deployer upgrade <version> \
  --config <path to config JSON>
```

Version `version` must be one of `v2.0.0` or `v3.0.0`.

You can also provide an optional `--override-artifacts-url` flag if you want to point to a different set of artifacts
from the default. Setting this flag is not recommended.

The config file should look like this:

```json
{
  "prank": "<address of the contract that owns the chain>",
  "opcm": "<address of the chain's OPCM>",
  "chainConfigs": [
    {
      "systemConfigProxy": "<address of the chain's system config proxy>",
      "proxyAdmin": "<address of the chain's proxy admin>",
      "absolutePrestate": "<32-byte hash of the chain's absolute prestate>"
    }
  ]
}
```

You can specify multiple chains in the `chainConfigs` array. The CLI will generate calldata for all chains you
specify. Note that all of these chains must be upgradeable using the provided OPCM and `prank` address.

The `upgrade` command will provide the following output:

```json
{
  "to": "<maps to the prank address>",
  "data": "<calldata>",
  "value": "0x0"
}
```

You can then use the `data` field in the `to` field as input to `cast` or other tools.