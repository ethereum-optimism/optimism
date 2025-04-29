# The Bootstrap Commands

> Note: if you are joining an existing superchain, you can skip to the `init` and `apply` commands to create your L2 chain(s)

Bootstrap commands are used to deploy global singletons and implementation contracts for new superchains.
The deployed contract be then be use with future invocations of `apply` so that new L2 chains can join that superchain.
Most users won't need to use these commands, since `op-deployer apply` will automatically use predeployed contracts if they are available. However, you may need to use bootstrap commands if you're deploying chains to an L1 that isn't natively supported by `op-deployer`.

There are several bootstrap commands available, which you can view by running `op-deployer bootstrap --help`. We'll
focus on the most important ones, which should be run in the sequence listed below.

**It is safe to call these commands from a hot wallet.** None of the contracts deployed by these command are "ownable,"
so the deployment address has no further control over the system.

# 1. bootstrap superchain

```shell
op-deployer bootstrap superchain \
  --l1-rpc-url="<rpc url>" \
  --private-key="<contract deployer private key>" \
  --artifacts-locator="<locator>" \
  --outfile="./.deployer/bootstrap_superchain.json" \
  --superchain-proxy-admin-owner="<role address>" \
  --protocol-versions-owner="<role address>" \
  --guardian="<role address>"
```

### --required-protocol-version, --recommended-protocol-version (optional)
Defaults to `OPStackSupport` value read from `op-geth`, but can be overridden by these flags.

### --superchain-proxy-admin-owner, --protocol-versions-owner, --guardian
In a dev environment, these can all be hot wallet EOAs. In a production environment, `--guardian` should be an HSM (hardward security module) protected hot wallet and the other two should be multisig cold-wallets (e.g. Gnosis Safes).

### Output

This command will deploy several contracts, and output a JSON like the one below:

```json
{
  "proxyAdminAddress": "0x269b95a33f48a9055b82ce739b0c105a83edd64a",
  "superchainConfigImplAddress": "0x2f4c87818d67fc3c365ea10051b94f98893f3c64",
  "superchainConfigProxyAddress": "0xd0c74806fa114c0ec176c0bf2e1e84ff0a8f91a1",
  "protocolVersionsImplAddress": "0xbded9e39e497a34a522af74cf018ca9717c5897e",
  "protocolVersionsProxyAddress": "0x2e8e4b790044c1e7519caac687caffd4cafca2d4"
}
```

# 2. bootstrap implementations

```shell
op-deployer bootstrap implementations \
  --artifacts-locator="<locator>" \
  --l1-rpc-url="<rpc url>" \
  --outfile="./.deployer/bootstrap_implementations.json" \
  --mips-version="<1 or 2, for MIPS32 or MIPS64>" \
  --private-key="<contract deployer private key>" \
  --protocol-versions-proxy="<address output from bootstrap superchain>" \
  --superchain-config-proxy="<address output from bootstrap superchain>" \
  --upgrade-controller="<superchain-proxy-admin-owner used in bootstrap superchain>"
```

### Output

This command will deploy implementations, blueprints, and the OPCM. Deployments are (for the most part)
deterministic, so contracts will only be deployed once per chain as long as the implementation and constructor args
remain the same. This applies to the `op-deployer apply` pipeline - that is, if someone else ran `op-deployer boostrap implementations`
at some point on a given L1 chain, then the `apply` pipeline will re-use those implementations.

The command will output a JSON like the one below:

```json
{
  "Opcm": "0x4eeb114aaf812e21285e5b076030110e7e18fed9",
  "DelayedWETHImpl": "0x5e40b9231b86984b5150507046e354dbfbed3d9e",
  "OptimismPortalImpl": "0x2d7e764a0d9919e16983a46595cfa81fc34fa7cd",
  "PreimageOracleSingleton": "0x1fb8cdfc6831fc866ed9c51af8817da5c287add3",
  "MipsSingleton": "0xf027f4a985560fb13324e943edf55ad6f1d15dc1",
  "SystemConfigImpl": "0x760c48c62a85045a6b69f07f4a9f22868659cbcc",
  "L1CrossDomainMessengerImpl": "0x3ea6084748ed1b2a9b5d4426181f1ad8c93f6231",
  "L1ERC721BridgeImpl": "0x276d3730f219f7ec22274f7263180b8452b46d47",
  "L1StandardBridgeImpl": "0x78972e88ab8bbb517a36caea23b931bab58ad3c6",
  "OptimismMintableERC20FactoryImpl": "0x5493f4677a186f64805fe7317d6993ba4863988f",
  "DisputeGameFactoryImpl": "0x4bba758f006ef09402ef31724203f316ab74e4a0",
  "AnchorStateRegistryImpl": "0x7b465370bb7a333f99edd19599eb7fb1c2d3f8d2",
  "SuperchainConfigImpl": "0x4da82a327773965b8d4d85fa3db8249b387458e7",
  "ProtocolVersionsImpl": "0x37e15e4d6dffa9e5e320ee1ec036922e563cb76c"
}
```
