---
description: Contract deployment examples
---

# Contract Deployment Example

Please refer to the `./boba_examples` folder. Contract examples include:

1. **hardhat-simple-storage** - shows how to deploy a storage contract to Goerli Boba
2. **init-fund-l2** - shows how to deposit funds from L1 to L2, on Goerli Boba
3. **truffle-erc20** Simple ERC20 Token Truffle Tutorial

We'll work though one of those examples in more detail.



<figure><img src="../../.gitbook/assets/simple ERC20.png" alt=""><figcaption></figcaption></figure>

Welcome to our ERC20 Truffle example. If you're interested in writing your first L2 smart contract using Truffle as your smart contract testing framework, then you've come to the right place. This repo serves as an example for how go through and compile/test/deploy your contracts on Ethereum and the Boba L2.

Let's begin.



<figure><img src="../../.gitbook/assets/step 1.png" alt=""><figcaption></figcaption></figure>

Compiling a contract for Boba is identical to compiling a contract for Ethereum mainchain. Notably, all standard solidity compiler versions can be used. For this ERC20, we will use `0.6.12`. Create a `truffle-config.js` and add the following to it:

```js
const HDWalletProvider = require('@truffle/hdwallet-provider')

require('dotenv').config()
const env = process.env

const pk_1 = env.pk_1
const pk_2 = env.pk_2

module.exports = {
  contracts_build_directory: './build',
  networks: {
    boba_goerli: {
      provider: function () {
        return new HDWalletProvider({
          privateKeys: [pk_1, pk_2],
          providerOrUrl: 'https://goerli.boba.network',
        })
      },
      network_id: 2888,
      host: 'https://goerli.boba.network',
    }
  },
  compilers: {
    solc: {
      version: '0.6.12',
    },
  },
}
```

Now add a `.env` file that follows the format of `env.example` with two private keys. **NOTE: these accounts must be funded, i.e. contain enough Goerli ETH to cover the cost of the deployment.** Then,

```
yarn compile
```

Yep, it's that easy. You can verify that everything went well by looking for the `build` directory that contains your new JSON files. Now let's move on to testing!



<figure><img src="../../.gitbook/assets/step 2.png" alt=""><figcaption></figcaption></figure>

Woot! It's time to test our contract. Since the JSON RPC provider URL (for Boba Goerli) has already been specified in your Truffle config file, all we need to do next is run the test command. Run:

```
yarn test:integration
```

You should see a set of passing tests for your ERC20 contract.

```bash
$ truffle test ./test/erc20.spec.js --network boba_goerli --config truffle-config.js
Using network 'boba_goerli'.

Compiling your contracts...
===========================
> Everything is up to date, there is nothing to compile.

  Contract: ERC20
    ✓ creation: should create an initial balance of 10000 for the creator (562ms)
    ✓ creation: test correct setting of vanity information (1697ms)
    ✓ creation: should succeed in creating over 2^256 - 1 (max) tokens (2350ms)
    ✓ transfers: ether transfer should be reversed. (1699ms)
    ✓ transfers: should transfer 10000 to accounts[1] with accounts[0] having 10000 (1986ms)
    ✓ transfers: should fail when trying to transfer 10001 to accounts[1] with accounts[0] having 10000 (664ms)
    ✓ transfers: should handle zero-transfers normally (570ms)
    ✓ approvals: msg.sender should approve 100 to accounts[1] (1981ms)
    ✓ approvals: approve max (2^256 - 1) (2124ms)
    ✓ events: should fire Transfer event properly (1413ms)
    ✓ events: should fire Transfer event normally on a zero transfer (1424ms)
    ✓ events: should fire Approval event properly (1603ms)


  12 passing (43s)

✨  Done in 53.27s.
```

If so, congrats! You're ready to deploy an application to Boba. It really is that easy.



<figure><img src="../../.gitbook/assets/step 3.png" alt=""><figcaption></figcaption></figure>

Now we're going to deploy a contract using `truffle`. For Truffle based deployments, we're going to use Truffle's `migrate` command to run a migrations file for us that will deploy the contract we specify.

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
    tokenSymbol
  )
}
```

Now we're ready to run our migrations file! Let's go ahead and deploy this contract:

```
yarn deploy
```

After a few seconds your contract should be deployed. Now you'll see this in your terminal:

```bash
$ yarn deploy
yarn run v1.22.15
$ truffle migrate --network boba_goerli --config truffle-config

Compiling your contracts...
===========================
> Everything is up to date, there is nothing to compile.



Starting migrations...
======================
> Network name:    'boba_goerli'
> Network id:      2888
> Block gas limit: 11000000 (0xa7d8c0)


1_deploy_ERC20_contract.js
==========================

   Replacing 'ERC20'
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



<figure><img src="../../.gitbook/assets/troubleshooting.png" alt=""><figcaption></figcaption></figure>

Example project not working? [Create a Github Issue](https://github.com/bobanetwork/boba/issues).
