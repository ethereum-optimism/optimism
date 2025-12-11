# Preparing for Fusaka breaking changes

The Fusaka upgrade on Ethereum is currently scheduled and requires mandatory upgrades for node operators.

* Holesky activation: October 1st
* Sepolia activation: October 14th
* Mainnet activation: December 3rd

## For Node Operators

Node operators will need to upgrade to the respective releases before the activation dates. These following steps are necessary for every node operator:

*Update:*  There is a recommended upgrade for the op-node and op-geth components.  The previously specified versions from this notice should still be Fusaka compatible, but, newer versions are recommended.

* op-node at [v1.16.2](https://github.com/bobanetwork/boba/releases/tag/op-node/v1.16.2-boba) (formnerly at [v1.14.1](https://github.com/bobanetwork/boba/releases/tag/op-node/v1.14.1))
* op-geth at [v1.101603.5](https://github.com/bobanetwork/op-geth/releases/tag/v1.101603.5) (formerly at [v1.101603.1](https://github.com/bobanetwork/op-geth/releases/tag/v1.101603.1))
* op-erigon currently does not have Fusaka support, operators are advised to migrate to op-geth

:::note
The op-geth database layout has changed since the last required update.  You may experience elevated CPU usage while first deploying the new op-geth images, but the CPU should return to normal after a few hours.
:::
