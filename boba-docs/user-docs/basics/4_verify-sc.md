---
description: Learn how to verify smart contracts on Boba
---

# Verifying Contracts

The Boba Explorers support verifying smart contracts via the [hardhat-etherscan](https://hardhat.org/hardhat-runner/plugins/nomiclabs-hardhat-etherscan#hardhat-etherscan) plug



## Verifying Contracts with Hardhat

### Installation

```bash
npm install --save-dev @nomiclabs/hardhat-etherscan
```

And add the following statement to your `hardhat.config.js`:

```js
require("@nomiclabs/hardhat-etherscan");
```

Or, if you are using TypeScript, add this to your `hardhat.config.ts`:

```js
import "@nomiclabs/hardhat-etherscan";
```

### Usage

You need to add the following Etherscan config to your `hardhat.config.js` file:

```js
module.exports = {
  networks: {
    boba_eth_mainnet: {
      url: process.env.LIGHTBRIDGE_RPC_BOBAETHMAINNET ?? 'https://mainnet.boba.network',
    },
    boba_sepolia: {
      url: 'https://sepolia.boba.network',
    },
  },
  etherscan: {
    apiKey: {
      boba_eth_mainnet: "boba", // not required, set placeholder
      boba_sepolia: "boba", // not required, set placeholder
    },
     customChains: [
       {
         network: "boba_eth_mainnet",
         chainId: 288,
         urls: {
           apiURL: "https://api.routescan.io/v2/network/mainnet/evm/288/etherscan",
           browserURL: "https://bobascan.com"
         },
       },
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
};
```

Lastly, run the `verify` task, passing the address of the contract, the network where it's deployed, and the constructor arguments that were used to deploy it (if any):

```bash
npx hardhat verify --network mainnet DEPLOYED_CONTRACT_ADDRESS "Constructor argument 1" "Constructor argument 2"
```

---

Alternatively you may want to use [Sourcify](https://sourcify.dev/) to verify your contracts.

