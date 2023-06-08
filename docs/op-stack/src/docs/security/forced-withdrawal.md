---
title: Forced withdrawal from an OP Stack blockchain
lang: en-US
---


## What is this?

Any assets you own on an OP Stack blockchain are backed by equivalent assets on the underlying L1, locked in a bridge. 
In this article you learn how to withdraw these assets directly from L1.

Note that the steps here do require access to an L2 endpoint.
However, that L2 endpoint can be a read-only replica.


## Setup 

The code to go along with this article is available at [our tutorials repository](https://github.com/ethereum-optimism/optimism-tutorial/tree/main/op-stack/forced-withdrawal).

1. Clone the repository, move to the correct directory, and install the required dependencies.

   ```sh
   git clone https://github.com/ethereum-optimism/optimism-tutorial.git
   cd optimism-tutorial/op-stack/forced-withdrawal
   npm install
   ```

1. Copy the environment setup variables.

   ```sh
   cp .env.example .env
   ```

1. Edit `.env` to set these variables:

   | Variable             | Meaning |
   | -------------------- | ------- |
   | L1URL                | URL to L1 (Goerli if you followed the directions on this site)
   | L2URL                | URL to the L2 from which you are withdrawing
   | PRIV_KEY             | Private key for an account that has ETH on L2. It also needs ETH on L1 to submit transactions
   | OPTIMISM_PORTAL_ADDR | Address of the `OptimismPortalProxy` on L1.


## Withdrawal

### ETH withdrawals

The easiest way to withdraw ETH is to send it to the bridge, or the cross domain messenger, on L2.

1. Enter the Hardhat console.

   ```sh
   npx hardhat console --network l1
   ```

1. Specify the amount of ETH you want to transfer.
   This code transfers one hundred'th of an ETH.

   ```js
   transferAmt = BigInt(0.01 * 1e18)
   ``` 

1. Create a contract object for the [`OptimismPortal`](https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/L1/OptimismPortal.sol) contract.

   ```js
   optimismContracts = require("@eth-optimism/contracts-bedrock")
   optimismPortalData = optimismContracts.getContractDefinition("OptimismPortal")
   optimismPortal = new ethers.Contract(process.env.OPTIMISM_PORTAL_ADDR, optimismPortalData.abi, await ethers.getSigner())
   ```

1. Send the transaction.

   ```js
   txn = await optimismPortal.depositTransaction(
      optimismContracts.predeploys.L2StandardBridge,
      transferAmt,
      1e6, false, []
   )
   rcpt = await txn.wait()
   ```


1. To [prove](https://sdk.optimism.io/classes/crosschainmessenger#proveMessage-2) and [finalize](https://sdk.optimism.io/classes/crosschainmessenger#finalizeMessage-2) the message we need the hash. 
   Optimism's [core-utils package](https://www.npmjs.com/package/@eth-optimism/core-utils) has the necessary function.

   ```js
   optimismCoreUtils = require("@eth-optimism/core-utils")
   withdrawalData = new optimismCoreUtils.DepositTx({
      from: (await ethers.getSigner()).address,
      to: optimismContracts.predeploys.L2StandardBridge,
      mint: 0,
      value: ethers.BigNumber.from(transferAmt),
      gas: 1e6,
      isSystemTransaction: false,
      data: "",
      domain: optimismCoreUtils.SourceHashDomain.UserDeposit,
      l1BlockHash: rcpt.blockHash,
      logIndex: rcpt.logs[0].logIndex,
   })
   withdrawalHash = withdrawalData.hash()
   ```

1. Create the object for the L1 contracts, [as explained in the documentation](../build/sdk.md).
   You will create an object similar to this one:

   ```js
   L1Contracts = {
      StateCommitmentChain: '0x0000000000000000000000000000000000000000',
      CanonicalTransactionChain: '0x0000000000000000000000000000000000000000',
      BondManager: '0x0000000000000000000000000000000000000000',
      AddressManager: '0x432d810484AdD7454ddb3b5311f0Ac2E95CeceA8',
      L1CrossDomainMessenger: '0x27E8cBC25C0Aa2C831a356bbCcc91f4e7c48EeeE',
      L1StandardBridge: '0x154EaA56f8cB658bcD5d4b9701e1483A414A14Df',
      OptimismPortal: '0x4AD19e14C1FD57986dae669BE4ee9C904431572C',
      L2OutputOracle: '0x65B41B7A2550140f57b603472686D743B4b940dB'
   }
   ```

1. Create the data structure for the standard bridge.

   ```js
    bridges = { 
      Standard: { 
         l1Bridge: l1Contracts.L1StandardBridge, 
         l2Bridge: "0x4200000000000000000000000000000000000010", 
         Adapter: optimismSDK.StandardBridgeAdapter
      },
      ETH: {
         l1Bridge: l1Contracts.L1StandardBridge, 
         l2Bridge: "0x4200000000000000000000000000000000000010", 
         Adapter: optimismSDK.ETHBridgeAdapter
      }
   }
   ```


1. Create [a cross domain messenger](https://sdk.optimism.io/classes/crosschainmessenger).
   This step, and subsequent ETH withdrawal steps, are explained in [this tutorial](https://github.com/ethereum-optimism/optimism-tutorial/tree/main/cross-dom-bridge-eth).

   ```js
   optimismSDK = require("@eth-optimism/sdk")
   l2Provider = new ethers.providers.JsonRpcProvider(process.env.L2URL)
   await l2Provider._networkPromise
   crossChainMessenger = new optimismSDK.CrossChainMessenger({
      l1ChainId: ethers.provider.network.chainId,
      l2ChainId: l2Provider.network.chainId,
      l1SignerOrProvider: await ethers.getSigner(),
      l2SignerOrProvider: l2Provider,
      bedrock: true,
      contracts: {
         l1: l1Contracts
      },
      bridges: bridges
   })   
   ```

1. Wait for the message status for the withdrawal to become `READY_TO_PROVE`.
   By default the state root is written every four minutes, so you're likely to need to need to wait.

   ```js
   await crossChainMessenger.waitForMessageStatus(withdrawalHash, 
       optimismSDK.MessageStatus.READY_TO_PROVE)
   ```
      
1. Submit the withdrawal proof.

   ```js
   await crossChainMessenger.proveMessage(withdrawalHash)
   ```

1. Wait for the message status for the withdrawal to become `READY_FOR_RELAY`.
   This waits the challenge period (7 days in production, but a lot less on test networks).

   ```js
   await crossChainMessenger.waitForMessageStatus(withdrawalHash, 
      optimismSDK.MessageStatus.READY_FOR_RELAY)
   ```   


1. Finalize the withdrawal.
   See that your balance changes by the withdrawal amount.

   ```js
   myAddr = (await ethers.getSigner()).address
   balance0 = await ethers.provider.getBalance(myAddr)
   finalTxn = await crossChainMessenger.finalizeMessage(withdrawalHash)
   finalRcpt = await finalTxn.wait()
   balance1 = await ethers.provider.getBalance(myAddr)
   withdrawnAmt = BigInt(balance1)-BigInt(balance0)
   ```

::: tip transferAmt > withdrawnAmt

Your L1 balance doesn't increase by the entire `transferAmt` because of the cost of `crossChainMessenger.finalizeMessage`, which submits a transaction.

:::