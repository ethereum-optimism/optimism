import { DeployFunction } from 'hardhat-deploy/dist/types'

import {
  assertContractVariable,
  deploy,
  getDeploymentAddress,
} from '../scripts/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const Lib_AddressManagerAddress = await getDeploymentAddress(
    hre,
    'Lib_AddressManager'
  )

  await deploy({
    hre,
    name: 'AddressDeprecator',
    args: [Lib_AddressManagerAddress],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'ADDRESS_MANAGER',
        Lib_AddressManagerAddress
      )
    },
  })
}

deployFn.tags = ['AddressDeprecator', 'setup', 'l1']

export default deployFn
