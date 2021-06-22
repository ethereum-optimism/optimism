/* Imports: External */
import { HardhatNetworkProvider } from 'hardhat/internal/hardhat-network/provider/provider'
import { VmError } from '@nomiclabs/ethereumjs-vm/dist/exceptions'
import BN from 'bn.js'

/* eslint-disable @typescript-eslint/no-var-requires */
// Handle hardhat ^2.4.0
let decodeRevertReason: (value: Buffer) => string
try {
  decodeRevertReason = require('hardhat/internal/hardhat-network/stack-traces/revert-reasons')
    .decodeRevertReason
} catch (err) {
  const {
    ReturnData,
  } = require('hardhat/internal/hardhat-network/provider/return-data')
  decodeRevertReason = (value: Buffer) => {
    return new ReturnData(value).decodeError()
  }
}
// Handle hardhat ^2.2.0
let TransactionExecutionError: any
try {
  TransactionExecutionError = require('hardhat/internal/hardhat-network/provider/errors')
    .TransactionExecutionError
} catch (err) {
  TransactionExecutionError = require('hardhat/internal/core/providers/errors')
    .TransactionExecutionError
}
/* eslint-enable @typescript-eslint/no-var-requires */

/* Imports: Internal */
import { MockContract, SmockedVM } from './types'
import { fromFancyAddress, toFancyAddress } from '../common'

/**
 * Checks to see if smock has been initialized already. Basically just checking to see if we've
 * attached smock state to the VM already.
 *
 * @param provider Base hardhat network provider to check.
 * @return Whether or not the provider has already been modified to support smock.
 */
const isSmockInitialized = (provider: HardhatNetworkProvider): boolean => {
  return (provider as any)._node._vm._smockState !== undefined
}

/**
 * Modifies a hardhat provider to be compatible with smock.
 *
 * @param provider Base hardhat network provider to modify.
 */
const initializeSmock = (provider: HardhatNetworkProvider): void => {
  if (isSmockInitialized(provider)) {
    return
  }

  // Will need to reference these things.
  const node = (provider as any)._node
  const vm: SmockedVM = node._vm

  // Attach some extra state to the VM.
  vm._smockState = {
    mocks: {},
    calls: {},
    messages: [],
  }

  // Wipe out our list of calls before each transaction.
  vm.on('beforeTx', () => {
    vm._smockState.calls = {}
  })

  // Watch for new EVM messages (call frames).
  vm.on('beforeMessage', (message: any) => {
    // Happens with contract creations. If the current message is a contract creation then it can't
    // be a call to a smocked contract.
    if (!message.to) {
      return
    }

    let target: string
    if (message.delegatecall) {
      target = fromFancyAddress(message._codeAddress)
    } else {
      target = fromFancyAddress(message.to)
    }

    // Check if the target address is a smocked contract.
    if (!(target in vm._smockState.mocks)) {
      return
    }

    // Initialize the array of calls to this smock if not done already.
    if (!(target in vm._smockState.calls)) {
      vm._smockState.calls[target] = []
    }

    // Record this message for later.
    vm._smockState.calls[target].push(message.data)
    vm._smockState.messages.push(message)
  })

  // Now *this* is a hack.
  // Ethereumjs-vm passes `result` by *reference* into the `afterMessage` event. Mutating the
  // `result` object here will actually mutate the result in the VM. Magic.
  vm.on('afterMessage', async (result: any) => {
    // We currently defer to contract creations, meaning we'll "unsmock" an address if a user
    // later creates a contract at that address. Not sure how to handle this case. Very open to
    // ideas.
    if (result.createdAddress) {
      const created = fromFancyAddress(result.createdAddress)
      if (created in vm._smockState.mocks) {
        delete vm._smockState.mocks[created]
      }
    }

    // Check if we have messages that need to be handled.
    if (vm._smockState.messages.length === 0) {
      return
    }

    // Handle the last message that was pushed to the array of messages. This works because smock
    // contracts never create new sub-calls (meaning this `afterMessage` event corresponds directly
    // to a `beforeMessage` event emitted during a call to a smock contract).
    const message = vm._smockState.messages.pop()

    let target: string
    if (message.delegatecall) {
      target = fromFancyAddress(message._codeAddress)
    } else {
      target = fromFancyAddress(message.to)
    }

    // Not sure if this can ever actually happen? Just being safe.
    if (!(target in vm._smockState.mocks)) {
      return
    }

    // Compute the mock return data.
    const mock: MockContract = vm._smockState.mocks[target]
    const {
      resolve,
      functionName,
      rawReturnValue,
      returnValue,
      gasUsed,
    } = await mock._smockit(message.data)

    // Set the mock return data, potentially set the `exceptionError` field if the user requested
    // a revert.
    result.gasUsed = new BN(gasUsed)
    result.execResult.returnValue = returnValue
    result.execResult.gasUsed = new BN(gasUsed)
    result.execResult.exceptionError =
      resolve === 'revert' ? new VmError('smocked revert' as any) : undefined
  })

  // Here we're fixing with hardhat's internal error management. Smock is a bit weird and messes
  // with stack traces so we need to help hardhat out a bit when it comes to smock-specific
  // errors.
  const originalManagerErrorsFn = node._manageErrors.bind(node)
  node._manageErrors = async (
    vmResult: any,
    vmTrace: any,
    vmTracerError?: any
  ): Promise<any> => {
    if (
      vmResult.exceptionError &&
      vmResult.exceptionError.error === 'smocked revert'
    ) {
      return new TransactionExecutionError(
        `VM Exception while processing transaction: revert ${decodeRevertReason(
          vmResult.returnValue
        )}`
      )
    }

    return originalManagerErrorsFn(vmResult, vmTrace, vmTracerError)
  }
}

