import { HardhatUserConfig } from 'hardhat/types'

// Hardhat plugins
import '@nomiclabs/hardhat-ethers'
import '@eth-optimism/hardhat-ovm'
//import 'hardhat-typechain'

const config: HardhatUserConfig = {
  mocha: {
    timeout: 300000,
  },
  networks: {
    omgx: {
      url: 'http://localhost:8545', //never is actually used - set by the .env
      ovm: true,
    },
  },
  solidity: '0.7.6',
  ovm: {
    solcVersion: '0.7.6',
  }
}

export default config
