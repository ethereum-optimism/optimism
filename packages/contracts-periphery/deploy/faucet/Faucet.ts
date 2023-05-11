/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { HardhatRuntimeEnvironment } from 'hardhat/types'
import '@nomiclabs/hardhat-ethers'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'
import { assertContractVariable } from '@eth-optimism/contracts-bedrock/src/deploy-utils'

import type { DeployConfig } from '../../src'

const deployFn: DeployFunction = async (hre: HardhatRuntimeEnvironment) => {
  const deployConfig = hre.deployConfig as DeployConfig

  const { deployer } = await hre.getNamedAccounts()
  console.log('Deploying Faucet')

  const { deploy } = await hre.deployments.deterministic('Faucet', {
    salt: hre.ethers.utils.solidityKeccak256(['string'], ['Faucet']),
    from: deployer,
    args: [deployConfig.faucetAdmin],
    log: true,
  })

  const result = await deploy()
  console.log(`Faucet deployed to ${result.address}`)

  const faucet = await hre.ethers.getContractAt('Faucet', result.address)
  await assertContractVariable(faucet, 'ADMIN', deployConfig.faucetAdmin)
}

deployFn.tags = ['Faucet', 'FaucetEnvironment']

export default deployFn
