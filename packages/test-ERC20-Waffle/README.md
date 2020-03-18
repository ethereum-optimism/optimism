# Getting Started with the OVM: Simple ERC20 Token Tutorial

Hi there! Welcome to our OVM ERC20 tutorial.

If you're interested in writing your first L2-compatible smart contract, you've come to the right place!  This tutorial will cover how to move an existing contract and its tests into the wonderful world of L2.

## Set up

To start out, clone this example repo as a starting point.

```bash
git clone https://github.com/ethereum-optimism/ovm-integration-tests.git
```
Now, enter the repository

```bash
cd ovm-integration-tests/ERC20-Example
```
Install all dependencies

```bash
yarn install
```

This project represents a fresh, non-OVM ERC20 contract example. Feel free to stop here and have a quick look at the contract and tests. 

In this tutorial, we'll cover the steps required to bring it into the world of L2. First, let's make sure all of our tests are running in our normal Ethereum environment: 

```bash
yarn test
```
You should see all of the tests passing. We're now ready to convert the project to build and test in an OVM environment!

## Configuring the Transpiler
First, we need to configure ``ethereum-waffle`` (which is an alternative to `truffle`) to use our new transpiler-enabled Solidity compiler.  To do this, edit the ``waffle-config.json`` and replace it with:

```json=
{
  "sourcesPath": "./contracts",
  "targetPath": "./build",
  "npmPath": "../../node_modules",
  "solcVersion": "../../node_modules/@eth-optimism/solc-transpiler",
  "compilerOptions": {
    "outputSelection": {
      "*": {
        "*": ["*"]
      }
    },
    "executionManagerAddress": "0xA193E42526F1FEA8C99AF609dcEabf30C1c29fAA"
  }
}
```
## Using the Full Node
To use the OVM to run our tests, open the test file at ``test/erc20.spec.js``. We can import the OVM-ified versions of `getWallets`, `createMockProvider`, and `deployContract` near the top of the test file:

```javascript=
const { createMockProvider, getWallets, deployContract } = require('@eth-optimism/rollup-full-node')
```

Now remove the duplicated imports from `ethereum-waffle`, replacing the import on `Line 2` with:

```javascript=
const {solidity} = require('ethereum-waffle');
```

Our imports at the top of the file should now look like: 

```javascript=
const {use, expect} = require('chai');
const {solidity} = require('ethereum-waffle');
const {createMockProvider, getWallets, deployContract } = require('@eth-optimism/rollup-full-node')
const ERC20 = require('../build/ERC20.json');
```


We're almost there!  After we've run our tests on the OVM, we need to stop our OVM server. We're going to add a single line of code after our `before()` hook in order to close our OVM Server after our tests run:

```javascript=
  before(async () => {
    ...
  })

  //ADD THIS
  after(() => {provider.closeOVM()}) 
```
## Running the New Tests
Great, we're ready to go!  Now you can try to re-run your tests on top of the OVM with

```bash
yarn test
```

## Wasn't that easy?
The OVM provides a fresh new take on layer 2 development: it's identical to layer 1 development.  No hoops, no tricks--the Ethereum you know and love, ready to scale up with L2.  For more info on our progress and what's going on behind the scenes, you can follow us on [Twitter](https://twitter.com/optimismPBC) and [check out our docs](https://docs.optimism.io)!

## Troubleshooting
Not working for you? It might help to check out this [easy to read Diff](https://i.imgur.com/DEU7wXC.png) to show you exactly which lines you should be altering. You can also check out the final working codebase that we have added to a seperate branch [here](https://github.com/ethereum-optimism/ERC20-Example/tree/final_result)

Still not working? [Create a Github Issue](https://github.com/ethereum-optimism/ERC20-Example/issues), or hop in our [Discord](https://discordapp.com/invite/jrnFEvq) channel and ask away.
