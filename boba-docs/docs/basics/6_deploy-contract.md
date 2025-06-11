---
description: Contract deployment examples
---

# Contract Deployment Example

Please refer to the [Contract example](https://github.com/bobanetwork/contract-example) repository.

We'll work through that one in this quick tutorial. The example linked above is a simple smart contract that allows you to store a single value and retrieve it.

It's a great starting point for learning how to deploy a contract to Boba. Let's begin.

## Step 1

Compiling a contract for Boba is identical to compiling a contract for Ethereum mainchain. Notably, all standard solidity compiler versions can be used. For this contract, we will use `0.8.9`.

If you check the `hardhat.config.ts` file, you'll see the following configuration in essence:

```js
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
```

Now add a `.env` file that follows the format of `env.example` with your private key. **NOTE: this account must be funded, i.e. contain enough Sepolia ETH to cover the cost of the deployment.** Then,

```
hardhat compile
```

Yep, it's that easy. You can verify that everything went well by looking for the `build` directory that contains your new JSON files. Now let's move on to testing!

## Step 2

Woot! It's time to test our contract. Since the JSON RPC provider URL (for Boba Sepolia) has already been specified in your Hardhat config file, all we need to do next is run the test command. Run:

```
yarn test:integration
```

You should see a set of passing tests for your contract. You can check a production-grade project here: [LightBridge](https://github.com/bobanetwork/light-bridge).

```bash
hardhat test test/contract.spec.ts --show-stack-traces
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

If so, congrats! You're ready to deploy an application to Boba. It really is that easy.

## Step 3

Now we're going to deploy a contract using `hardhat`.

First, let's create that deployment file. Create a new directory called `deploy` in the topmost path of your project and create a file within it called `contract.deploy.ts`.

Next, within `contract.deploy.ts`, we're going to add the following logic:

```js

console.log(`'Deploying contract...`)

const provider = hre.ethers.provider
const network = hre.network

console.log(`Network name=${network?.name}`)

const deployer = new Wallet(network.config.accounts[0], provider)

const Factory__YourContract = new ethers.ContractFactory(
  YourContractJson.abi,
  YourContractJson.bytecode,
  deployer
)

let gasLimit = prompt("Custom gas limit? [number/N]")
if (isNaN(gasLimit?.toLowerCase())) {
  gasLimit = null;
} else {
  gasLimit = parseInt(gasLimit)
}

YourContract = await Factory__YourContract.deploy({gasLimit})
let res = await YourContract.deployTransaction.wait()
console.log(`Deployed contract: `, res)

console.log(`Contract deployed to: ${YourContract.address}`)
```

Now we're ready to run our deployment file! Let's go ahead and deploy this contract:

```
hardhat deploy ./contracts/YourContract.sol --network boba_sepolia
```

After a few seconds your contract should be deployed. Now you'll see this in your terminal:

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

That's pretty much it. Contracts deployed! Tutorial complete. Hopefully now you know the basics of working with Boba!

## Troubleshooting

Example project not working? [Create a Github Issue](https://github.com/bobanetwork/boba/issues).
