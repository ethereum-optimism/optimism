# Prepare for Fjord Breaking Changes

The Fjord upgrade for Boba Sepolia was activated on **1722297600 Tue July 30 00:00:00 UTC 2024**. The Fjord Boba Mainnet upgrade will be optimistically activated **1725951600 Tue Sep 10 7:00:00 UTC 2024**.

## For Node Operators

Node operators are required to upgrade to Fjord before the activation date to avoid chain divergence. For Boba Mainnet, the `op-node` release [v1.6.8](https://github.com/bobanetwork/boba/releases/tag/v1.6.8) and the `op-erigon` release [v1.2.0](https://github.com/bobanetwork/op-erigon/releases/tag/v1.2.0) contain these changes.

### Update to the Lastest Release

* op-node [v1.6.8](https://github.com/bobanetwork/boba/releases/tag/v1.6.8)
* op-erigon [v1.2.0](https://github.com/bobanetwork/op-erigon/releases/tag/v1.2.0)
* op-geth [v1.101315.2](https://github.com/ethereum-optimism/op-geth/releases/tag/v1.101315.2)

### Configure the Fjord Activation Date

For `op-node` and `op-erigon`, the Fjord activation date is included in the release, so no action is needed by the operators to configure the node.

For `op-geth`, the node operator is required to add `--override.fjord=1725951600` to set up the Fjord hardfork.

### Verify Your Configuration

Make the following checks to verify that your node is properly configured.

- `op-node` , `op-erigon` and `op-geth` will log their configurations at startup
- Check that the Fjord time is set to `activation-timestamp` in the `op-node` startup logs
- Check that the Fjord time is set to `activation-timestamp` in the `op-erigon` startup logs
- Check that the Fjord time is set to `activation-timestamp` in the `op-geth` startup logs
