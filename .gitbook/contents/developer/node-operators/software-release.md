# Node Software Releases

This page provides a list of the necessary versions of node software and instructions on how to keep them updated.

Our latest releases, notes and changelogs can be found on Github. `op-node` releases can be found [here](https://github.com/bobanetwork/v3-anchorage/tags) and `op-erigon` release can be found [here](https://github.com/bobanetwork/v3-erigon/releases).

## Required Version by Network

These are the minimal required versions for the `op-node`, `op-erigon` and `op-geth` by network.

| Network      | op-node                                                      | op-erigon                                                    | op-geth                                                      |
| ------------ | ------------------------------------------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------ |
| Boba Sepolia | [v1.6.1](https://github.com/bobanetwork/v3-anchorage/releases/tag/op-node%2Fv1.6.1) | [v1.1.1](https://github.com/bobanetwork/v3-erigon/releases/tag/v1.1.1) | [v1.101308.1](https://github.com/ethereum-optimism/op-geth/releases/tag/v1.101308.1) |
| Op Sepolia   | [v1.6.0](https://github.com/bobanetwork/v3-anchorage/releases/tag/v1.6.0) | [v1.1.1](https://github.com/bobanetwork/v3-erigon/releases/tag/v1.1.1) | [v1.101308.1](https://github.com/ethereum-optimism/op-geth/releases/tag/v1.101308.1) |
| Op Mainnet   | [v1.6.0](https://github.com/bobanetwork/v3-anchorage/releases/tag/v1.6.0) | [v1.1.1](https://github.com/bobanetwork/v3-erigon/releases/tag/v1.1.1) | [v1.101308.1](https://github.com/ethereum-optimism/op-geth/releases/tag/v1.101308.1) |

## [op-erigon v1.1.1](https://github.com/bobanetwork/v3-erigon/releases/tag/v1.1.1)

**Description**

This is a mandatory release for node operators on all networks. It introduces the minimum transaction priority fee and fixes the deposit transaction issue where the `msg.value` and the deposit amount differ.

**Required Action**

Upgrade your `op-erigon` software.

## [op-node v1.6.1](https://github.com/bobanetwork/v3-anchorage/releases/tag/v1.6.1)

**Description**

This is a mandatory release for node operators on Boba Sepolia networks. The **Ecotone** and **Delta** protocol upgrades will activate on Wed Feb 28 2024 00:00:00 UTC 2024 on Sepolia Boba Chains.

**Required Action**

Upgrade your `op-node` software.

**Suggested action**

Explicitly specify the Beacon endpoint: `--l1.beacon` and `$OP_NODE_L1_BEACON`

## [op-node v1.6.0](https://github.com/bobanetwork/v3-anchorage/releases/tag/v1.6.0)

**Description**

This is a mandatory release for node operators on Op networks. It supports the Ecotone hardfork.

**Required Action**

Upgrade your `op-node` software.

**Suggested action**

Explicitly specify the Beacon endpoint: `--l1.beacon` and `$OP_NODE_L1_BEACON`

## [op-geth v1.101308.1](https://github.com/ethereum-optimism/op-geth/releases/tag/v1.101308.1)

**Description**

This is a mandatory release for node operators on Op networks.

**Required Action**

* Upgrade your `op-geth` software.
