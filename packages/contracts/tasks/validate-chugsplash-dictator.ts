'use strict'

import { ethers } from 'ethers'
import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'
import { getContractFactory, getContractDefinition } from '../src/contract-defs'

import {
  getInput,
  color as c,
  getEtherscanUrl,
  printComparison,
  printSectionHead,
} from '../src/validation-utils'

task('validate:chugsplash-dictator')
  .addParam(
    'dictator',
    'Address of the ChugSplashDictator to validate.',
    undefined,
    types.string
  )
  .addParam(
    'proxy',
    'Address of the L1ChugSplashProxy to validate.',
    undefined,
    types.string
  )
  .addParam(
    'multisig',
    'Address of the multisig contract which should be the final owner',
    undefined,
    types.string
  )
  .addOptionalParam(
    'contractsRpcUrl',
    'RPC Endpoint to query for data',
    process.env.CONTRACTS_RPC_URL,
    types.string
  )
  .setAction(async (args) => {
    if (!args.contractsRpcUrl) {
      throw new Error(
        c.red('RPC URL must be set in your env, or passed as an argument.')
      )
    }
    const provider = new ethers.providers.JsonRpcProvider(args.contractsRpcUrl)

    const network = await provider.getNetwork()
    console.log() // the whitespacooooooor
    console.log(c.cyan("First make sure you're on the right chain:"))
    console.log(
      `Reading from the ${c.red(network.name)} network (Chain ID: ${c.red(
        '' + network.chainId
      )})`
    )
    await getInput(c.yellow('OK? Hit enter to continue.'))
    console.log()

    const dictatorArtifact = getContractDefinition('ChugSplashDictator')
    const dictatorCode = await provider.getCode(args.dictator)
    printSectionHead(
      `Validate the Chugsplash Dictator deployment at\n${getEtherscanUrl(
        network,
        args.dictator
      )}`
    )
    printComparison(
      'Compare the deployed ChugSplashDictator bytecode against local build artifacts',
      'Deployed ChugSplashDictator code',
      { name: 'Compiled bytecode', value: dictatorArtifact.deployedBytecode },
      { name: 'Deployed bytecode', value: dictatorCode }
    )
    await getInput(c.yellow('OK? Hit enter to continue.'))
    console.log()

    console.log(
      c.cyan("The next 4 checks will validate the ChugSplashDictator's config")
    )
    // Connect to the deployed ChugSplashDictator.
    const dictatorContract = getContractFactory('ChugSplashDictator')
      .attach(args.dictator)
      .connect(provider)
    const finalOwner = await dictatorContract.finalOwner()
    printComparison(
      'Compare the finalOwner address in the ChugSplashDictator to the multisig address',
      'finalOwner',
      { name: 'multisig address', value: args.multisig },
      { name: 'finalOwner      ', value: finalOwner }
    )
    await getInput(c.yellow('OK? Hit enter to continue.'))
    console.log()

    const dictatorMessengerSlotKey = await dictatorContract.messengerSlotKey()
    const dictatorMessengerSlotVal = await dictatorContract.messengerSlotVal()
    const proxyMessengerSlotVal = await provider.getStorageAt(
      args.proxy,
      dictatorMessengerSlotKey
    )
    printComparison(
      'Compare the Messenger slot key/value to be set, with the current values in the proxy',
      `Storage slot key ${dictatorMessengerSlotKey}`,
      {
        name: `Value in the proxy at slot key\n${dictatorMessengerSlotKey}`,
        value: proxyMessengerSlotVal,
      },
      {
        name: `Dictator will setStorage at slot key\n${dictatorMessengerSlotKey}`,
        value: dictatorMessengerSlotVal,
      }
    )
    await getInput(c.yellow('OK? Hit enter to continue.'))
    console.log()

    const dictatorBridgeSlotKey = await dictatorContract.bridgeSlotKey()
    const dictatorBridgeSlotVal = await dictatorContract.bridgeSlotVal()
    const proxyBridgeSlotVal = await provider.getStorageAt(
      args.proxy,
      dictatorBridgeSlotKey
    )
    printComparison(
      'Compare the Bridge slot key/value to be set, with the current values in the proxy',
      `Storage slot key ${dictatorBridgeSlotKey}`,
      {
        name: `Value currently in the proxy at slot key\n${dictatorBridgeSlotKey}`,
        value: proxyBridgeSlotVal,
      },
      {
        name: `Dictator will setStorage in the proxy at slot key\n${dictatorBridgeSlotKey}`,
        value: dictatorBridgeSlotVal,
      }
    )
    await getInput(c.yellow('OK? Hit enter to continue.'))
    console.log()

    const bridgeArtifact = getContractDefinition('L1StandardBridge')
    const expectedCodeHash = ethers.utils.keccak256(
      bridgeArtifact.deployedBytecode
    )
    const actualCodeHash = await dictatorContract.codeHash()
    printComparison(
      "Compare the Dictator's codeHash against hash of the local L1StandardBridge build artifacts",
      "Dictator's codeHash",
      {
        name: 'Expected codeHash',
        value: expectedCodeHash,
      },
      {
        name: 'Actual codeHash',
        value: actualCodeHash,
      }
    )
    await getInput(c.yellow('OK? Hit enter to continue.'))
    console.log()
    console.log(c.green('Chugsplash Dictator Validation complete!'))
  })
