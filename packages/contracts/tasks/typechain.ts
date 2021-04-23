/* External Imports */
import { task } from 'hardhat/config'
import { TASK_COMPILE } from 'hardhat/builtin-tasks/task-names'

task(TASK_COMPILE).setAction(async (args, hre: any, runSuper) => {
  if (hre.network.config.ovm && !hre.config.typechain.outDir.endsWith('-ovm')) {
    hre.config.typechain.outDir += '-ovm'
  }
  return runSuper(args)
})
