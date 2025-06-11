# Preparing for Pectra breaking changes

The Pectra upgrade on Ethereum is set to be optimistically activated on

* Holesky slot: 1740434112 (Mon, Feb 24 at 21:55:12 UTC)
* Sepolia slot: 1741159776 (Wed, Mar 5 at 07:29:36 UTC)
* +30 day mainnet slot: 1744154711 (Tue, Apr 8 at 23:25:11 UTC)

## For Node Operators

Node operators will need to upgrade to the respective releases before the activation dates.These following steps are necessary for every node operator:

* op-node at [v1.6.17](https://github.com/bobanetwork/boba/releases/tag/v1.6.17)
* op-geth at [v1.101503.0](https://github.com/bobanetwork/op-geth/releases/tag/v1.101503.0)

* op-erigon at [v1.2.12-beta.1](https://github.com/bobanetwork/op-erigon/releases/tag/v1.2.12-beta.1)

> [!NOTE]
>
> Full node operators, meaning those who are running op-geth with `gc-mode=full`, will need to reference the [`op-geth v1.101411.8`release notes](https://github.com/bobanetwork/op-geth/releases/tag/v1.101411.8) to handle an intermediate upgrade step before upgrading to the latest release. Archive node operators, `gc-mode=archive`, can skip this step and upgrade directly to the latest release.
