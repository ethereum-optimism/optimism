#!/usr/bin/env ts-node-script

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

const main = async () => {
  await hre.run('deploy', {
    l1BlockTimeSeconds: process.env.BLOCK_TIME_SECONDS,
    ctcForceInclusionPeriodSeconds: process.env.FORCE_INCLUSION_PERIOD_SECONDS,
    ctcMaxTransactionGasLimit: process.env.MAX_TRANSACTION_GAS_LIMIT,
    emMinTransactionGasLimit: process.env.MIN_TRANSACTION_GAS_LIMIT,
    emMaxtransactionGasLimit: process.env.MAX_TRANSACTION_GAS_LIMIT,
    emMaxGasPerQueuePerEpoch: process.env.MAX_GAS_PER_QUEUE_PER_EPOCH,
    emSecondsPerEpoch: process.env.SECONDS_PER_EPOCH,
    emOvmChainId: process.env.CHAIN_ID,
    sccFraudProofWindow: parseInt(process.env.FRAUD_PROOF_WINDOW_SECONDS, 10),
    sccSequencerPublishWindow: process.env.SEQUENCER_PUBLISH_WINDOW_SECONDS,
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
  }).reduce((contracts, child) => {
    const contractName = child.name.replace('.json', '')
    const artifact = require(path.resolve(__dirname, `../deployments/custom/${child.name}`))
    contracts[nicknames[contractName] || contractName] = artifact.address
    return contracts
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
