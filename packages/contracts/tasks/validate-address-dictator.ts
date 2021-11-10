'use strict'

import { ethers } from 'ethers'
import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'
import { hexStringEquals } from '@eth-optimism/core-utils'
import { getContractFactory } from '../src/contract-defs'

import { getInput, color as c } from '../src/task-utils'

const printComparison = (
  action: string,
  description: string,
  value1: string,
  value2: string
) => {
  console.log(action + ':')
  if (hexStringEquals(value1, value2)) {
    console.log(c.green(`${description} looks good! 😎`))
  } else {
    throw new Error(`${description} looks wrong`)
  }
  console.log() // Add some whitespace
}

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
    if (!process.env.CONTRACTS_RPC_URL) {
      throw new Error(c.red('CONTRACTS_RPC_URL not set in your env.'))
    }
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

    // Improvement: use the artifact in deployments/${network.name}
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    const dictatorArtifact = require('../artifacts/contracts/L1/deployment/AddressDictator.sol/AddressDictator.json')

    const dictatorCode = await provider.getCode(args.dictator)
    printComparison(
      'Verifying AddressDictator source code against local build artifacts',
      'Deployed AddressDictator code',
      dictatorArtifact.deployedBytecode,
      dictatorCode
    )

    const dictatorContract = getContractFactory('AddressDictator')
      .attach(args.dictator)
      .connect(provider)

    const finalOwner = await dictatorContract.finalOwner()
    printComparison(
      'Validating that finalOwner address in the dictator matches multisig address',
      'finalOwner',
      finalOwner,
      args.multisig
    )

    const manager = await dictatorContract.manager()
    printComparison(
      'Validating the AddressManager address in the dictator',
      'addressManager',
      manager,
      args.manager
    )

    // TODO:
    // Get names and addresses from the Dictator.
    // const namedAddresses = Array<{ name: string; addr: string }> =
    //   await dictatorContract.getNamedAddresses()
    // for (const pair of namedAddresses) {
    //   import dictatorArtifact from '../artifacts/contracts/L1/deployment/AddressDictator.sol/AddressDictator.json'
    //   const dictatorCode = await provider.getCode(args.dictator)
    //   printComparison(
    //     'Verifying AddressDictator source code against local build artifacts',
    //     'Deployed AddressDictator code',
    //     dictatorArtifact.deployedBytecode,
    //     dictatorCode
    //   )
    // }

    // Loop over those and compare the addresses/deployedBytecode to deployment artifacts.
    // Verify libAddressManager where applicable.
    // Verify other values in Post-deployment contracts checklist
  })
