'use strict'

import { ethers } from 'ethers'
import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'
import { hexStringEquals } from '@eth-optimism/core-utils'
import { getContractFactory } from '../src/contract-defs'

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

    console.log(
      'Validating the deployment on chain with properties:',
      await provider.getNetwork()
    )

    const dictatorContract = getContractFactory('AddressDictator')
      .attach(args.dictator)
      .connect(provider)

    console.log('Validating the finalOwner address in the dictator:')
    const finalOwner = await dictatorContract.finalOwner()
    if (hexStringEquals(finalOwner, args.multisig)) {
      console.log('LGTM')
    } else {
      console.log('finalOwner looks wrong')
    }

    console.log('Validating the AddressManager address in the dictator:')
    const manager = await dictatorContract.manager()
    if (hexStringEquals(manager, args.manager)) {
      console.log('manager LGTM')
    } else {
      console.log('manager looks wrong')
    }
  })
