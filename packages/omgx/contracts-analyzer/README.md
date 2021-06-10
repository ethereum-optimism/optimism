- [The Contracts Analyzer](#the-contract-analyzer)
  * [Prerequisites](#prerequisites)
  * [Setting Up](#setting-up)
  * [Add Contracts](#add-contracts)
  * [Notes](#notes)
  * [Deploying Contracts to LOCAL TEST SYSTEM](#deploying-contracts-to-local-test-system)
  * [Deploying Contracts to OMGX RINKEBY](#deploying-contracts-to-omgx-rinkeby)
  * [Test](#test)

# The Contracts Analyzer

This repo is used to analyze contracts written for L1, as a starting point for evaluating potential code changes needed to deploy them to L2.

## Prerequisites

Please make sure you've installed:

- [Git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git)
- [Node.js](https://nodejs.org/en/download/)
- [Yarn](https://classic.yarnpkg.com/en/docs/install#mac-stable)

## Setting Up

Set up the project by running:

```bash
yarn install
cd packages/omgx/contracts-analyzer
```

## Add Contracts

Copy your contracts into `/contracts` and run:

```bash
yarn build #build the smart contracts with optimistic solc
yarn analyze
```

You will probably have to `yarn add` multiple packages, and change/update pragmas, such as, to `pragma solidity 0.6.12;`

## Notes

The code compliles the contracts, which will typically provide extensive debug information and warnings/errors, and also checks for contract size and inline assembly. The second contract size check is superfluous, since the compiler already does that.

## Deploying Contracts to LOCAL TEST SYSTEM

First, make sure you have a `.env` in `packages/omgx/contracts-analyzer`. The five TEST_PRIVATE_KEYs are used for testing. The DEPLOYER_PRIVATE_KEY is used to deploy contracts. Since (right now) we don't charge oWETH to deploy contracts on L2, the wallets do not need any funds in them.

```javascript
L2_NODE_WEB3_URL=http://localhost:8545
DEPLOYER_PRIVATE_KEY=0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6
TEST_PRIVATE_KEY_1=0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6
TEST_PRIVATE_KEY_2=0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a
TEST_PRIVATE_KEY_3=0x8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba
TEST_PRIVATE_KEY_4=0x92db14e403b83dfe3df233f83dfa3a0d7096f21ca9b0d6d6b8d88b2b4ec1564e
TEST_PRIVATE_KEY_5=0x4bbbf85ce3377467afe5d46f804f221813b2bb87f24d81f60f1fcdbf7cbf4356
```

Then, deploy:

```bash
yarn deploy
```

You should then see a list of contract addresses.

## Deploying Contracts to OMGX RINKEBY

Make sure you have a `.env` in `packages/omgx/contracts-analyzer`. Then, set your variables similar to this:

```javascript
L2_NODE_WEB3_URL=https://rinkeby.omgx.network
DEPLOYER_PRIVATE_KEY=0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6
TEST_PRIVATE_KEY_1=0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6
TEST_PRIVATE_KEY_=0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a
TEST_PRIVATE_KEY_3=0x8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba
TEST_PRIVATE_KEY_4=0x92db14e403b83dfe3df233f83dfa3a0d7096f21ca9b0d6d6b8d88b2b4ec1564e
TEST_PRIVATE_KEY_5=0x4bbbf85ce3377467afe5d46f804f221813b2bb87f24d81f60f1fcdbf7cbf4356
```

Next, comment out `{gasPrice: 0, gasLimit: 800000}` in three places - they are not needed for now:

```
GitHub/optimism/packages/omgx/contracts-analyzer/scripts/deploy.js:
   66      // Transfer Sushi Ownership to Chef
   67      console.log(" 🔑 Transfer Sushi Ownership to Chef")
   68:     await (await SushiToken.transferOwnership(MasterChef.address, { gasLimit: 800000, gasPrice: 0 })).wait()
   69    }
   70  
   ..
   72      // Transfer ownership of MasterChef to Dev
   73      console.log(" 🔑 Transfer ownership of MasterChef to Dev")
   74:     await (await MasterChef.transferOwnership(deployAddress, { gasLimit: 800000, gasPrice: 0 })).wait()
   75    }
   76  

GitHub/optimism/packages/omgx/contracts-analyzer/scripts/utils.js:
   30    
   31    const nonce = await signerProvider.getTransactionCount()
   32:   const deployed = await contractArtifacts.deploy(...contractArgs, { nonce, ...overrides, gasPrice: 0, gasLimit: 800000 });
   33    await deployed.deployTransaction.wait()
   34  
```

Finally, deploy:

```bash
yarn deploy
```

You should then see a list of contract addresses.

## Testing

```bash
yarn test
```

If you get `Error: insufficient funds for intrinsic transaction cost`, this means you need to explicitely set `{gasPrice: 0, gasLimit: 800000}` when you call contracts, such as in `await this.Factory__MasterChef.deploy`.
