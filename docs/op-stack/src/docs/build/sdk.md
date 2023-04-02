---
title: Using the SDK with OP Stack
lang: en-US
---

When building applications for use with your OP Stack, you can continue to use [the Optimism JavaScript SDK](https://sdk.optimism.io/).
The main difference is you need to provide some contract addresses to the `CrossDomainMessenger` because they aren't preconfigured.


## Contract addresses

### L1 contract addresses

The contract addresses are in `.../optimism/packages/contracts-bedrock/deployments/getting-started`, which you created when you deployed the L1 contracts.

| Contract name when creating `CrossDomainMessenger` | File with address |
| - | - |
| `AddressManager`         | `Lib_AddressManager.json`
| `L1CrossDomainMessenger` | `Proxy__OVM_L1CrossDomainMessenger.json`
| `L1StandardBridge`       | `Proxy__OVM_L1StandardBridge.json`
| `OptimismPortal`         | `OptimismPortalProxy.json`
| `L2OutputOracle`         | `L2OutputOracleProxy.json`


### Unneeded contract addresses

Some contracts are required by the SDK, but not actually used.
For these contracts you can just specify the zero address:

- `StateCommitmentChain`
- `CanonicalTransactionChain`
- `BondManager`

In JavaScript you can create the zero address using the expression `"0x".padEnd(42, "0")`.

## The CrossChainMessenger object

These directions assume you are inside the [Hardhat console](https://hardhat.org/hardhat-runner/docs/guides/hardhat-console).
They further assume that your project already includes the Optimism SDK [`@eth-optimism/sdk`](https://www.npmjs.com/package/@eth-optimism/sdk).

1. Import the SDK

   ```js
   optimismSDK = require("@eth-optimism/sdk")
   ```

1. Set the configuration parameters.

   | Variable name | Value |
   | - | - |
   | `l1Url` | URL to an RPC provider for L1, for example `https://eth-goerli.g.alchemy.com/v2/<api key>`
   | `l2Url` | URL to your OP Stack. If running on the same computer, it is `http://localhost:8545`
   | `privKey` | The private key for an account that has some ETH on the L1


1. Create the [providers](https://docs.ethers.org/v5/api/providers/) and [signers](https://docs.ethers.org/v5/api/signer/).

   ```js
   l1Provider = new ethers.providers.JsonRpcProvider(l1Url)
   l2Provider = new ethers.providers.JsonRpcProvider(l2Url)
   l1Signer = new ethers.Wallet(privKey).connect(l1Provider)
   l2Signer = new ethers.Wallet(privKey).connect(l2Provider)
   ```

1. Create the L1 contracts structure.

   ```js
   zeroAddr = "0x".padEnd(42, "0")
   l1Contracts = {
      StateCommitmentChain: zeroAddr,
      CanonicalTransactionChain: zeroAddr,
      BondManager: zeroAddr,
      // These contracts have the addresses you found out earlier.
      AddressManager: "0x....",   // Lib_AddressManager.json
      L1CrossDomainMessenger: "0x....",   // Proxy__OVM_L1CrossDomainMessenger.json  
      L1StandardBridge: "0x....",   // Proxy__OVM_L1StandardBridge.json
      OptimismPortal: "0x....",   // OptimismPortalProxy.json
      L2OutputOracle: "0x....",   // L2OutputOracleProxy.json
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


1. Create the [`CrossChainMessenger`](https://sdk.optimism.io/classes/crosschainmessenger) object.

   ```js
   crossChainMessenger = new optimismSDK.CrossChainMessenger({
      bedrock: true,
      contracts: {
         l1: l1Contracts
      },
      bridges: bridges,
      l1ChainId: await l1Signer.getChainId(),
      l2ChainId: await l2Signer.getChainId(),
      l1SignerOrProvider: l1Signer,
      l2SignerOrProvider: l2Signer,    
   })
   ```

## Verify SDK functionality

To verify the SDK's functionality, transfer some ETH from L1 to L2.

1. Get the current balances.

   ```js
   balances0 = [
      await l1Provider.getBalance(l1Signer.address),
      await l2Provider.getBalance(l1Signer.address)
   ]
   ```

1. Transfer 1 gwei.

   ```js
   tx = await crossChainMessenger.depositETH(1e9)
   rcpt = await tx.wait()
   ```

1. Get the balances after the transfer.

   ```js
   balances1 = [
      await l1Provider.getBalance(l1Signer.address),
      await l2Provider.getBalance(l1Signer.address)
   ]
   ```

1. See that the L1 balance changed (probably by a lot more than 1 gwei because of the cost of the transaction).

   ```js
   (balances0[0]-balances1[0])/1e9
   ```

1. See that the L2 balance changed (it might take a few minutes).  

   ```js
   ((await l2Provider.getBalance(l1Signer.address))-balances0[1])/1e9
   ```
