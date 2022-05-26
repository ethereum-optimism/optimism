import path from 'path'
import fs from 'fs'

import { HardhatUserConfig, task, subtask } from 'hardhat/config'
import { TASK_COMPILE_SOLIDITY_GET_SOURCE_PATHS } from 'hardhat/builtin-tasks/task-names'
import '@nomiclabs/hardhat-waffle'
import '@typechain/hardhat'
import 'hardhat-gas-reporter'
import 'solidity-coverage'
import 'hardhat-deploy'
import '@foundry-rs/hardhat'

import './tasks/deposits'

subtask(TASK_COMPILE_SOLIDITY_GET_SOURCE_PATHS).setAction(
  async (_, __, runSuper) => {
    const paths = await runSuper()

    return paths.filter((p: string) => !p.endsWith('.t.sol'))
  }
)

task('accounts', 'Prints the list of accounts', async (_, hre) => {
  const accounts = await hre.ethers.getSigners()

  for (const account of accounts) {
    console.log(account.address)
  }
})

// TODO(tynes): migrate this functionality upstream
task('compile').setAction(async (taskArgs, hre, runSuper) => {
  await runSuper(taskArgs)

  const getAllFiles = function(directory: string, allFiles: Array<string> = []) {
    const files = fs.readdirSync(directory)

    for (const file of files) {
      const next = path.join(directory, file)
      if (fs.statSync(next).isDirectory()) {
        allFiles = getAllFiles(next, allFiles)
      } else {
        allFiles.push(next)
      }
    }
    return allFiles
  }

  // recursively get all of the source code and
  // get the relative paths to each file
  const allFiles = getAllFiles(hre.config.paths.sources)
    .map(f => path.relative(__dirname, f))

  // get the configured artifacts output path
  const artifactsPath = hre.config.paths.artifacts
  // get the absolute path to each foundry artifact
  const paths = await hre.artifacts.getArtifactPaths()
  for (const p of paths) {
    // skip tests
    if (p.includes('.t.sol')) {
      continue
    }

    // read the artifact path from the filesystem
    // put this in a try catch as it will fail if multiple
    // contracts are defined in the same file
    const info = path.parse(p)
    let artifact
    try {
      artifact = await hre.artifacts.readArtifact(info.name)
    } catch (e) {
      console.log(e)
      continue
    }

    // find the path to the source code
    let target
    for (const file of allFiles) {
      if (path.parse(file).base === path.parse(info.dir).base) {
        target = file
        break
      }
    }

    // unable to find the source code
    if (!target) {
      continue
    }

    // write the artifact
    const dir = path.join(artifactsPath, target)
    fs.mkdirSync(dir, {recursive: true})
    const out = path.join(dir, info.base)
    fs.writeFileSync(out, JSON.stringify(artifact, null, 2))
  }
});

const config: HardhatUserConfig = {
  networks: {
    devnetL1: {
      url: 'http://localhost:8545',
      accounts: [
        'ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80',
      ],
    },
  },
  gasReporter: {
    enabled: process.env.REPORT_GAS !== undefined,
    currency: 'USD',
  },
  namedAccounts: {
    deployer: {
      default: 0,
    },
  },
}

export default config
