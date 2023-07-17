import { ethers } from 'ethers'

import { FaucetModuleConfigs, Time } from '../../src/config/faucet'

const config: FaucetModuleConfigs = {
  GithubFam: {
    address: '0x95bd6dEA0DdbbD4D0EA35d3F491C4FCd9B416E2c',
    ttl: 2 * Time.MINUTE,
    amount: ethers.utils.parseEther('.05'),
    name: 'GITHUB_ADMIN_FAM',
    enabled: true,
  },
  OptimistFam: {
    address: '0x1bE1E2cAC524556Bf25Fa1f6283A4ba9d89cc4bE',
    ttl: 2 * Time.MINUTE,
    amount: ethers.utils.parseEther('1.0'),
    name: 'OPTIMIST_ADMIN_FAM',
    enabled: true,
  },
}

export default config
