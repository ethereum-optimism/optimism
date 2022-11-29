import { ethers } from 'ethers'
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

  let controller = hre.deployConfig.controller
  if (controller === ethers.constants.AddressZero) {
    if (hre.network.config.live === false) {
      console.log(`WARNING!!!`)
      console.log(`WARNING!!!`)
      console.log(`WARNING!!!`)
      console.log(`WARNING!!! A controller address was not provided.`)
      console.log(
        `WARNING!!! Make sure you are ONLY doing this on a test network.`
      )
      controller = deployer
    } else {
      throw new Error(
        `controller address MUST NOT be the deployer on live networks`
      )
    }
  }

  let finalOwner = hre.deployConfig.finalSystemOwner
  if (finalOwner === ethers.constants.AddressZero) {
    if (hre.network.config.live === false) {
      console.log(`WARNING!!!`)
      console.log(`WARNING!!!`)
      console.log(`WARNING!!!`)
      console.log(`WARNING!!! A proxy admin owner address was not provided.`)
      console.log(
        `WARNING!!! Make sure you are ONLY doing this on a test network.`
      )
      finalOwner = deployer
    } else {
      throw new Error(`must specify the finalSystemOwner on live networks`)
    }
  }

  const config = await makeDictatorConfig(hre, controller, finalOwner, false)
  await deployAndVerifyAndThen({
    hre,
    name: 'MigrationSystemDictator',
    args: [config],
    postDeployAction: async (contract) => {
      await assertDictatorConfig(contract, config)
    },
  })
}

deployFn.tags = ['MigrationSystemDictator']

export default deployFn
