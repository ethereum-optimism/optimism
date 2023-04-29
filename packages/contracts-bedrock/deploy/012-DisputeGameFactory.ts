import { DeployFunction } from 'hardhat-deploy/dist/types'

import { assertContractVariable, deploy } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  // Deploy the dispute game factory implementation contract.
  // The owner is set to the deployer and transferred to the
  // SystemDictator before we trigger the dictator steps.
  await deploy({
    hre,
    name: 'DisputeGameFactoryImpl',
    args: [deployer],
    postDeployAction: async (contract) => {
      await assertContractVariable(contract, 'owner', deployer)
    },
  })
}

deployFn.tags = ['DisputeGameFactoryImpl', 'setup', 'l1']

export default deployFn
