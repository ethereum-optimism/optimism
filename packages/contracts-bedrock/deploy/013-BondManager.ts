import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'

import {
  assertContractVariable,
  deploy,
  getContractFromArtifact,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const isLiveDeployer =
    deployer.toLowerCase() === hre.deployConfig.controller.toLowerCase()

  const DisputeGameFactoryProxy = await getContractFromArtifact(
    hre,
    'DisputeGameFactoryProxy'
  )

  // Deploy the BondManager implementation contract.
  await deploy({
    hre,
    name: 'BondManagerImpl',
    args: [ DisputeGameFactoryProxy.address ],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'DISPUTE_GAME_FACTORY',
        DisputeGameFactoryProxy.address
      )
    },
  })
}

deployFn.tags = ['BondManagerImpl', 'setup', 'l1']

export default deployFn
