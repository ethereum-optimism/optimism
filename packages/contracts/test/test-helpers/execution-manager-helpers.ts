/* External Imports */
import { Contract, Signer } from 'ethers'
import { fromPairs, cloneDeep } from 'lodash'
import { getCurrentTime, ZERO_ADDRESS, add0x } from '@eth-optimism/core-utils'

/* Internal Imports */
import { GAS_LIMIT } from './constants'
import { encodeMethodId, encodeFunctionData } from './ethereum-helpers'

/* Contract Imports */
import * as ExecutionManagerJson from '../../artifacts/ExecutionManager.json'

export const OVM_METHOD_IDS = fromPairs(
  [
    'makeCall',
    'makeDelegateCall',
    'makeStaticCall',
    'makeStaticCallThenCall',
    'callThroughExecutionManager',
    'staticFriendlySLOAD',
    'notStaticFriendlySSTORE',
    'notStaticFriendlyCREATE',
    'notStaticFriendlyCREATE2',
    'getADDRESS',
    'getCALLER',
    'getGASLIMIT',
    'getQueueOrigin',
    'getTIMESTAMP',
    'getCHAINID',
    'ovmADDRESS',
    'ovmCALLER',
    'ovmCREATE',
    'ovmCREATE2'
  ].map((methodId) => [methodId, encodeMethodId(methodId)])
)

/**
 * Override the ABI description of a particular function, changing it's `constant` & `outputs` values.
 * @param {Array} an abi object.
 * @param {string} the name of the function we would like to change.
 * @param {Object} an object containing the new `constant` & `outputs` values.
 */
export function overrideAbiFunctionData(
  abiDefinition: any,
  functionName: string,
  functionData: { constant: boolean; outputs: any[]; stateMutability: string }
): void {
  for (const functionDefinition of abiDefinition) {
    if (functionDefinition.name === functionName) {
      functionDefinition.constant = functionData.constant
      functionDefinition.outputs = functionData.outputs.map((output) => {
        return { internalType: output, name: '', type: output }
      })
      functionDefinition.stateMutability = functionData.stateMutability
    }
  }
}

/**
 * Use executeTransaction with `eth_call`.
 * @param {ethers.Contract} an ExecutionManager contract instance used for it's address & provider.
 * @param {Array} an array of parameters which should be fed into `executeTransaction(...)`.
 * @param {OutputTypes} an array ABI types which should be used to decode the output of the call.
 */
export function callExecutionManagerExecuteTransaction(
  executionManager: Contract,
  parameters: any[],
  outputTypes: any[]
): Promise<any[]> {
  const modifiedAbi = cloneDeep(ExecutionManagerJson.abi)
  overrideAbiFunctionData(modifiedAbi, 'executeTransaction', {
    constant: true,
    outputs: outputTypes,
    stateMutability: 'view',
  })
  const callableExecutionManager = new Contract(
    executionManager.address,
    modifiedAbi,
    executionManager.provider
  )
  return callableExecutionManager.executeTransaction.apply(null, parameters)
}

export const executePersistedTestTransaction = async (
  executionManager: Contract,
  wallet: Signer,
  callContractAddress: string,
  methodName: string,
  args: any[]
): Promise<string> => {
  const callBytes = encodeFunctionData(
    methodName,
    args
  )

  const data = executionManager.interface.encodeFunctionData(
    'executeTransaction',
    [
      getCurrentTime(),
      0,
      callContractAddress,
      callBytes,
      ZERO_ADDRESS,
      ZERO_ADDRESS,
      true,
    ]
  )

  const receipt = await wallet.sendTransaction({
    to: executionManager.address,
    data: add0x(data),
    gasLimit: GAS_LIMIT,
  })

  return receipt.hash
}

export const executeTestTransaction = async (
  executionManager: Contract,
  contractAddress: string,
  methodName: string,
  args: any[],
  queueOrigin = ZERO_ADDRESS
): Promise<string> => {
  const callBytes = encodeFunctionData(
    methodName,
    args
  )

  const data = executionManager.interface.encodeFunctionData(
    'executeTransaction',
    [
      getCurrentTime(),
      queueOrigin,
      contractAddress,
      callBytes,
      ZERO_ADDRESS,
      ZERO_ADDRESS,
      true,
    ]
  )

  return executionManager.provider.call({
    to: executionManager.address,
    data,
    gasLimit: GAS_LIMIT,
  })
}