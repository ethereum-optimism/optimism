# The Bootstrap Commands

> Note: if you are joining an existing superchain, you can skip to the `init` and `apply` commands to create your L2 chain(s)

Bootstrap commands are used to deploy global singletons and implementation contracts for new superchains.
The deployed contracts can then be used with future invocations of `apply` so that new L2 chains can join that superchain.
Most users won't need to use these commands, since `op-deployer apply` will automatically use standard predeployed contracts for the L1/settlement-layer you are deploying on. However, you will need to use bootstrap commands if you're creating a new superchain.

There are several bootstrap commands available, which you can view by running `op-deployer bootstrap --help`. We'll
focus on the most important ones, which should be run in the sequence listed below.

**It is safe to call these commands from a hot wallet.** None of the contracts deployed by these command are "ownable,"
so the deployment address has no further control over the system.

# 1. bootstrap superchain

```shell
op-deployer bootstrap superchain \
  --l1-rpc-url="<rpc url>" \
  --private-key="<contract deployer private key>" \
  --outfile="./.deployer/bootstrap_superchain.json" \
  --superchain-proxy-admin-owner="<role address>" \
  --protocol-versions-owner="<role address>" \
  --guardian="<role address>"
```

### --required-protocol-version, --recommended-protocol-version (optional)
Defaults to `OPStackSupport` value read from `op-geth`, but can be overridden by these flags.

### --superchain-proxy-admin-owner, --protocol-versions-owner, --guardian
In a dev environment, these can all be hot wallet EOAs. In a production environment, `--guardian` should be an HSM (hardware security module) protected hot wallet and the other two should be multisig cold-wallets (e.g. Gnosis Safes).

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
  --l1-rpc-url="<rpc url>" \
  --outfile="./.deployer/bootstrap_implementations.json" \
  --private-key="<contract deployer private key>" \
  --protocol-versions-proxy="<contract address output from bootstrap superchain>" \
  --superchain-config-proxy="<contract address output from bootstrap superchain>" \
  --superchain-proxy-admin="<contract address from bootstrap superchain>" \
  --challenger="<role address for the superchain's challenger>" \
  --upgrade-controller="<superchain-proxy-admin-owner used in bootstrap superchain>"
```

### Output

This command will deploy implementations, blueprints, and the OPCM. Deployments are (for the most part)
deterministic, so contracts will only be deployed once per chain as long as the implementation and constructor args
remain the same. This applies to the `op-deployer apply` pipeline - that is, if someone else ran `op-deployer bootstrap implementations`
at some point on a given L1 chain, then the `apply` pipeline will re-use those implementations.

The command will output a JSON like the one below:

```json
{
  "opcmAddress": "0x82879934658738b6d5e8f781933ae7bbae05ba31",
  "opcmContractsContainerAddress": "0x1e8de1574a2e085b7a292c760d90cf982d3c1a11",
  "opcmGameTypeAdderAddress": "0xcab868d42d9088b86598a96d010db5819c19b847",
  "opcmDeployerAddress": "0xf8b6718b28fa36b430334e78adaf97174fed818c",
  "opcmUpgraderAddress": "0xa4d0a44890fafce541bdc4c1ca36fca1b5d22f56",
  "opcmInteropMigratorAddress": "0xf0fca53bb450dd2230c7eb58a39a5dbfc8492fb6",
  "opcmStandardValidatorAddress": "0x1364a02f64f03cd990f105058b8cc93a9a0ab2a1",
  "delayedWETHImplAddress": "0x570da3694c06a250aea4855b4adcd09505801f9a",
  "optimismPortalImplAddress": "0x1aa1d3fc9b39d7edd7ca69f54a35c66dcf1168f1",
  "ethLockboxImplAddress": "0xe6e51fa10d481002301534445612c61bae6b3258",
  "preimageOracleSingletonAddress": "0x1fb8cdfc6831fc866ed9c51af8817da5c287add3",
  "mipsSingletonAddress": "0x7a8456ba22df0cb303ae1c93d3cf68ea3a067006",
  "systemConfigImplAddress": "0x9f2b1fffd8a7aeef7aeeb002fd8477a4868e7e0a",
  "l1CrossDomainMessengerImplAddress": "0x085952eb0f0c3d1ca82061e20e0fe8203cdd630a",
  "l1ERC721BridgeImplAddress": "0xbafd2cae054ddf69af27517c6bea912de6b7eb8f",
  "l1StandardBridgeImplAddress": "0x6abaa7b42b9a947047c01f41b9bcb8684427bf24",
  "optimismMintableERC20FactoryImplAddress": "0xdd0b293b8789e9208481cee5a0c7e78f451d32bf",
  "disputeGameFactoryImplAddress": "0xe7ab0c07ee92aae31f213b23a132a155f5c2c7cc",
  "anchorStateRegistryImplAddress": "0xda4f46fad0e38d763c56da62c4bc1e9428624893",
  "superchainConfigImplAddress": "0xdaf60e3c5ef116810779719da88410cce847c2a4",
  "protocolVersionsImplAddress": "0xa95ac4790fedd68d9c3b30ed730afaec6029eb31"
}
```
