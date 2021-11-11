'use strict'

import { ethers } from 'ethers'
import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'
import { hexStringEquals } from '@eth-optimism/core-utils'
import { getContractFactory } from '../src/contract-defs'

import {
  getInput,
  color as c,
  getArtifact,
  getEtherscanUrl,
  printComparison,
} from '../src/validation-utils'

task('validate:address-dictator')
  .addParam(
    'dictator',
    'Address of the AddressDictator to validate.',
    undefined,
    types.string
  )
  .addParam(
    'manager',
    'Address of the Address Manager contract which would be updated by the Dictator.',
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
    console.log()
    console.log(c.cyan("First make sure you're on the right chain:"))
    console.log(
      `Reading from the ${c.red(network.name)} network (Chain ID: ${c.red(
        '' + network.chainId
      )})`
    )
    await getInput(c.yellow('OK? Hit enter to continue.'))

    // eslint-disable-next-line @typescript-eslint/no-var-requires
    const dictatorArtifact = require('../artifacts/contracts/L1/deployment/AddressDictator.sol/AddressDictator.json')
    const dictatorCode = await provider.getCode(args.dictator)
    console.log(
      c.cyan(`
Now validating the Address Dictator deployment at\n${getEtherscanUrl(
        network,
        args.dictator
      )}`)
    )
    printComparison(
      'Comparing deployed AddressDictator bytecode against local build artifacts',
      'Deployed AddressDictator code',
      { name: 'Compiled bytecode', value: dictatorArtifact.deployedBytecode },
      { name: 'Deployed bytecode', value: dictatorCode }
    )

    // Connect to the deployed AddressDictator.
    const dictatorContract = getContractFactory('AddressDictator')
      .attach(args.dictator)
      .connect(provider)

    const finalOwner = await dictatorContract.finalOwner()
    printComparison(
      'Comparing the finalOwner address in the AddressDictator to the multisig address',
      'finalOwner',
      { name: 'multisig address', value: args.multisig },
      { name: 'finalOwner      ', value: finalOwner }
    )

    const manager = await dictatorContract.manager()
    printComparison(
      'Validating the AddressManager address in the AddressDictator',
      'addressManager',
      { name: 'manager', value: args.manager },
      { name: 'Address Manager', value: manager }
    )
    await getInput(c.yellow('OK? Hit enter to continue.'))

    // Get names and addresses from the Dictator.
    const namedAddresses = await dictatorContract.getNamedAddresses()

    // In order to reduce noise for the user, we query the AddressManager identify addresses that
    // will not be changed, and skip over them in this block.
    const managerContract = getContractFactory('Lib_AddressManager')
      .attach(args.manager)
      .connect(provider)

    // Now we loop over those and compare the addresses/deployedBytecode to deployment artifacts.
    for (const pair of namedAddresses) {
      const currentAddress = await managerContract.getAddress(pair.name)
      const artifact = getArtifact(pair.name)
      const addressChanged = !hexStringEquals(currentAddress, pair.addr)
      if (addressChanged) {
        console.log(
          c.cyan(`
Now validating the ${pair.name} deployment.
Current address: ${getEtherscanUrl(network, currentAddress)}
Upgraded address ${getEtherscanUrl(network, pair.addr)}`)
        )

        const code = await provider.getCode(pair.addr)
        printComparison(
          `Verifying ${pair.name} source code against local deployment artifacts`,
          `Deployed ${pair.name} code`,
          {
            name: 'artifact.deployedBytecode',
            value: artifact.deployedBytecode,
          },
          { name: 'Deployed bytecode        ', value: code }
        )

        // Identify contracts which inherit from Lib_AddressResolver, and check that they
        // have the right manager address.
        if (Object.keys(artifact)) {
          if (artifact.abi.some((el) => el.name === 'libAddressManager')) {
            const libAddressManager = await getContractFactory(
              'Lib_AddressResolver'
            )
              .attach(pair.addr)
              .connect(provider)
              .libAddressManager()

            printComparison(
              `Verifying ${pair.name} has the correct AddressManager address`,
              `The AddressManager address in ${pair.name}`,
              { name: 'Deployed value', value: libAddressManager },
              { name: 'Expected value', value: manager }
            )
          }
        }
      }
      await getInput(c.yellow('OK? Hit enter to continue.'))
    }
    console.log(c.green('AddressManager Validation complete!'))
  })
