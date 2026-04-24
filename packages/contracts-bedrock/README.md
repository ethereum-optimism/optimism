# OP Stack Smart Contracts

This package contains the L1 and L2 smart contracts for the OP Stack.

## Local Development

Tool versions are pinned in the monorepo `mise.toml`. From the monorepo root:

```bash
mise trust mise.toml
mise install
cd packages/contracts-bedrock
just install
```

Common commands:

```bash
just build
just test
just pr
```

`just pr` runs the local PR checks. Outside CI, it may format Solidity files before running checks.

For more information, check out the [book][book].

[book]: https://devdocs.optimism.io/contracts-bedrock
