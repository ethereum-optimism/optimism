# Governance Modes

Starting with Upgrade 16, OP Deployer supports two governance modes for OP chains:

## 1. Optimism Governed Mode

This is the standard mode for chains that are part of the Optimism ecosystem and governed by the Optimism Collective. In this mode:

- The chain uses a shared SuperchainConfig, governed by Optimism
- L1 Proxy Admin Ownership (L1PAO) is delegated to the Superchain multisig
- The chain is upgradeable by the Optimism governance process

To use this mode, select the `standard-governed` intent type when initializing your deployment:

```shell
op-deployer init \
  --l1-chain-id <chain ID of your L1> \
  --l2-chain-ids <comma separated list of chain IDs for your L2s> \
  --outdir <directory to write the intent and state files> \
  --intent-type standard-governed
```

## 2. Independent Governance Mode

This mode is for chains that want to be technically compatible with the OP Stack but governed independently. In this mode:

- The chain uses a chain-specific SuperchainConfig
- L1PAO belongs to the chain's own governance structure
- The chain is not governed by the Optimism Collective

To use this mode, select the `standard-non-governed` intent type when initializing your deployment:

```shell
op-deployer init \
  --l1-chain-id <chain ID of your L1> \
  --l2-chain-ids <comma separated list of chain IDs for your L2s> \
  --outdir <directory to write the intent and state files> \
  --intent-type standard-non-governed
```

After initialization, you'll need to edit the `intent.toml` file to specify your own SuperchainConfigProxy address.

## Which Mode Should I Choose?

- **Optimism Governed Mode**: Choose this if you want your chain to be part of the Optimism ecosystem and benefit from the governance and upgrades provided by the Optimism Collective.

- **Independent Governance Mode**: Choose this if you want to maintain full control over your chain's governance and upgrade process.

## Changing Governance Modes

It's important to decide on a governance mode before deployment, as changing between modes after deployment requires complex governance operations. If you are unsure which mode to choose, consult with the Optimism team before proceeding with deployment.
