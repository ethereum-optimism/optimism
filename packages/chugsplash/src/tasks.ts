import path from 'path'
import { task } from 'hardhat/config'
import { ethers } from 'ethers'
import { toRpcHexString } from '@eth-optimism/core-utils'
import {
  parseChugSplashConfig,
  validateChugSplashConfig,
  makeActionBundleFromConfig,
} from './config'
import { fromRawChugSplashAction, isSetStorageAction } from './actions'
import { ChugSplashProxy } from './ifaces'

const TASK_CHUGSPLASH_DEPLOY = 'chugsplash:deploy'
const TASK_CHUGSPLASH_CHECK = 'chugsplash:check'
const TASK_CHUGSPLASH_VERIFY = 'chugsplash:verify'

task(TASK_CHUGSPLASH_CHECK)
  .setDescription('Checks if a given deployment file is correctly formatted')
  .addParam('deployConfig', 'path to chugsplash deploy config')
  .setAction(async (args: { deployConfig: string }) => {
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    const config = require(path.resolve(args.deployConfig)).default
    try {
      validateChugSplashConfig(config)
    } catch (err) {
      console.log(err)
    }
  })

task(TASK_CHUGSPLASH_VERIFY)
  .setDescription('Checks if deployment config matches the actual deployment')
  .addParam('deployConfig', 'path to chugsplash deploy config')
  .setAction(async (args: { deployConfig: string }, hre) => {
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    const config = require(path.resolve(args.deployConfig)).default
    const bundle = await makeActionBundleFromConfig(config, process.env)
    for (const action of bundle.actions) {
      const parsedAction = fromRawChugSplashAction(action.action)
      if (isSetStorageAction(parsedAction)) {
        const storage = await hre.network.provider.send('eth_getStorageAt', [
          parsedAction.target,
          toRpcHexString(ethers.BigNumber.from(parsedAction.key)),
          'latest',
        ])

        // TODO: Figure out how to best present this information
        if (storage !== parsedAction.value) {
          throw new Error(
            `Storage at ${parsedAction.target} does not match expected storage`
          )
        }
      } else {
        const proxy = new ethers.Contract(
          parsedAction.target,
          ChugSplashProxy,
          hre.network.provider as any
        )

        const implementation = await proxy.getImplementation({
          from: ethers.constants.AddressZero,
        })

        const code = await hre.network.provider.send('eth_getCode', [
          implementation,
          'latest',
        ])

        // TODO: Figure out how to best present this information
        if (code !== parsedAction.code) {
          throw new Error(
            `Code at ${parsedAction.target} does not match expected code`
          )
        }
      }
    }
  })

task(TASK_CHUGSPLASH_DEPLOY)
  .setDescription('Deploys a system based on the given deployment file')
  .addParam('deployConfig', 'path to chugsplash deploy config')
  .setAction(async (args: { deployConfig: string }) => {
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    const config = require(path.resolve(args.deployConfig)).default
    console.log(
      JSON.stringify(parseChugSplashConfig(config, process.env), null, 2)
    )
  })
