/* Imports: External */
import { ethers } from 'ethers'
import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'

const DEFAULT_L1_BLOCK_TIME_SECONDS = 15
const DEFAULT_CTC_FORCE_INCLUSION_PERIOD_SECONDS = 60 * 60 * 24 * 30 // 30 days
const DEFAULT_CTC_MAX_TRANSACTION_GAS_LIMIT = 11_000_000
const DEFAULT_SCC_FRAUD_PROOF_WINDOW = 60 * 60 * 24 * 7 // 7 days
const DEFAULT_SCC_SEQUENCER_PUBLISH_WINDOW = 60 * 30 // 30 minutes

task('deploy')
  .addOptionalParam(
    'l1BlockTimeSeconds',
    'Number of seconds on average between every L1 block.',
    DEFAULT_L1_BLOCK_TIME_SECONDS,
    types.int
  )
  .addOptionalParam(
    'ctcForceInclusionPeriodSeconds',
    'Number of seconds that the sequencer has to include transactions before the L1 queue.',
    DEFAULT_CTC_FORCE_INCLUSION_PERIOD_SECONDS,
    types.int
  )
  .addOptionalParam(
    'ctcMaxTransactionGasLimit',
    'Max gas limit for L1 queue transactions.',
    DEFAULT_CTC_MAX_TRANSACTION_GAS_LIMIT,
    types.int
  )
  .addOptionalParam(
    'sccFraudProofWindow',
    'Number of seconds until a transaction is considered finalized.',
    DEFAULT_SCC_FRAUD_PROOF_WINDOW,
    types.int
  )
  .addOptionalParam(
    'sccSequencerPublishWindow',
    'Number of seconds that the sequencer is exclusively allowed to post state roots.',
    DEFAULT_SCC_SEQUENCER_PUBLISH_WINDOW,
    types.int
  )
  .addOptionalParam(
    'ovmSequencerAddress',
    'Address of the sequencer. Must be provided or this deployment will fail.',
    undefined,
    types.string
  )
  .addOptionalParam(
    'ovmProposerAddress',
    'Address of the account that will propose state roots. Must be provided or this deployment will fail.',
    undefined,
    types.string
  )
  .addOptionalParam(
    'ovmAddressManagerOwner',
    'Address that will own the Lib_AddressManager. Must be provided or this deployment will fail.',
    undefined,
    types.string
  )
  .setAction(async (args, hre: any, runSuper) => {
    // Necessary because hardhat doesn't let us attach non-optional parameters to existing tasks.
    const validateAddressArg = (argName: string) => {
      if (args[argName] === undefined) {
        throw new Error(
          `argument for ${argName} is required but was not provided`
        )
      }
      if (!ethers.utils.isAddress(args[argName])) {
        throw new Error(
          `argument for ${argName} is not a valid address: ${args[argName]}`
        )
      }
    }

    validateAddressArg('ovmSequencerAddress')
    validateAddressArg('ovmProposerAddress')
    validateAddressArg('ovmAddressManagerOwner')

    args.ctcForceInclusionPeriodBlocks = Math.floor(
      args.ctcForceInclusionPeriodSeconds / args.l1BlockTimeSeconds
    )

    hre.deployConfig = args
    return runSuper(args)
  })
