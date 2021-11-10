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

    // const dictatorArtifact = require('../artifacts/contracts/L1/deployment/AddressDictator.sol/AddressDictator.json')
    // eslint-disable-next-line @typescript-eslint/no-var-requires
    const dictatorArtifact = require(`../deployments/${network.name}/AddressDictator.json`)
    const dictatorCode = await provider.getCode(args.dictator)
    printComparison(
      'Verifying AddressDictator source code against local build artifacts',
      'Deployed AddressDictator code',
      dictatorArtifact.deployedBytecode,
      dictatorCode
    )

    // connect to the deployed AddressDictator
    const dictatorContract = getContractFactory('AddressDictator')
      .attach(args.dictator)
      .connect(provider)

    const finalOwner = await dictatorContract.finalOwner()
    printComparison(
      'Validating that finalOwner address in the AddressDictator matches multisig address',
      'finalOwner',
      finalOwner,
      args.multisig
    )

    const manager = await dictatorContract.manager()
    printComparison(
      'Validating the AddressManager address in the AddressDictator',
      'addressManager',
      manager,
      args.manager
    )

    // Get names and addresses from the Dictator.
    const namedAddresses = await dictatorContract.getNamedAddresses()

    // connect to the deployed AddressManager so we can see which are changed or unchanged.
    const managerContract = getContractFactory('Lib_AddressManager')
      .attach(args.manager)
      .connect(provider)
    // Loop over those and compare the addresses/deployedBytecode to deployment artifacts.
    for (const pair of namedAddresses) {
      // Check for addresses that will not be changed:
      const currentAddress = await managerContract.getAddress(pair.name)
      const addressChanged = !hexStringEquals(currentAddress, pair.addr)
      if (addressChanged) {
        console.log(`${pair.name} address will be updated.`)
        console.log(`Before ${currentAddress}`)
        console.log(`After ${pair.addr}`)

        // eslint-disable-next-line @typescript-eslint/no-var-requires
        const artifact = require(`../deployments/${network.name}/${pair.name}.json`)
        const code = await provider.getCode(pair.addr)
        printComparison(
          `Verifying ${pair.name} source code against local deployment artifacts`,
          `Deployed ${pair.name} code`,
          artifact.deployedBytecode,
          code
        )
      }
    }

    // Verify libAddressManager is set properly
    // Verify other values in Post-deployment contracts checklist
  })
