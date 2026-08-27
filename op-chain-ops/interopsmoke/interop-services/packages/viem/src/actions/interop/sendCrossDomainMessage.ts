/** @module sendMessage */
import type {
  Account,
  Address,
  Chain,
  Client,
  ContractFunctionReturnType,
  DeriveChain,
  EstimateContractGasErrorType,
  EstimateContractGasParameters,
  Hash,
  Hex,
  SimulateContractParameters,
  Transport,
  WriteContractErrorType,
} from 'viem'
import { estimateContractGas, simulateContract } from 'viem/actions'

import { l2ToL2CrossDomainMessengerAbi } from '@/abis/index.js'
import { contracts } from '@/contracts.js'
import type { BaseWriteContractActionParameters } from '@/core/baseWriteAction.js'
import { baseWriteAction } from '@/core/baseWriteAction.js'
import type { ErrorType } from '@/types/utils.js'

/**
 * @category Types
 */
export type SendCrossDomainMessageParameters<
  TChain extends Chain | undefined = Chain | undefined,
  TAccount extends Account | undefined = Account | undefined,
  TChainOverride extends Chain | undefined = Chain | undefined,
  TDerivedChain extends Chain | undefined = DeriveChain<TChain, TChainOverride>,
> = BaseWriteContractActionParameters<
  TChain,
  TAccount,
  TChainOverride,
  TDerivedChain
> & {
  /** Chain ID of the destination chain. */
  destinationChainId: number
  /** Target contract or wallet address. */
  target: Address
  /** Message payload to call target with. */
  message: Hex
}

/**
 * @category Types
 */
export type SendCrossDomainMessageReturnType = Hash

/**
 * @category Types
 */
export type SendCrossDomainMessageContractReturnType =
  ContractFunctionReturnType<
    typeof l2ToL2CrossDomainMessengerAbi,
    'nonpayable',
    'sendMessage'
  >

/**
 * @category Types
 */
export type SendCrossDomainMessageErrorType =
  | EstimateContractGasErrorType
  | WriteContractErrorType
  | ErrorType

/**
 * Initiates the intent of sending a L2 to L2 message. Used in the interop flow.
 * @category Actions
 * @param client - L2 Client
 * @param parameters - {@link SendCrossDomainMessageParameters}
 * @returns transaction hash - {@link SendCrossDomainMessageReturnType}
 */
export async function sendCrossDomainMessage<
  chain extends Chain | undefined,
  account extends Account | undefined,
  chainOverride extends Chain | undefined = undefined,
>(
  client: Client<Transport, chain, account>,
  parameters: SendCrossDomainMessageParameters<chain, account, chainOverride>,
): Promise<SendCrossDomainMessageReturnType> {
  const { destinationChainId, target, message, ...txParameters } = parameters

  return baseWriteAction(
    client,
    {
      abi: l2ToL2CrossDomainMessengerAbi,
      contractAddress: contracts.l2ToL2CrossDomainMessenger.address,
      contractFunctionName: 'sendMessage',
      contractArgs: [destinationChainId, target, message],
    },
    txParameters as BaseWriteContractActionParameters,
  )
}

/**
 * Estimates gas for {@link sendMessage}
 * @category Actions
 * @param client - L2 Client
 * @param parameters - {@link SendCrossDomainMessageParameters}
 * @returns estimated gas value.
 */
export async function estimateSendCrossDomainMessageGas<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
  TChainOverride extends Chain | undefined = undefined,
>(
  client: Client<Transport, TChain, TAccount>,
  parameters: SendCrossDomainMessageParameters<
    TChain,
    TAccount,
    TChainOverride
  >,
): Promise<bigint> {
  const { destinationChainId, target, message, ...txParameters } = parameters

  return estimateContractGas(client, {
    abi: l2ToL2CrossDomainMessengerAbi,
    address: contracts.l2ToL2CrossDomainMessenger.address,
    functionName: 'sendMessage',
    args: [destinationChainId, target, message],
    ...txParameters,
  } as EstimateContractGasParameters)
}

/**
 * Simulate contract call for {@link sendMessage}
 * @category Actions
 * @param client - L2 Client
 * @param parameters - {@link SendCrossDomainMessageParameters}
 * @returns contract return value - {@link SendCrossDomainMessageContractReturnType}
 */
export async function simulateSendCrossDomainMessage<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
  TChainOverride extends Chain | undefined = undefined,
>(
  client: Client<Transport, TChain, TAccount>,
  parameters: SendCrossDomainMessageParameters<
    TChain,
    TAccount,
    TChainOverride
  >,
): Promise<SendCrossDomainMessageContractReturnType> {
  const { account, destinationChainId, target, message } = parameters

  const res = await simulateContract(client, {
    account,
    abi: l2ToL2CrossDomainMessengerAbi,
    address: contracts.l2ToL2CrossDomainMessenger.address,
    chain: client.chain,
    functionName: 'sendMessage',
    args: [destinationChainId, target, message],
  } as SimulateContractParameters)

  return res.result as SendCrossDomainMessageContractReturnType
}
