/* External Imports */
import { getLogger, ZERO_ADDRESS } from '@pigi/core-utils'
import { Contract } from 'ethers'
import { Log } from 'ethers/providers'

/* Internal Imports */
import { CREATOR_CONTRACT_ADDRESS } from '.'

/* Contract Imports */

const log = getLogger('utils')

/**
 * Contract Definitions!
 * Useful if you need to deploy an ExecutionManager from a different package
 */
// Contract Imports
import * as L2ExecutionManager from '../../build/contracts/L2ExecutionManager.json'
import * as ContractAddressGenerator from '../../build/contracts/ContractAddressGenerator.json'
import * as RLPEncode from '../../build/contracts/RLPEncode.json'
// Contract Exports
export const L2ExecutionManagerContractDefinition = {
  abi: L2ExecutionManager.abi,
  bytecode: L2ExecutionManager.bytecode,
}
export const ContractAddressGeneratorContractDefinition = {
  abi: ContractAddressGenerator.abi,
  bytecode: ContractAddressGenerator.bytecode,
}
export const RLPEncodeContractDefinition = {
  abi: RLPEncode.abi,
  bytecode: RLPEncode.bytecode,
}

/**
 * OVM Event parsing!
 * Helper function for converting normal EVM logs into OVM logs.
 * This is used to detect if a contract was deployed, or to read logs with the correct
 * OVM addresses.
 */
const ExecutionManagerEvents = {
  activeContract: 'ActiveContract',
  createdContract: 'CreatedContract',
}

interface LogConversionResult {
  ovmTo: string
  ovmFrom: string
  ovmCreatedContractAddress: string
  ovmLogs: Log[]
}

/**
 * Convert internal logs into OVM logs. Or in other words, take the logs which
 * are emitted by a normal Ganache or Geth node (this will include logs from the ExecutionManager),
 * parse them, and then convert them into logs which look like they would if you were running this tx
 * using an OVM backend.
 *
 * TODO: Add documentation on how the events are parsed
 *
 * @param executionManager an Ethers executionManager object which allows us to parse the event & get
 *                         the execution manager's address.
 * @param logs an array of internal logs which we will parse and then convert.
 * @return LogConversionResult which contains the converted logs & information on entrypoint & created contract address.
 */
export const convertInternalLogsToOvmLogs = (
  executionManager: Contract,
  logs: Log[],
  debugMode: boolean = false
): LogConversionResult => {
  if (logs.length === 0) {
    throw new Error('Expected logs from ExecutionManager!')
  }

  let ovmCreatedContractAddress = null // The address of a newly created contract (NOTE: null is what is returned by Ethers.js)
  let activeContract // A pointer to the current active contract, used for overwriting the internal logs `address` feild.
  let logCounter = 0 // Counter used to iterate over all the to be converted logs
  let ovmFrom = ZERO_ADDRESS
  let ovmTo

  if (executionManager.interface.parseLog(logs[0]).name === 'CallingWithEOA') {
    // Initiate EOA log parsing
    ovmFrom = executionManager.interface.parseLog(logs[1]).values[
      '_activeContract'
    ]
    // Check if we are creating a new contract
    if (
      executionManager.interface.parseLog(logs[2]).name === 'EOACreatedContract'
    ) {
      ovmCreatedContractAddress = executionManager.interface.parseLog(logs[2])
        .values['_ovmContractAddress']
      ovmTo = ovmCreatedContractAddress
    } else {
      ovmTo = executionManager.interface.parseLog(logs[2]).values[
        '_activeContract'
      ]
    }
    logCounter += 3
  } else {
    ovmTo = executionManager.interface.parseLog(logs[0]).values[
      '_activeContract'
    ]
  }

  if (debugMode) {
    // Print all the ExecutionManager logs -- very useful for debugging
    log.debug('Converting logs! Pre-conversion log list:')
    logs.map((_log) => log.debug(executionManager.interface.parseLog(_log)))
  }

  // Now iterate over the remaining logs, converting them and adding them to our ovmLogs list
  const ovmLogs: Log[] = []
  for (; logCounter < logs.length; logCounter++) {
    const internalLog = logs[logCounter]
    const parsedLog = executionManager.interface.parseLog(internalLog)

    // Check if this log is emitted by the Execution Manager if so we may need to switch the active contract
    if (
      internalLog.address.toLowerCase() ===
      executionManager.address.toLowerCase()
    ) {
      // Check if we've switched context -- this is used to replace the contractAddress
      if (parsedLog.name === ExecutionManagerEvents.activeContract) {
        activeContract = parsedLog.values['_activeContract']
      } else {
        // Otherwise simply skip the log
        continue
      }
    }

    // Push an ovmLog which is the same as the internal log but with an ovmContract address
    ovmLogs.push({ ...internalLog, ...{ address: activeContract } })
  }

  return {
    ovmTo,
    ovmFrom,
    ovmCreatedContractAddress,
    ovmLogs,
  }
}
