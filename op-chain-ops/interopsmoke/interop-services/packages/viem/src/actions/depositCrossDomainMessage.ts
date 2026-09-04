import type {
  Account,
  Address,
  Chain,
  Client,
  ContractFunctionReturnType,
  DeriveChain,
  EstimateContractGasParameters,
  GetChainParameter,
  Hash,
  Hex,
  SimulateContractParameters,
  Transport,
} from 'viem'
import { estimateContractGas, simulateContract } from 'viem/actions'

import { crossDomainMessengerAbi } from '@/abis/index.js'
import {
  baseWriteAction,
  type BaseWriteContractActionParameters,
} from '@/core/baseWriteAction.js'
import type { GetContractAddressParameter } from '@/types/utils.js'

/**
 * Deposit a cross-domain message from the root chain (L1).
 * @category Actions
 * @param client - Client for the depositing chain
 * @param parameters - {@link DepositCrossDomainMessageParameters}
 * @returns The transaction hash. {@link DepositCrossDomainMessageReturnType}
 * @example
 * import { depositCrossDomainMessage } from '@eth-optimism/viem'
 * import { op } from '@eth-optimism/viem/chains'
 *
 * const hash = await depositCrossDomainMessage(client, {
 *   target: '0x0000000000000000000000000000000000000000',
 *   message: '0x',
 *   targetChain: op,
 * })
 */
export type DepositCrossDomainMessageParameters<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
  TChainOverride extends Chain | undefined,
  TDerivedChain extends Chain | undefined = DeriveChain<TChain, TChainOverride>,
> = GetChainParameter<TChain, TChainOverride> &
  GetContractAddressParameter<TDerivedChain, 'l1CrossDomainMessenger'> &
  BaseWriteContractActionParameters<
    TChain,
    TAccount,
    TChainOverride,
    TDerivedChain
  > & {
    /** The address of the target contract */
    target: Address
    /** The calldata to invoke the target with */
    message: Hex
    /** The value to send with the transaction */
    value: bigint
    /** The minimum gas limit for the transaction */
    minGasLimit?: bigint
  }

export type DepositCrossDomainMessageReturnType = Hash

export type DepositCrossDomainMessageContractReturnType =
  ContractFunctionReturnType<
    typeof crossDomainMessengerAbi,
    'payable',
    'sendMessage'
  >

/**
 * Deposit a cross-domain message from the root chain (L1).
 * @category Actions
 * @param client - Client for the depositing chain
 * @param parameters - {@link DepositCrossDomainMessageParameters}
 * @returns The transaction hash. {@link DepositCrossDomainMessageReturnType}
 */
export async function depositCrossDomainMessage<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
  TChainOverride extends Chain | undefined,
>(
  client: Client<Transport, TChain, TAccount>,
  parameters: DepositCrossDomainMessageParameters<
    TChain,
    TAccount,
    TChainOverride
  >,
): Promise<DepositCrossDomainMessageReturnType> {
  const {
    chain = client.chain,
    target,
    message,
    targetChain,
    minGasLimit = 0n,
  } = parameters

  const cdmAddress = (() => {
    if (parameters.l1CrossDomainMessengerAddress)
      return parameters.l1CrossDomainMessengerAddress
    if (chain)
      return targetChain!.contracts.l1CrossDomainMessenger[chain.id].address
    return Object.values(targetChain!.contracts.l1CrossDomainMessenger)[0]
      .address
  })()

  return baseWriteAction(
    client,
    {
      abi: crossDomainMessengerAbi,
      contractAddress: cdmAddress,
      contractFunctionName: 'sendMessage',
      contractArgs: [target, message, minGasLimit],
    },
    parameters as BaseWriteContractActionParameters,
  )
}

/**
 * Simulate the {@link depositCrossDomainMessage} action.
 * @category Actions
 * @param client - Client for the depositing chain
 * @param parameters - {@link DepositCrossDomainMessageParameters}
 * @returns The contract functions return value. {@link DepositCrossDomainMessageContractReturnType}
 */
export async function simulateDepositCrossDomainMessage<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
  TChainOverride extends Chain | undefined,
>(
  client: Client<Transport, TChain, TAccount>,
  parameters: DepositCrossDomainMessageParameters<
    TChain,
    TAccount,
    TChainOverride
  >,
): Promise<DepositCrossDomainMessageContractReturnType> {
  const {
    chain = client.chain,
    target,
    message,
    targetChain,
    minGasLimit = 0n,
  } = parameters

  const cdmAddress = (() => {
    if (parameters.l1CrossDomainMessengerAddress)
      return parameters.l1CrossDomainMessengerAddress
    if (chain)
      return targetChain!.contracts.l1CrossDomainMessenger[chain.id].address
    return Object.values(targetChain!.contracts.l1CrossDomainMessenger)[0]
      .address
  })()

  const { result } = await simulateContract(client, {
    abi: crossDomainMessengerAbi,
    address: cdmAddress,
    functionName: 'sendMessage',
    args: [target, message, minGasLimit],
    ...parameters,
  } as SimulateContractParameters)

  return result as DepositCrossDomainMessageContractReturnType
}

/**
 * Estimate the gas cost of the {@link depositCrossDomainMessage} action.
 * @category Actions
 * @param client - Client for the depositing chain
 * @param parameters - {@link DepositCrossDomainMessageParameters}
 * @returns The gas cost
 */
export async function estimateDepositCrossDomainMessageGas<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
  TChainOverride extends Chain | undefined,
>(
  client: Client<Transport, TChain, TAccount>,
  parameters: DepositCrossDomainMessageParameters<
    TChain,
    TAccount,
    TChainOverride
  >,
): Promise<bigint> {
  const {
    chain = client.chain,
    target,
    message,
    targetChain,
    minGasLimit = 0n,
  } = parameters

  const cdmAddress = (() => {
    if (parameters.l1CrossDomainMessengerAddress)
      return parameters.l1CrossDomainMessengerAddress
    if (chain)
      return targetChain!.contracts.l1CrossDomainMessenger[chain.id].address
    return Object.values(targetChain!.contracts.l1CrossDomainMessenger)[0]
      .address
  })()

  return estimateContractGas(client, {
    abi: crossDomainMessengerAbi,
    address: cdmAddress,
    functionName: 'sendMessage',
    args: [target, message, minGasLimit],
    ...parameters,
  } as EstimateContractGasParameters)
}
