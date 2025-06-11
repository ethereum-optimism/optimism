# Prepare for Granite Breaking Changes

The Granite upgrade for Boba Eth Mainnet will be activated on **1729753200 Thu 24 Oct 07:00:00 UTC 2024**.

## For Node Operators

Node operators are required to upgrade to Granite before the activation date to avoid chain divergence. For Boba Eth Mainnet, the `op-node` release [v1.6.11](https://github.com/bobanetwork/boba/releases/tag/v1.6.11) and the `op-erigon` release [v1.2.5](https://github.com/bobanetwork/op-erigon/releases/tag/v1.2.5) contain these changes.

### Update to the Lastest Release

* op-node [v1.6.11](https://github.com/bobanetwork/boba/releases/tag/v1.6.11)
* op-erigon [v1.2.5](https://github.com/bobanetwork/op-erigon/releases/tag/v1.2.5)
* op-geth [v1.101408.0](https://github.com/ethereum-optimism/op-geth/releases/tag/v1.101408.0)

### Configure the Granite Activation Date

For `op-node` and `op-erigon`, the Granite activation date is included in the release, so no action is needed by the operators to configure the node.

For `op-geth`, the node operator is required to add `--override.granite=1729753200` to set up the Granite hardfork.

### Verify Your Configuration

Make the following checks to verify that your node is properly configured.

- `op-node` , `op-erigon` and `op-geth` will log their configurations at startup
- Check that the Granite time is set to `activation-timestamp` in the `op-node` startup logs
- Check that the Granite time is set to `activation-timestamp` in the `op-erigon` startup logs
- Check that the Granite time is set to `activation-timestamp` in the `op-geth` startup logs
