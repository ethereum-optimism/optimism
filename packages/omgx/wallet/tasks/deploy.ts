/* Imports: External */
import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'

const DEFAULT_EM_OVM_CHAIN_ID = 28

task('deploy', 'Deploy contracts to L1 and L2').addOptionalParam(
  'emOvmChainId',
  'Chain ID for the L2 network.',
  DEFAULT_EM_OVM_CHAIN_ID,
  types.int
).setAction(async (args, hre: any, runSuper) => {
    hre.deployConfig = args
    return runSuper(args)
  })

