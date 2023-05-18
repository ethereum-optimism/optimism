import { ethers } from 'ethers'

import { FaucetModuleConfigs, Time } from '../../src/config/faucet'

const config: FaucetModuleConfigs = {
  GithubFam: {
    authModuleDeploymentName: 'GithubAdminFaucetAuthModule',
    ttl: Time.DAY,
    amount: ethers.utils.parseEther('.05'),
    name: 'GITHUB_ADMIN_FAM',
    enabled: true,
  },
  OptimistFam: {
    authModuleDeploymentName: 'OptimistAdminFaucetAuthModule',
    ttl: Time.DAY,
    amount: ethers.utils.parseEther('1.0'),
    name: 'OPTIMIST_ADMIN_FAM',
    enabled: true,
  },
}

export default config