/**
 * Attaches a smocked contract to a hardhat network provider. Will also modify the provider to be
 * compatible with smock if not done already.
 *
 * @param mock Smocked contract to attach to a provider.
 * @param provider Hardhat network provider to attach the contract to.
 */
export const bindSmock = async (
  mock: MockContract,
  provider: HardhatNetworkProvider
): Promise<void> => {
  if (!isSmockInitialized(provider)) {
    initializeSmock(provider)
  }

  const vm: SmockedVM = (provider as any)._node._vm
  const pStateManager = vm.pStateManager || vm.stateManager

  // Add mock to our list of mocks currently attached to the VM.
  vm._smockState.mocks[mock.address.toLowerCase()] = mock

  // Set the contract code for our mock to 0x00 == STOP. Need some non-empty contract code because
  // Solidity will sometimes throw if it's calling something without code (I forget the exact
  // scenario that causes this throw).
  await pStateManager.putContractCode(
    toFancyAddress(mock.address),
    Buffer.from('00', 'hex')
  )
}

/**
 * Detaches a smocked contract from a hardhat network provider.
 *
 * @param mock Smocked contract to detach to a provider, or an address.
 * @param provider Hardhat network provider to detatch the contract from.
 */
export const unbindSmock = async (
  mock: MockContract | string,
  provider: HardhatNetworkProvider
): Promise<void> => {
  if (!isSmockInitialized(provider)) {
    initializeSmock(provider)
  }

  const vm: SmockedVM = (provider as any)._node._vm
  const pStateManager = vm.pStateManager || vm.stateManager

  // Add mock to our list of mocks currently attached to the VM.
  const address = typeof mock === 'string' ? mock : mock.address.toLowerCase()
  delete vm._smockState.mocks[address]

  // Set the contract code for our mock to 0x00 == STOP. Need some non-empty contract code because
  // Solidity will sometimes throw if it's calling something without code (I forget the exact
  // scenario that causes this throw).
  await pStateManager.putContractCode(
    toFancyAddress(address),
    Buffer.from('', 'hex')
  )
}
