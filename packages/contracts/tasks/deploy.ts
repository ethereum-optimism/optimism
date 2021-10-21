/* Imports: External */
import { ethers } from 'ethers'
import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'

const DEFAULT_L1_BLOCK_TIME_SECONDS = 15
const DEFAULT_CTC_MAX_TRANSACTION_GAS_LIMIT = 11_000_000
const DEFAULT_CTC_L2_GAS_DISCOUNT_DIVISOR = 32
const DEFAULT_CTC_ENQUEUE_GAS_COST = 60_000
const DEFAULT_SCC_FRAUD_PROOF_WINDOW = 60 * 60 * 24 * 7 // 7 days
const DEFAULT_SCC_SEQUENCER_PUBLISH_WINDOW = 60 * 30 // 30 minutes
const DEFAULT_DEPLOY_CONFIRMATIONS = 12

task('deploy')
  // Rollup config options
  .addOptionalParam(
    'l1BlockTimeSeconds',
    'Number of seconds on average between every L1 block.',
    DEFAULT_L1_BLOCK_TIME_SECONDS,
    types.int
  )
  .addOptionalParam(
    'ctcMaxTransactionGasLimit',
    'Max gas limit for L1 queue transactions.',
    DEFAULT_CTC_MAX_TRANSACTION_GAS_LIMIT,
    types.int
  )
  .addOptionalParam(
    'ctcL2GasDiscountDivisor',
    'Max gas limit for L1 queue transactions.',
    DEFAULT_CTC_L2_GAS_DISCOUNT_DIVISOR,
    types.int
  )
  .addOptionalParam(
    'ctcEnqueueGasCost',
    'Max gas limit for L1 queue transactions.',
    DEFAULT_CTC_ENQUEUE_GAS_COST,
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
  // Permissioned address options
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
  // Reusable address options
  .addOptionalParam(
    'proxyL1CrossDomainMessenger',
    'Address of the L1CrossDomainMessenger Proxy, for use in a deployment which is keeping the existing contract.',
    undefined,
    types.string
  )
  .addOptionalParam(
    'proxyL1StandardBridge',
    'Address of the L1StandardBridge Proxy, for use in a deployment which is keeping the existing contract.',
    undefined,
    types.string
  )
  .addOptionalParam(
    'libAddressManager',
    'Address of the Lib_AddressManager, for use in a deployment which is keeping the existing contract.',
    undefined,
    types.string
  )
  .addOptionalParam(
    'numDeployConfirmations',
    'Number of confirmations to wait for each transaction in the deployment. More is safer.',
    DEFAULT_DEPLOY_CONFIRMATIONS,
    types.int
  )
  .addOptionalParam(
    'forked',
    'Enable this when using a forked network (use "true")',
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

    // validate potentially conflicting arguments
    const validateArgOrTag = (argName: string, tagName: string) => {
      // ensure that both an arg and tag were not provided for a given contract
      const hasTag = args.tags.includes(tagName)
      if (hasTag && ethers.utils.isAddress(args[argName])) {
        throw new Error(
          `cannot deploy a new ${tagName} if the address of an existing one is provided`
        )
      }
      // ensure that either a valid address is provided or we'll deploy a new one.
      try {
        validateAddressArg(argName)
        console.log(
          `Running deployments with the existing ${tagName} at ${args[argName]}`
        )
      } catch (error) {
        if (!hasTag) {
          throw new Error(
            `${error.message} \nmust either deploy a new ${tagName}, or provide the address for an existing one`
          )
        }
        console.log(`Running deployments with a new ${tagName}`)
      }
    }

    validateArgOrTag('libAddressManager', 'Lib_AddressManager')
    validateArgOrTag(
      'proxyL1CrossDomainMessenger',
      'Proxy__L1CrossDomainMessenger'
    )
    validateArgOrTag('proxyL1StandardBridge', 'Proxy__L1StandardBridge')

    hre.deployConfig = args
    return runSuper(args)
  })
