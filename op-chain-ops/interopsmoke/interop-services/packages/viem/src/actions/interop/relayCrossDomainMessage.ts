import type {
  Account,
  Chain,
  Client,
  ContractFunctionReturnType,
  DeriveChain,
  EstimateContractGasErrorType,
  EstimateContractGasParameters,
  Hash,
  SimulateContractParameters,
  Transport,
  WriteContractErrorType,
} from 'viem'
import { estimateContractGas, simulateContract } from 'viem/actions'

import { l2ToL2CrossDomainMessengerAbi } from '@/abis/index.js'
import type { BuildExecutingMessageReturnType } from '@/actions/interop/buildExecutingMessage.js'
import { contracts } from '@/contracts.js'
import {
  baseWriteAction,
  type BaseWriteContractActionParameters,
} from '@/core/baseWriteAction.js'
import type { ErrorType } from '@/types/utils.js'

/**
 * @category Types
 */
export type RelayCrossDomainMessageParameters<
  TChain extends Chain | undefined = Chain | undefined,
  TAccount extends Account | undefined = Account | undefined,
  TChainOverride extends Chain | undefined = Chain | undefined,
  TDerivedChain extends Chain | undefined = DeriveChain<TChain, TChainOverride>,
> = BaseWriteContractActionParameters<
  TChain,
  TAccount,
  TChainOverride,
  TDerivedChain
> &
  /** executing message built from the sent message log */
  BuildExecutingMessageReturnType

/**
 * @category Types
 */
export type RelayCrossDomainMessageReturnType = Hash

/**
 * @category Types
 */
export type RelayCrossDomainMessageContractReturnType =
  ContractFunctionReturnType<
    typeof l2ToL2CrossDomainMessengerAbi,
    'payable',
    'relayMessage'
  >

/**
 * @category Types
 */
export type RelayCrossDomainMessageErrorType =
  | EstimateContractGasErrorType
  | WriteContractErrorType
  | ErrorType

/**
 * Relays a message emitted by the CrossDomainMessenger
 * @category Actions
 * @param client - L2 Client
 * @param parameters - {@link RelayCrossDomainMessageParameters}
 * @returns transaction hash - {@link RelayCrossDomainMessageReturnType}
 * @example
 * import { createPublicClient } from 'viem'
 * import { http } from 'viem/transports'
 * import { op, unichain } from '@eth-optimism/viem/chains'
 *
 * const publicClientOp = createPublicClient({ chain: op, transport: http() })
 * const publicClientUnichain = createPublicClient({ chain: unichain, transport: http() })
 *
 * const receipt = await publicClientOp.getTransactionReceipt({ hash: '0x...' })
 * const messages = await getCrossDomainMessages(publicClientOp, { logs: receipt.logs })
 *
 * const message = messages.filter((message) => message.destination === unichain.id)[0]
 * const params = await buildExecutingMessage(publicClientOp, { log: message.log })
 *
 * const hash = await relayCrossDomainMessage(publicClientUnichain, params)
 */
export async function relayCrossDomainMessage<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
  TChainOverride extends Chain | undefined = undefined,
>(
  client: Client<Transport, TChain, TAccount>,
  parameters: RelayCrossDomainMessageParameters<
    TChain,
    TAccount,
    TChainOverride
  >,
): Promise<RelayCrossDomainMessageReturnType> {
  const { id, payload, ...txParameters } = parameters

  return baseWriteAction(
    client,
    {
      abi: l2ToL2CrossDomainMessengerAbi,
      contractAddress: contracts.l2ToL2CrossDomainMessenger.address,
      contractFunctionName: 'relayMessage',
      contractArgs: [id, payload],
    },
    txParameters as BaseWriteContractActionParameters,
  )
}

/**
 * Estimates gas for {@link relayCrossDomainMessage}
 * @category Actions
 * @param client - L2 Client
 * @param parameters - {@link RelayCrossDomainMessageParameters}
 * @returns estimated gas value.
 */
export async function estimateRelayCrossDomainMessageGas<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
  TChainOverride extends Chain | undefined = undefined,
>(
  client: Client<Transport, TChain, TAccount>,
  parameters: RelayCrossDomainMessageParameters<
    TChain,
    TAccount,
    TChainOverride
  >,
): Promise<bigint> {
  const { id, payload, ...txParameters } = parameters

  return estimateContractGas(client, {
    abi: l2ToL2CrossDomainMessengerAbi,
    address: contracts.l2ToL2CrossDomainMessenger.address,
    functionName: 'relayMessage',
    args: [id, payload],
    ...txParameters,
  } as EstimateContractGasParameters)
}

/**
 * Simulate contract call for {@link relayCrossDomainMessage}
 * @category Actions
 * @param client - L2 Client
 * @param parameters - {@link RelayCrossDomainMessageParameters}
 * @returns contract return value - {@link RelayCrossDomainMessageContractReturnType}
 */
export async function simulateRelayCrossDomainMessage<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
  TChainOverride extends Chain | undefined = undefined,
>(
  client: Client<Transport, TChain, TAccount>,
  parameters: RelayCrossDomainMessageParameters<
    TChain,
    TAccount,
    TChainOverride
  >,
): Promise<RelayCrossDomainMessageContractReturnType> {
  const { account, id, payload, accessList } = parameters

  const res = await simulateContract(client, {
    account,
    abi: l2ToL2CrossDomainMessengerAbi,
    address: contracts.l2ToL2CrossDomainMessenger.address,
    chain: client.chain,
    functionName: 'relayMessage',
    args: [id, payload],
    accessList,
  } as SimulateContractParameters)

  return res.result as RelayCrossDomainMessageContractReturnType
}
