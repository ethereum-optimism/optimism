# OVM Truffle Provider Wrapper
The OVM uses a specific `chainId`, which Truffle, at the moment, does not allow to be configured globally within a project, so this package simply wraps the provider that is used in order to set the `chainId` field on all transactions.

## Configuration
ChainId defaults to 108 but is configurable by setting the `OVM_CHAIN_ID` environment variable.

## Example Usage in truffle-config.js:
```$javascript
const HDWalletProvider = require('truffle-hdwallet-provider');
const ProviderWrapper = require("@eth-optimism/ovm-truffle-provider-wrapper");
const mnemonic = 'candy maple cake sugar pudding cream honey rich smooth crumble sweet treat';

module.exports = {
  networks: {
    test: {
      provider: function () {
        return ProviderWrapper.wrapProviderAndStartLocalNode(new HDWalletProvider(mnemonic, "http://127.0.0.1:8545/", 0, 10));
      },
    },
    live_example: {
      provider: function () {
        return ProviderWrapper.wrapProvider(new HDWalletProvider(mnemonic, "http://127.0.0.1:8545/", 0, 10));
      },
    },
  },
  compilers: {
    solc: {
      // Add path to the solc-transpiler
      version: "../../node_modules/@eth-optimism/solc-transpiler",
    }
  }
}
```