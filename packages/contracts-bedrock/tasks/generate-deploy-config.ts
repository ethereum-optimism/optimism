import fs from 'fs'
import path from 'path'

import { task } from 'hardhat/config'
import { HardhatRuntimeEnvironment } from 'hardhat/types'

task(
  'generate-deploy-config',
  'generates a json config file for the current network'
).setAction(async ({}, hre: HardhatRuntimeEnvironment) => {
  try {
    const base = path.join(hre.config.paths.deployConfig, hre.network.name)
    if (fs.existsSync(`${base}.ts`)) {
      // eslint-disable-next-line @typescript-eslint/no-var-requires
      const config = require(`${base}.ts`).default
      fs.writeFileSync(`${base}.json`, JSON.stringify(config, null, 2), 'utf8')
    } else {
      throw new Error('not found')
    }
  } catch (err) {
    throw new Error(
      `error while loading deploy config for network: ${hre.network.name}, ${err}`
    )
  }
})
