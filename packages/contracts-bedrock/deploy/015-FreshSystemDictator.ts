import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'

import {
  deployAndVerifyAndThen,
  assertDictatorConfig,
  makeDictatorConfig,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const config = await makeDictatorConfig(
    hre,
    deployer,
    hre.deployConfig.finalSystemOwner,
    true
  )
  await deployAndVerifyAndThen({
    hre,
    name: 'FreshSystemDictator',
    args: [config],
    postDeployAction: async (contract) => {
      await assertDictatorConfig(contract, config)
    },
  })
}

deployFn.tags = ['FreshSystemDictator', 'fresh']

export default deployFn
