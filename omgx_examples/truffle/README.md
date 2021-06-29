# Getting Started with Optimistic Ethereum: Simple ERC20 Token Truffle Tutorial

Hi there! Welcome to our Optimistic Ethereum ERC20 Truffle example! If you're interested in writing your first L2-compatible smart contract using Truffle as your smart contract testing framework, then you've come to the right place! This repo serves as an example for how go through and compile/test/deploy your contracts on both Ethereum, Optimistic Ethereum, and OMGX.

Let's begin!

## Prerequisites

First, spin up a local L1/L2. Install needed modules:

```bash
$ cd optimism
$ yarn clean
$ yarn
$ yarn build
```

Then, navigate to the `/ops` folder and start the system. Make sure you have the docker app running!

```bash
$ cd ops
$ docker-compose down #only needed if you are currently running a local system
$ docker-compose build
$ docker-compose up -V
```

## Step 1: Compile your contracts for Optimistic Ethereum

Compiling a contract for Optimistic Ethereum is pretty easy! First we'll need to install the [`@eth-optimism/solc`](https://www.npmjs.com/package/@eth-optimism/solc). Since we currently only support `solc` versions `0.5.16`, `0.6.12`, and `0.7.6` for Optimistic Ethereum contracts, we'll be using version `0.7.6` in this example.
Let's add this package:

```sh
yarn add @eth-optimism/solc@0.7.6-alpha.1
```

Next, we just need to add a new `truffle-config-ovm.js` file to compile our contracts. Create `truffle-config-ovm.js` and add the following to it:

```js
const mnemonicPhrase = "candy maple cake sugar pudding cream honey rich smooth crumble sweet treat"
const HDWalletProvider = require('@truffle/hdwallet-provider')

module.exports = {
  contracts_build_directory: './build-ovm',
  networks: {
    optimism: {
      provider: function () {
        return new HDWalletProvider({
          mnemonic: {
            phrase: mnemonicPhrase
          },
          providerOrUrl: 'http://127.0.0.1:8545'
        })
      },
      network_id: 420,
      host: '127.0.0.1',
      port: 8545,
      gasPrice: 0,
    },
    omgx_rinkeby: {
      provider: function () {
        return new HDWalletProvider({
          mnemonic: {
            phrase: mnemonicPhrase
          },
          providerOrUrl: 'http://rinkeby.omgx.network'
        })
      },
      network_id: 28,
      host: 'http://rinkeby.omgx.network',
      port: 8545,
      gasPrice: 0,
    }
  },
  compilers: {
    solc: {
      // Add path to the optimism solc fork
      version: "node_modules/@eth-optimism/solc",
      settings: {
        optimizer: {
          enabled: true,
          runs: 1
        },
      }
    }
  }
}
```

Here, we specify the new custom Optimistic Ethereum compiler we just installed and the new build path for our optimistically compiled contracts. We also specify the network parameters of a local Optimistic Ethereum instance. This local instance will be set up soon, but we'll set this up in our config now so that it's easy for us later when we compile and deploy our Optimistic Ethereum contracts.

And we're ready to compile! All you have to do is specify the `truffle-config-ovm.js` config in your `truffle` command, like so:

```sh
yarn truffle compile --config truffle-config-ovm.js
```

Our `truffle-config-ovm.js` config file tells Truffle that we want to use the Optimistic Ethereum solidity compiler.

Yep, it's that easy. You can verify that everything went well by looking for the `build-ovm` directory that contains your new JSON files.

Here, `build-ovm` signifies that the contracts contained in this directory have been compiled for the OVM, the **O**ptimistic **V**irtual **M**achine, as opposed to the Ethereum Virtual Machine. Now let's move on to testing!

## Step 2: Testing your Optimistic Ethereum contracts

Woot! It's finally time to test our contract on top of Optimistic Ethereum. Since the two JSON RPC provider URLs (one for your local instance Ethereum and Optimistic Ethereum) have already been specified in your Truffle config files, all we need to do next is run the test command.To do that, run:

```sh
yarn truffle test ./test/erc20.spec.js --network optimism --config truffle-config-ovm.js
```

Notice that we are using `truffle-config-ovm.js` to let `truffle` know that we want to use the `build-ovm` folder as our path to our JSON files. (Remember that these JSON files were compiled using the Optimistic Ethereum solidity compiler!)

Additionally, we also specify the network we are testing on.
In this case, we're testing our contract on `optimistic_ethereum`.

You should see a set of passing tests for your ERC20 contract. If so, congrats!
You're ready to deploy an application to Optimistic Ethereum.
It really is that easy.

## Step 3: Deploying your Optimistic Ethereum contracts

Now we're going to deploy an Optimisic Ethereum contract using `truffle`. For Truffle based deployments, we're going to use Truffle's `migrate` command to run a migrations file for us that will deploy the contract we specify.

First, let's create that migrations file. Create a new directory called `migrations` in the topmost path of your project and create a file within it called `1_deploy_ERC20_contract.js`.

Next, within `1_deploy_ERC20_contract.js`, we're going to add the following logic:

```js
const ERC20 = artifacts.require('ERC20')

module.exports = function (deployer, accounts) {
  const tokenName = 'My Optimistic Coin'
  const tokenSymbol = 'OPT'
  const tokenDecimals = 1

  // deployment steps
  deployer.deploy(
    ERC20, 
    10000, 
    tokenName, 
    tokenDecimals, 
    tokenSymbol,
    { gasPrice: 0 }
  )
}
```

To quickly explain this file, first we import our artifact for our ERC20 contract.
Since we specified the build directory in our Truffle configs, Truffle knows whether we want to use either an Ethereum or Optimistic Ethereum contract artifact.

Now we're ready to run our migrations file!
Let's go ahead and deploy this contract:

```sh
yarn truffle migrate --network optimism --config truffle-config-ovm.js #deploy on your local L2

or...

yarn truffle migrate --network omgx_rinkeby --config truffle-config-ovm.js #deploy on omgx Rinkeby
```

After a few seconds your contract should be deployed! Now you'll see this in your terminal:

![Truffle contract migrations to Optimistic Ethereum complete](./assets/deploy-to-optimistic-ethereum.png)

And uh... yeah.
That's pretty much it.
Contracts deployed!
Tutorial complete. Hopefully now you know the basics of working with Optimistic Ethereum! 🅾️

------

## Further Reading

### OVM vs. EVM Incompatibilities

Our goal is to bring the OVM as close to 100% compatibility with all existing Ethereum projects, but our software is still in an early stage. [Our community hub docs](https://community.optimism.io/docs/protocol/evm-comparison.html) will maintain the most up to date list of known incompatibilities between the OVM and EVM, along with our plans to fix them.

### Wasn't that easy?

The OVM provides a fresh new take on layer 2 development: it's _mostly_ identical to layer 1 development.
However, there are a few differences that are worth noting, which you can read more about in our [EVM comparison documentation](https://community.optimism.io/docs/protocol/evm-comparison.html).
No hoops, no tricks--the Ethereum you know and love, ready to scale up with L2.
For more info on our progress and what's going on behind the scenes, you can follow us on [Twitter](https://twitter.com/optimismPBC).

Want to try deploying contracts to the Optimistic Ethereum testnet next? [Check out the full integration guide](https://community.optimism.io/docs/developers/integration.html) on the Optimism community hub.

------

## Troubleshooting

Example project not working? [Create a Github Issue](https://github.com/ethereum-optimism/Truffle-ERC20-Example/issues), or hop in our [Discord](https://discordapp.com/invite/jrnFEvq) channel and ask away.