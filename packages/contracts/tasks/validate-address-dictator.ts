'use strict'

import { ethers } from 'ethers'
import { task } from 'hardhat/config'
import * as types from 'hardhat/internal/core/params/argumentTypes'
import { hexStringEquals } from '@eth-optimism/core-utils'
import { getContractFactory, getContractDefinition } from '../src/contract-defs'
import { names } from '../src/address-names'

import {
  getInput,
  color as c,
  getArtifactFromManagedName,
  getEtherscanUrl,
  printSectionHead,
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
    printSectionHead("First make sure you're on the right chain:")
    console.log(
      `Reading from the ${c.red(network.name)} network (Chain ID: ${c.red(
        '' + network.chainId
      )})`
    )
    await getInput(c.yellow('OK? Hit enter to continue.'))

    const dictatorArtifact = getContractDefinition('AddressDictator')
    const dictatorCode = await provider.getCode(args.dictator)
    printSectionHead(`
Validate the Address Dictator deployment at\n${getEtherscanUrl(
      network,
      args.dictator
    )}`)

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

    const deployedManager = await dictatorContract.manager()
    printComparison(
      'Validating the AddressManager address in the AddressDictator',
      'addressManager',
      { name: 'manager        ', value: args.manager },
      { name: 'Address Manager', value: deployedManager }
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
      if (pair.name === 'L2CrossDomainMessenger') {
        console.log('L2CrossDomainMessenger is set to:', pair.addr)
        await getInput(c.yellow('OK? Hit enter to continue.'))
        // This is an L2 predeploy, so we skip bytecode and config validation.
        continue
      }
      const currentAddress = await managerContract.getAddress(pair.name)
      const artifact = getArtifactFromManagedName(pair.name)
      const addressChanged = !hexStringEquals(currentAddress, pair.addr)
      if (addressChanged) {
        printSectionHead(
          `Validate the ${pair.name} deployment.
Current address: ${getEtherscanUrl(network, currentAddress)}
Upgraded address ${getEtherscanUrl(network, pair.addr)}`
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
              { name: 'Expected value', value: deployedManager }
            )
            await getInput(c.yellow('OK? Hit enter to continue.'))
          }
        }
      }
      await validateDeployedConfig(provider, network, args.manager, pair)
    }
    console.log(c.green('\nAddressManager Validation complete!'))
  })

/**
 * Validates that the deployed contracts have the expected storage variables.
 *
 * @param {*} provider
 * @param {{ name: string; addr: string }} pair The contract name and address
 */
const validateDeployedConfig = async (
  provider,
  network,
  manager,
  pair: { name: string; addr: string }
) => {
  printSectionHead(`
Ensure that the ${pair.name} at\n${getEtherscanUrl(
    network,
    pair.addr
  )} is configured correctly`)
  if (pair.name === names.managed.contracts.StateCommitmentChain) {
    const scc = getContractFactory(pair.name)
      .attach(pair.addr)
      .connect(provider)
    //  --scc-fraud-proof-window 604800 \
    const fraudProofWindow = await scc.FRAUD_PROOF_WINDOW()
    printComparison(
      'Checking the fraudProofWindow of the StateCommitmentChain',
      'StateCommitmentChain.fraudProofWindow',
      {
        name: 'Configured fraudProofWindow',
        value: ethers.BigNumber.from(604_800).toHexString(),
      },
      {
        name: 'Deployed fraudProofWindow  ',
        value: ethers.BigNumber.from(fraudProofWindow).toHexString(),
      }
    )
    await getInput(c.yellow('OK? Hit enter to continue.'))

    //  --scc-sequencer-publish-window 12592000 \
    const sequencerPublishWindow = await scc.SEQUENCER_PUBLISH_WINDOW()
    printComparison(
      'Checking the sequencerPublishWindow of the StateCommitmentChain',
      'StateCommitmentChain.sequencerPublishWindow',
      {
        name: 'Configured sequencerPublishWindow  ',
        value: ethers.BigNumber.from(12592000).toHexString(),
      },
      {
        name: 'Deployed sequencerPublishWindow',
        value: ethers.BigNumber.from(sequencerPublishWindow).toHexString(),
      }
    )
    await getInput(c.yellow('OK? Hit enter to continue.'))
  } else if (pair.name === names.managed.contracts.CanonicalTransactionChain) {
    const ctc = getContractFactory(pair.name)
      .attach(pair.addr)
      .connect(provider)

    //  --ctc-max-transaction-gas-limit 15000000 \
    const maxTransactionGasLimit = await ctc.maxTransactionGasLimit()
    printComparison(
      'Checking the maxTransactionGasLimit of the CanonicalTransactionChain',
      'CanonicalTransactionChain.maxTransactionGasLimit',
      {
        name: 'Configured maxTransactionGasLimit',
        value: ethers.BigNumber.from(15_000_000).toHexString(),
      },
      {
        name: 'Deployed maxTransactionGasLimit  ',
        value: ethers.BigNumber.from(maxTransactionGasLimit).toHexString(),
      }
    )
    await getInput(c.yellow('OK? Hit enter to continue.'))
    //  --ctc-l2-gas-discount-divisor 32 \
    const l2GasDiscountDivisor = await ctc.l2GasDiscountDivisor()
    printComparison(
      'Checking the l2GasDiscountDivisor of the CanonicalTransactionChain',
      'CanonicalTransactionChain.l2GasDiscountDivisor',
      {
        name: 'Configured l2GasDiscountDivisor',
        value: ethers.BigNumber.from(32).toHexString(),
      },
      {
        name: 'Deployed l2GasDiscountDivisor  ',
        value: ethers.BigNumber.from(l2GasDiscountDivisor).toHexString(),
      }
    )
    await getInput(c.yellow('OK? Hit enter to continue.'))
    //  --ctc-enqueue-gas-cost 60000 \
    const enqueueGasCost = await ctc.enqueueGasCost()
    printComparison(
      'Checking the enqueueGasCost of the CanonicalTransactionChain',
      'CanonicalTransactionChain.enqueueGasCost',
      {
        name: 'Configured enqueueGasCost',
        value: ethers.BigNumber.from(60000).toHexString(),
      },
      {
        name: 'Deployed enqueueGasCost  ',
        value: ethers.BigNumber.from(enqueueGasCost).toHexString(),
      }
    )
    await getInput(c.yellow('OK? Hit enter to continue.'))
  } else if (pair.name === names.managed.contracts.OVM_L1CrossDomainMessenger) {
    const messengerManager = await getContractFactory('L1CrossDomainMessenger')
      .attach(pair.addr)
      .connect(provider)
      .libAddressManager()
    printComparison(
      'Ensure that the L1CrossDomainMessenger (implementation) is initialized with a non-zero Address Manager variable',
      "L1CrossDomainMessenger's Lib_AddressManager",
      {
        name: 'Configured Lib_AddressManager',
        value: messengerManager,
      },
      {
        name: 'Deployed Lib_AddressManager  ',
        value: manager,
      }
    )
  } else {
    console.log(c.green(`${pair.name} has no config to check`))
    await getInput(c.yellow('OK? Hit enter to continue.'))
  }
}
