/* External Imports */
import { getLogger } from '@pigi/core-utils'
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
  ovmEntrypoint: string
  ovmCreatedContractAddress: string
  ovmLogs: Log[]
}

/**
 * Convert internal logs into OVM logs. Or in other words, take the logs which
 * are emitted by a normal Ganache or Geth node (this will include logs from the ExecutionManager),
 * parse them, and then convert them into logs which look like they would if you were running this tx
 * using an OVM backend.
 *
 * There are two structures of logs which we expect:
 * 1) Normal contract execution, 2) CreatorContract execution (essentially our form of EOA)
 *
 *  ## Normal contract execution will look like:
 *  [
 *    ActiveContract -- This will always be the entrypoint
 *    Contract1 logs... -- The contract address for normal contract logs will have the **code contract** address. We want the ovm contract address!
 *    ActiveContract -- This event is triggered if the Entrypoint contract CALLs another contract
 *    Contract2 logs...
 *    ActiveContract...
 *    ...etc...
 *  ]
 *
 *  These logs will be parsed to instead look like:
 *  [
 *    Contract1 logs (with correct ovm contract address)
 *    Contract2 logs (^)
 *    ...etc...
 *  ]
 *  And we will return the ovmEntrypoint as the Contract1 address (and a null ovmCreatedContract)
 *
 *  ## CreatorContract execution
 *  [
 *    ActiveContract -- This will always be the entrypoint === CREATOR_CONTRACT_ADDRESS
 *    ActiveContract -- This will always be the newly created ovm contract address
 *    CreatedContract -- This will be the newly created ovm contract address (just like previous ActiveContract event)
 *    Contract1 logs... -- Any logs emitted in the initcode or contracts called in the constructor
 *    ActiveContract -- This event is triggered if the created contract CALLs another contract
 *    ...etc...
 *  ]
 *
 *  These logs will be parsed to instead look like:
 *  [
 *    Contract1 logs (with correct ovm contract address)
 *    Contract2 logs (^)
 *    ...etc...
 *  ]
 *
 *  This time when parsing, the `ovmCreatedContractAddress` will be equal to the newly created contract address, but
 *  otherwise logs will be parsed in the same way.
 *
 *  This function will handle these two cases.
 *
 * @param executionManager an Ethers executionManager object which allows us to parse the event & get
 *                         the execution manager's address.
 * @param logs an array of internal logs which we will parse and then convert.
 * @return LogConversionResult which contains the converted logs & information on entrypoint & created contract address.
 */
export const convertInternalLogsToOvmLogs = (
  executionManager: Contract,
  logs: Log[]
): LogConversionResult => {
  let ovmCreatedContractAddress = null // The address of a newly created contract (NOTE: null is what is returned by Ethers.js)
  let activeContract // A pointer to the current active contract, used for overwriting the internal logs `address` feild.
  let logCounter = 0 // Counter used to iterate over all the to be converted logs

  // Set the ovmEntrypoint based on the first event -- it must be an activeContract event referencing the ovmEntrypoint.
  const ovmEntrypoint = executionManager.interface.parseLog(logs[0]).values[
    '_activeContract'
  ]
  activeContract = ovmEntrypoint
  logCounter++

  // Handle special case #2 -- CreatorContract execution
  // If the entrypoint is the CREATOR_CONTRACT_ADDRESS then we must have created a contract. Get the new contract address
  // and set it as the `ovmCreatedContractAddress` field.
  if (ovmEntrypoint === CREATOR_CONTRACT_ADDRESS) {
    // The ovmContractAddress will be the 3rd event emitted if the entrypoint is the creator contract
    ovmCreatedContractAddress = executionManager.interface.parseLog(logs[2])
      .values['_ovmContractAddress']
    logCounter += 2
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
    ovmEntrypoint,
    ovmCreatedContractAddress,
    ovmLogs,
  }
}
