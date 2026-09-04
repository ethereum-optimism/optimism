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
import { contracts } from '@/contracts.js'
import type { BaseWriteContractActionParameters } from '@/core/baseWriteAction.js'
import { baseWriteAction } from '@/core/baseWriteAction.js'

/**
 * Withdraw a cross-domain message from the child chain (L2).
 * @category Actions
 * @param client - Client for the withdrawing chain
 * @param parameters - {@link WithdrawCrossDomainMessageParameters}
 * @returns The transaction hash. {@link WithdrawCrossDomainMessageReturnType}
 * @example
 * import { withdrawCrossDomainMessage } from '@eth-optimism/viem'
 * import { op } from '@eth-optimism/viem/chains'
 *
 * const hash = await withdrawCrossDomainMessage(client, {
 *   target: '0x0000000000000000000000000000000000000000',
 *   message: '0x',
 *   targetChain: op,
 * })
 */
export type WithdrawCrossDomainMessageParameters<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
  TChainOverride extends Chain | undefined,
  TDerivedChain extends Chain | undefined = DeriveChain<TChain, TChainOverride>,
> = GetChainParameter<TChain, TChainOverride> &
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
    /** The address of the CrossDomainMessenger to use. Defaults to the L2CrossDomainMessenger Predeploy */
    crossDomainMessengerAddress?: Address
  }

export type WithdrawCrossDomainMessageReturnType = Hash

export type WithdrawCrossDomainMessageContractReturnType =
  ContractFunctionReturnType<
    typeof crossDomainMessengerAbi,
    'payable',
    'sendMessage'
  >

/**
 * Withdraw a cross-domain message from the child chain (L2).
 * @category Actions
 * @param client - Client for the withdrawing chain
 * @param parameters - {@link WithdrawCrossDomainMessageParameters}
 * @returns The transaction hash. {@link WithdrawCrossDomainMessageReturnType}
 */
export async function withdrawCrossDomainMessage<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
  TChainOverride extends Chain | undefined,
>(
  client: Client<Transport, TChain, TAccount>,
  parameters: WithdrawCrossDomainMessageParameters<
    TChain,
    TAccount,
    TChainOverride
  >,
): Promise<WithdrawCrossDomainMessageReturnType> {
  const { target, message, minGasLimit = 0n } = parameters

  const cdmAddress =
    parameters.crossDomainMessengerAddress ??
    contracts.l2CrossDomainMessenger.address

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
 * Simulate the {@link withdrawCrossDomainMessage} action.
 * @category Actions
 * @param client - Client for the withdrawing chain
 * @param parameters - {@link WithdrawCrossDomainMessageParameters}
 * @returns The contract functions return value. {@link WithdrawCrossDomainMessageContractReturnType}
 */
export async function simulateWithdrawCrossDomainMessage<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
  TChainOverride extends Chain | undefined,
>(
  client: Client<Transport, TChain, TAccount>,
  parameters: WithdrawCrossDomainMessageParameters<
    TChain,
    TAccount,
    TChainOverride
  >,
): Promise<WithdrawCrossDomainMessageContractReturnType> {
  const { target, message, minGasLimit = 0n } = parameters

  const cdmAddress =
    parameters.crossDomainMessengerAddress ??
    contracts.l2CrossDomainMessenger.address

  const { result } = await simulateContract(client, {
    abi: crossDomainMessengerAbi,
    address: cdmAddress,
    functionName: 'sendMessage',
    args: [target, message, minGasLimit],
    ...parameters,
  } as SimulateContractParameters)

  return result as WithdrawCrossDomainMessageContractReturnType
}

/**
 * Estimate the gas cost of the {@link withdrawCrossDomainMessage} action.
 * @category Actions
 * @param client - Client for the withdrawing chain
 * @param parameters - {@link WithdrawCrossDomainMessageParameters}
 * @returns The gas cost
 */
export async function estimateWithdrawCrossDomainMessageGas<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
  TChainOverride extends Chain | undefined,
>(
  client: Client<Transport, TChain, TAccount>,
  parameters: WithdrawCrossDomainMessageParameters<
    TChain,
    TAccount,
    TChainOverride
  >,
): Promise<bigint> {
  const { target, message, minGasLimit = 0n } = parameters

  const cdmAddress =
    parameters.crossDomainMessengerAddress ??
    contracts.l2CrossDomainMessenger.address

  return estimateContractGas(client, {
    abi: crossDomainMessengerAbi,
    address: cdmAddress,
    functionName: 'sendMessage',
    args: [target, message, minGasLimit],
    ...parameters,
  } as EstimateContractGasParameters)
}
