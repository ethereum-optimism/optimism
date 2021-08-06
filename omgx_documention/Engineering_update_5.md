# DRAFT Engineering update #5

- [1. New code/features](#1-new-code-features)
- [2. Hybrid Compute](#2-hybrid-compute)
- [3. Staking and Earning](#3-staking-and-earning)
- [4. Compound Protocol DAO](#4-compound-protocol-dao)
- [5. Key Security](#5-key-security)

August 5 2021

Greetings from your engineering team. We are in the process of our first `regenesis`, which involves deploying new system contracts to Rinkeby and then restoring the system state (including account balances and nonces). The Rinkeby regenesis is needed so that Rinkeby reflects the contracts that are currently being deployed to Ethereum Mainnet. The OMGX Mainnet contracts will serve as the starting point for spinning up and testing the mainnet infrastructure - such as the `api-watcher` for cross-chain analytics, the mainnet AWS cloud `integration` and `production` endpoints, the `replica` service needed by blockexplorers, and The Graph (which is needed by several DeFi mainnet launch partners).

## 1. New code/features 

* As desired and expected, *load and stress testing* of OMGX Rinkeby surfaced rare errors in coordinating the L1 with the L2, requiring (minor) changes to the L2 Geth and the data transport layer - see for example https://github.com/omgnetwork/optimism/pull/230. 

* Improvement of the *cross-chain liquidity pools* - the contracts now revert unfillable swaps (https://github.com/omgnetwork/optimism/pull/257) so that user funds do not get stuck in edge cases in which a swap passes initial balance checks but then cannot be filled when the transaction actually arrives on the other chain. 

* *Fees* are now turned on by default on OMGX Rinkeby, so Rinkeby ETH is needed for contract deployments and transfers. The fee logic considers ETH price, chain utilization and congestion, and two hard coded limits, the `floor` and the `ceiling`. These two parameters define the lowest possible fee (e.g. transactions will never cost less than X) and the highest possible fee (e.g. transactions will never cost more than Y). In some system configurations (e.g. when OMGX utilization is low but ETH is expensive), operating losses will need to covered by operator subsidies, the goal being to guarantee a 24/7 pleasant, cost-effective, and convenient L2 experience. At the code of the OMGX L2 operations model is this function:

```javascript
private async _updateGasPrice(): Promise<void> {
    const gasPrice = await this.state.OVM_GasPriceOracle.gasPrice()
    const gasPriceInt = parseInt(gasPrice.toString())
    this.logger.info("Got L2 gas price", { gasPrice: gasPriceInt })

    let targetGasPrice = this.options.gasFloorPrice

    if (this.state.L1ETHCostFee.gt(this.state.L2ETHCollectFee)) {
      const estimatedGas = BigNumber.from(this.state.numberOfBlocksInterval).mul(this.state.avgL2GasLimitPerBlock)
      const estimatedGasPrice = this.state.L1ETHCostFee.sub(this.state.L2ETHCollectFee).div(estimatedGas)

      if (estimatedGasPrice.gt(BigNumber.from(this.options.gasRoofPrice))) {
        targetGasPrice = this.options.gasRoofPrice
      } else if (estimatedGasPrice.gt(BigNumber.from(this.options.gasFloorPrice))) {
        targetGasPrice = parseInt(estimatedGasPrice.toString())
      }
    }

    if (gasPriceInt !== targetGasPrice && (
      targetGasPrice > (1 + this.options.gasPriceMinPercentChange) * gasPriceInt ||
      targetGasPrice < (1 - this.options.gasPriceMinPercentChange) * gasPriceInt)
    ) {
      this.logger.debug("Updating L2 gas price...")
      const tx = await this.state.OVM_GasPriceOracle.setGasPrice(targetGasPrice, { gasPrice: 0 })
      await tx.wait()
      this.logger.info("Updated L2 gas price", { gasPrice: targetGasPrice })
    } else {
      this.logger.info("No need to update L2 gas price", { gasPrice: gasPriceInt, targetGasPrice })
    }
  }
```

* Mainnet support for the *webwallet*. In concert with pending contract deployment on mainnet, the webwallet will soon expose a third net choice, `mainnet`, in addition to `local` and `rinkeby`. 

* History support for the *webwallet*. Also, for mainnet, the webwallet now shows more information about transactions, so that wallet users can see the status of their transactions inclucing block data, hashes, and cross-chain timestamps and receipts.

* Name change for the *webwallet*. Reflecting its broad actual use (wallet, earn/stake, NFT minting, transaction histories, DAO interface, and more...) the webwallet will be renamed to *gateway*. The current endpoints will keep working, but in the near future, https://gateway.rinkeby.omgx.network (and https://gateway.mainnet.omgx.network) will serve as the main initial points of contact with the OMGX L2. 

## 2. Hybrid Compute

Although not ready for prime time, you will see increasing mention to a system referred to as *Turing* or *Hybrid compute*. This system will make it possible for external contract interactions to trigger contracts to interact with arbitrary off-chain compute endpoints. This means, for example, that DeFi contracts will be able to update themselves. Consider an external caller that wishes to execute a stableswap - there are situations in which it would be useful for the contract to optimize internal parameters based on current market conditions. With Turing, a contract can e.g. seek extra information about loss-reducing curve parameters from AWS lambda endpoints. 

Follow along with our development work on the `offchain-prototype` branch (https://github.com/omgnetwork/optimism/tree/offchain-prototype). If this sounds confusing, that's because it sort of is - among other things, it would be difficult for a normal blockchain (with distributed mining) to implement such logic. In our system, which is designed to be fully compatible with fraud-proving and rollups, the first contract call always reverts, and it's the reversion that triggers the off-chain compute request. The AWS endpoint then pushes new parameters into the contract, so that when the external caller retries their contract interaction, the contract now features recently updated curve parameters. From the perspective of L2 Geth, this is just a succession of seemingly unrelated, normal events, namely, (1) a contract is called and the call reverts, (2) a external API pushes new data into that contract, and (3) the original contract interaction, when resubmitted, now succeeds.

It certainly is possible to implement such a system with off-the-shelf tech - for example a smart contract could be written to always revert upon initial interaction, the revert could be detected off-chain, and then an off-chain watcher could quickly push new parameters into a stableswap contract. However, Turing makes this easy and ensures that the desired events take place in an orderly manner. It is interesting to wonder if Turing can be somehow combined with NFTs, generating for example NFTs that can control or harness off-chain compute, to gradually evolve, acquire new skills,  or interact with other NFTs in creative ways. 

## 3. Staking and Earning

As already noted in previous engineering updates, you can earn rewards today for staking ETH on OMGX Rinkeby, to provide liquidity for the fast swap on/off. With OMGX Mainnet on the horizon, you will be able to simultaneously help the network by allowing users to quickly on-ramp and off-ramp, and earn staking rewards. You can try out the Rinkeby LP system at https://webwallet.rinkeby.omgx.network > Earn.

## 4. Compound Protocol DAO

In work spearheaded by our four summer engineering interns (Go Esteban, Jesus, Ryan, and Mehul!), the contracts related to Compound v2 DAO have been deployed on OMGX Rinkeby and pass integration tests. Currently, the interns are adding a simple UI to the webwallet/gateway, to allow our community to begin to more directly shape our direction and future.

## 5. Key Security

Again with the pending Mainnet, private key management will soon be turned over to separate security layer (https://github.com/omgnetwork/optimism/pull/188), based on the Hashicorp Vault (https://learn.hashicorp.com/vault). Since these keys will among other things hold significant ETH for ongoing L2 operations, we need these keys to be well protected.
