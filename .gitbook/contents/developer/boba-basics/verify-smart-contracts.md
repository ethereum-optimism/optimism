---
description: Learn how to verify smart contracts on Boba
---

# Verifying Contracts

The Boba Explorers support verifying smart contracts via the [hardhat-etherscan](https://hardhat.org/hardhat-runner/plugins/nomiclabs-hardhat-etherscan#hardhat-etherscan) plug

<figure><img src="../../../assets/verifying contracts with hardhat.png" alt=""><figcaption></figcaption></figure>

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
    'boba-mainnet': {
      url: 'https://mainnet.boba.network',
    },
    bobabnb: {
      url: 'https://bnb.boba.network',
    },
  },
  etherscan: {
    apiKey: {
      'boba-mainnet': process.env.BOBA_MAINNET_KEY,
      bobabnb: 'NO_KEY_REQUIRED',
    },
     customChains: [
      {
        network: 'boba-mainnet',
        chainId: 288,
        urls: {
          apiURL: 'https://api.routescan.io/v2/network/mainnet/evm/288/etherscan',
          browserURL: 'https://bobascan.com',
        },
      },
      {
        network: 'bobabnb',
        chainId: 56288,
        urls: {
          apiURL: 'https://api.routescan.io/v2/network/mainnet/evm/56288/etherscan',
          browserURL: 'https://bobascan.com',
        },
      },
    ],
  },
  }
};
```

Lastly, run the `verify` task, passing the address of the contract, the network where it's deployed, and the constructor arguments that were used to deploy it (if any):

```bash
npx hardhat verify --network mainnet DEPLOYED_CONTRACT_ADDRESS "Constructor argument 1" "Constructor argument 2"
```

<figure><img src="../../.gitbook/assets/wefgwefgerfg.png" alt=""><figcaption></figcaption></figure>
