/* Imports: External */
import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'

task('deploy', 'Deploy contracts to L1 and L2')
  .setAction(async (args, hre: any, runSuper) => {
    hre.deployConfig = args
    return runSuper(args)
  })
