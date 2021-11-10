'use strict'

import { ethers } from 'ethers'
import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'
import { hexStringEquals } from '@eth-optimism/core-utils'
import { getContractFactory } from '../src/contract-defs'
import { getDeployedContractArtifact } from '../src/contract-deployed-artifacts'
import { getInput, color as c } from '../src/task-utils'

task('validate:address-dictator')
  // Provided by the signature Requestor
  .addParam(
    'dictator',
    'Address of the AddressDictator to validate.',
    undefined,
    types.string
  )
  // I'm not certain if this value should be entered manually or read from the artifacts,
  // but I think it's best if the
  .addParam(
    'manager',
    'Address of the Address Manager contract which would be updated',
    undefined,
    types.string
  )
  // Provided by the signers themselves.
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
    const provider = new ethers.providers.JsonRpcProvider(args.contractsRpcUrl)

    const network = await provider.getNetwork()
    console.log(
      `
Validating the deployment on the chain with:
Name: ${network.name}
Chain ID: ${network.chainId}`
    )
    const res = await getInput(c.yellow('Does that look right? (LGTM/n)\n> '))
    if (res !== 'LGTM') {
      throw new Error(
        c.red('User indicated that validation was run against the wrong chain')
      )
    }
    console.log()

    console.log(
      'Verifying AddressDictator source code against local artifacts:'
    )
    const dictatorArtifact = getDeployedContractArtifact(
      'AddressDictator',
      'kovan'
    )

    const dictatorCode = await provider.getCode(args.dictator)
    if (hexStringEquals(dictatorArtifact.deployedBytecode, dictatorCode)) {
      console.log(c.green('Deployed dictator code Looks good! 😎'))
    } else {
      throw new Error('Deployed AddressDictator code looks wrong')
    }
    console.log()

    const dictatorContract = getContractFactory('AddressDictator')
      .attach(args.dictator)
      .connect(provider)

    console.log('Validating the finalOwner address in the dictator:')
    const finalOwner = await dictatorContract.finalOwner()
    if (hexStringEquals(finalOwner, args.multisig)) {
      console.log(c.green('finalOwner Looks good! 😎'))
    } else {
      throw new Error('finalOwner looks wrong')
    }
    console.log()

    console.log('Validating the AddressManager address in the dictator:')
    const manager = await dictatorContract.manager()
    if (hexStringEquals(manager, args.manager)) {
      console.log(c.green('manager Looks good! 😎'))
    } else {
      throw new Error('manager looks wrong')
    }
    console.log()
  })
