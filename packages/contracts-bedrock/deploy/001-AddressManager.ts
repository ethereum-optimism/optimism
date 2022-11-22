import { DeployFunction } from 'hardhat-deploy/dist/types'

import {
  assertContractVariable,
  deployAndVerifyAndThen,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  await deployAndVerifyAndThen({
    hre,
    name: 'Lib_AddressManager',
    contract: 'AddressManager',
    args: [],
    postDeployAction: async (contract) => {
      // Owner is temporarily set to the deployer.
      await assertContractVariable(contract, 'owner', deployer)
    },
  })
}

deployFn.tags = ['AddressManager']

export default deployFn
