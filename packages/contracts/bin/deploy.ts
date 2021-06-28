import { Wallet } from 'ethers'
import path from 'path'
import dirtree from 'directory-tree'
import fs from 'fs'

// Ensures that all relevant environment vars are properly set. These lines *must* come before the
// hardhat import because importing will load the config (which relies on these vars). Necessary
// because CI currently uses different var names than the ones we've chosen here.
// TODO: Update CI so that we don't have to do this anymore.
process.env.HARDHAT_NETWORK = 'custom' // "custom" here is an arbitrary name. only used for CI.
process.env.CONTRACTS_TARGET_NETWORK = 'custom'
process.env.CONTRACTS_DEPLOYER_KEY = process.env.DEPLOYER_PRIVATE_KEY
process.env.CONTRACTS_RPC_URL =
  process.env.L1_NODE_WEB3_URL || 'http://127.0.0.1:8545'

import hre from 'hardhat'

const sequencer = new Wallet(process.env.SEQUENCER_PRIVATE_KEY)
const deployer = new Wallet(process.env.DEPLOYER_PRIVATE_KEY)

const parseEnv = () => {
  const ensure = (env, type) => {
    if (typeof process.env[env] === 'undefined') {
      return undefined
    }
    if (type === 'number') {
      return parseInt(process.env[env], 10)
    }
    return process.env[env]
  }

  return {
    l1BlockTimeSeconds: ensure('BLOCK_TIME_SECONDS', 'number'),
    ctcForceInclusionPeriodSeconds: ensure('FORCE_INCLUSION_PERIOD_SECONDS', 'number'),
    ctcMaxTransactionGasLimit: ensure('MAX_TRANSACTION_GAS_LIMIT', 'number'),
    emMinTransactionGasLimit: ensure('MIN_TRANSACTION_GAS_LIMIT', 'number'),
    emMaxtransactionGasLimit: ensure('MAX_TRANSACTION_GAS_LIMIT', 'number'),
    emMaxGasPerQueuePerEpoch: ensure('MAX_GAS_PER_QUEUE_PER_EPOCH', 'number'),
    emSecondsPerEpoch: ensure('ECONDS_PER_EPOCH', 'number'),
    emOvmChainId: ensure('CHAIN_ID', 'number'),
    sccFraudProofWindow: ensure('FRAUD_PROOF_WINDOW_SECONDS', 'number'),
    sccSequencerPublishWindow: ensure('SEQUENCER_PUBLISH_WINDOW_SECONDS', 'number'),
  }
}

const main = async () => {
  const config = parseEnv()

  await hre.run('deploy', {
    l1BlockTimeSeconds: config.l1BlockTimeSeconds,
    ctcForceInclusionPeriodSeconds: config.ctcForceInclusionPeriodSeconds,
    ctcMaxTransactionGasLimit: config.ctcMaxTransactionGasLimit,
    emMinTransactionGasLimit: config.emMinTransactionGasLimit,
    emMaxtransactionGasLimit: config.emMaxtransactionGasLimit,
    emMaxGasPerQueuePerEpoch: config.emMaxGasPerQueuePerEpoch,
    emSecondsPerEpoch: config.emSecondsPerEpoch,
    emOvmChainId: config.emOvmChainId,
    sccFraudProofWindow: config.sccFraudProofWindow,
    sccSequencerPublishWindow: config.sccFraudProofWindow,
    ovmSequencerAddress: sequencer.address,
    ovmProposerAddress: sequencer.address,
    ovmRelayerAddress: sequencer.address,
    ovmAddressManagerOwner: deployer.address,
    noCompile: process.env.NO_COMPILE ? true : false,
  })

  // Stuff below this line is currently required for CI to work properly. We probably want to
  // update our CI so this is no longer necessary. But I'm adding it for backwards compat so we can
  // get the hardhat-deploy stuff merged. Woot.
  const nicknames = {
    'Lib_AddressManager': 'AddressManager',
    'mockOVM_BondManager': 'OVM_BondManager'
  }

  const contracts: any = dirtree(
    path.resolve(__dirname, `../deployments/custom`)
  ).children.filter((child) => {
    return child.extension === '.json'
  }).reduce((contractsAccumulator, child) => {
    const contractName = child.name.replace('.json', '')
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    const artifact = require(path.resolve(__dirname, `../deployments/custom/${child.name}`))
    contractsAccumulator[nicknames[contractName] || contractName] = artifact.address
    return contractsAccumulator
  }, {})

  contracts.OVM_Sequencer = await sequencer.getAddress()
  contracts.Deployer = await deployer.getAddress()

  const addresses = JSON.stringify(contracts, null, 2)
  const dumpsPath = path.resolve(__dirname, "../dist/dumps")
  if (!fs.existsSync(dumpsPath)) {
    fs.mkdirSync(dumpsPath)
  }
  const addrsPath = path.resolve(dumpsPath, 'addresses.json')
  fs.writeFileSync(addrsPath, addresses)
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.log(
      JSON.stringify({ error: error.message, stack: error.stack }, null, 2)
    )
    process.exit(1)
  })
