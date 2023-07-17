import { ethers } from 'ethers'

import { FaucetModuleConfigs, Time } from '../../src/config/faucet'

const config: FaucetModuleConfigs = {
  GithubFam: {
    address: '0xa8Dc386a773Fb15dC564a0d9Ea1944036F05F1D0',
    ttl: 2 * Time.MINUTE,
    amount: ethers.utils.parseEther('.05'),
    name: 'GITHUB_ADMIN_FAM',
    enabled: true,
  },
  OptimistFam: {
    address: '0x610745dC6728a5311A06c365a5F6DC65E637Eb6f',
    ttl: 2 * Time.MINUTE,
    amount: ethers.utils.parseEther('1.0'),
    name: 'OPTIMIST_ADMIN_FAM',
    enabled: true,
  },
}

export default config
