import '@eth-optimism/hardhat-ovm';
import '@eth-optimism/plugins/hardhat/compiler';
import '@eth-optimism/plugins/hardhat/ethers';
import '@nomiclabs/hardhat-ethers';
import '@nomiclabs/hardhat-waffle';
import "@tenderly/hardhat-tenderly";

const config = {
  mocha: {
    timeout: 60000,
  },
  networks: {
    omgx: {
      url: 'http://localhost:8545',
      // This sets the gas price to 0 for all transactions on L2. We do this
      // because account balances are not automatically initiated with an ETH
      // balance.
      gas: 10000000,
      gasPrice: 0,
      chainId: 28,
      ovm: true
    },
  },
  solidity: {
    compilers: [
      {
        version: "0.6.12",
        settings: {
          optimizer: {
            enabled: true,
            runs: 1
          }
        }
      },
    ],
  },
  ovm: {
    solcVersion: '0.6.12',
  },
}

export default config
