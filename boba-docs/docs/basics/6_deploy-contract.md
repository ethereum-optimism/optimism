---
description: Contract deployment examples
---

# Contract Deployment Example

import Tabs from '@theme/Tabs';
import TabItem from '@theme/TabItem';

This is a quick tutorial to help you learn how to deploy a smart contract to the Boba network. We'll provide an example contract that allows you to store a single value and retrieve it. 

## Prerequisites

Before you start, please make sure you have the following installed on your machine:

- [`git`](https://github.com/)
- [`hardhat`](http://hardhat.org/docs)
- [`yarn`](https://yarnpkg.com/)
- [`npx`](https://docs.npmjs.com/cli/v8/commands/npx)
- A crypto wallet with access to your private key

Then clone the [Contract example](https://github.com/bobanetwork/contract-example) repository to your environment; this will be the smart contract we work with in this tutorial. 

Once you've prepared your environment, you're ready to get started!

## Compile the Contract

Compiling a contract for Boba is identical to compiling a contract for Ethereum mainchain. Notably, all standard [`solidity`](https://soliditylang.org/) compiler versions can be used. For this contract, we will use `0.8.9`.

### Set the Config

Checking the `hardhat.config.ts` file, you'll see some of the following configuration:

```js title="contract-example/hardhat.config.ts"
networks: {
  boba_sepolia: {
    url: 'https://sepolia.boba.network',
      accounts: [process.env.DEPLOYER_PK],
  },
},
solidity: {
  version: '0.8.9',
    settings: {
    optimizer: {enabled: true, runs: 200},
  },
},
etherscan: {
  apiKey: {
      boba_sepolia: "boba", // not required, set placeholder
  },
  customChains: [
    {
      network: "boba_sepolia",
      chainId: 28882,
      urls: {
        apiURL: "https://api.routescan.io/v2/network/testnet/evm/28882/etherscan",
        browserURL: "https://testnet.bobascan.com"
      },
    },
  ],
}

...
```
Notice that `process.env.DEPLOYER_PK`. We need to set that variable with your own private key. Navigate to the `.env` file, and you'll find this:

```js title="contract-example/.env"
DEPLOYER_PK=0xYOUR_PRIVATE_KEY_HERE
```

Replace "`YOUR_PRIVATE_KEY_HERE`" with your private key (and keep the preceding `0x`).

### Compile the Contract

:::warning
This account must be funded, i.e. contain enough Sepolia ETH to cover the cost of the deployment.
:::

With your config set, you can compile the contract with your choice of package manager by running the following:

<Tabs>
  <TabItem value="yarn">

  ```bash
  yarn hardhat compile
  ```

  </TabItem>
  <TabItem value="npx">

  ```bash
  npx hardhat compile
  ```

  </TabItem>
</Tabs>

You can verify that everything went well by looking for the `build` directory that contains your new `JSON` files. Now let's move on to testing!

## Test the Contract

We've set the `JSON RPC` provider for Boba Sepolia in the `hardhat` config file, so all we need to do next is run the test command:

```bash
yarn test
```

You should see a set of passing tests for your contract: 

```bash
$ hardhat test test/contract.spec.ts --show-stack-traces
Using network 'boba_sepolia'.

Compiling your contracts...
===========================
> Everything is up to date, there is nothing to compile.

   your tests
Contract deployed at:  0x5FbDB2315678afecb367f032d93F642f64180aa3
    √ always succeeds
Contract deployed at:  0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512
    √ contract deployed


  2 passing (3s)

✨  Done in 3s.
```

If so, congrats! You're ready to deploy an application to the Boba network.

## Deploy the Contract

Included in our provided example contract is the deployment file (`contract-example/deploy/contract.deploy.ts`, for the curious user). To use the file, run the following:

```bash
npx hardhat deploy --network boba_sepolia
```

After a few seconds your contract should deploy, and you should see this in your terminal:

```bash
yarn deploy
yarn run v1.22.15
hardhat deploy --network boba_sepolia

Compiling your contracts...
===========================
> Everything is up to date, there is nothing to compile.



Starting deployment scripts...
======================
> Network name:    'boba_sepolia'

contract.deploy.ts
==========================

   Deploying 'YourContract'
   -----------------
   > transaction hash:    0xe7cc5d048ffd426587b7d9c89aed4b0d7b2bd29c5532300bce8a9a57a4c4d689
   > Blocks: 0            Seconds: 0
   > contract address:    0xE769105D8bDC4Fb070dD3057c7e48BB98771dE15
   > block number:        6270
   > block timestamp:     1635787822
   > account:             0x21724227d169eAcBf216dE61EE7dc28F80CF8A92
   > balance:             0.901997296123301024
   > gas used:            855211 (0xd0cab)
   > gas price:           0.02 gwei
   > value sent:          0 ETH
   > total cost:          0.00001710422 ETH

   > Saving artifacts
   -------------------------------------
   > Total cost:       0.00001710422 ETH


Summary
=======
> Total deployments:   1
> Final cost:          0.00001710422 ETH


✨  Done in 10.11s.
```

## Verify the Contract

Using our tool [Bobascan](https://testnet.bobascan.com/), you can verify the contract has been deployed and is valid. Sign in by connecting your crypto wallet to Bobascan, then navigate to your transactions. [TODO: ADD IMAGES OF WHERE TO FIND TRANSACTIONS AND CONTRACT ADDRESS] This will lead you to the contract address of your newly deployed contract. Copy this address, then replace "`YOUR_CONTRACT_ADDRESS_HERE`" with it in the command below to verify your contract:

```bash
npx hardhat verify --network boba_sepolia YOUR_CONTRACT_ADDRESS_HERE
```

You should see the following output once you've successfully verified your contract:

```console
Nothing to compile
Successfully submitted source code for contract
contracts/YourContract.sol:YourContract at 0xFe0447Eecf08Eb16DfC5a3Fa2210B30dE3f2803d
for verification on the block explorer. Waiting for verification result...

Successfully verified contract YourContract on Etherscan.
https://testnet.bobascan.com/address/0xFe0447Eecf08Eb16DfC5a3Fa2210B30dE3f2803d#code
```

That's all it takes to deploy and verify your contract! This contract is simple enough (it stores and retrieves a single value) to get you started in using more complicated contracts. If you're feeling more ambitious, you can check one of our production-grade projects [here](https://github.com/bobanetwork/light-bridge).

## Troubleshooting

Example project not working? [Create a Github Issue](https://github.com/bobanetwork/boba/issues).
