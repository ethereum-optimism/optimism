/** @module sendSuperchainERC20 */
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
  SimulateContractParameters,
  Transport,
  WriteContractErrorType,
} from 'viem'
import { estimateContractGas, simulateContract } from 'viem/actions'

import { superchainTokenBridgeAbi } from '@/abis/index.js'
import { contracts } from '@/contracts.js'
import type { BaseWriteContractActionParameters } from '@/core/baseWriteAction.js'
import { baseWriteAction } from '@/core/baseWriteAction.js'
import type { ErrorType } from '@/types/utils.js'

/**
 * @category Types
 */
export type SendSuperchainERC20Parameters<
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
  /** Token to send. */
  tokenAddress: Address
  /** Address to send tokens to. */
  to: Address
  /** Amount of tokens to send. */
  amount: bigint
  /** Chain ID of the destination chain. */
  chainId: number
}

/**
 * @category Types
 */
export type SendSuperchainERC20ReturnType = Hash

/**
 * @category Types
 */
export type SendSuperchainERC20ContractReturnType = ContractFunctionReturnType<
  typeof superchainTokenBridgeAbi,
  'nonpayable',
  'sendERC20'
>

/**
 * @category Types
 */
export type SendSuperchainERC20ErrorType =
  | EstimateContractGasErrorType
  | WriteContractErrorType
  | ErrorType

/**
 * Sends tokens to a target address on another chain. Used in the interop flow.
 * @category Actions
 * @param client - L2 Client
 * @param parameters - {@link SendSuperchainERC20Parameters}
 * @returns transaction hash - {@link SendSuperchainERC20ReturnType}
 */
export async function sendSuperchainERC20<
  chain extends Chain | undefined,
  account extends Account | undefined,
  chainOverride extends Chain | undefined = undefined,
>(
  client: Client<Transport, chain, account>,
  parameters: SendSuperchainERC20Parameters<chain, account, chainOverride>,
): Promise<SendSuperchainERC20ReturnType> {
  const { tokenAddress, to, amount, chainId, ...txParameters } = parameters

  return baseWriteAction(
    client,
    {
      abi: superchainTokenBridgeAbi,
      contractAddress: contracts.superchainTokenBridge.address,
      contractFunctionName: 'sendERC20',
      contractArgs: [tokenAddress, to, amount, BigInt(chainId)],
    },
    txParameters as BaseWriteContractActionParameters,
  )
}

/**
 * Estimates gas for {@link sendSuperchainERC20}
 * @category Actions
 * @param client - L2 Client
 * @param parameters - {@link SendSuperchainERC20Parameters}
 * @returns estimated gas value.
 */
export async function estimateSendSuperchainERC20Gas<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
  TChainOverride extends Chain | undefined = undefined,
>(
  client: Client<Transport, TChain, TAccount>,
  parameters: SendSuperchainERC20Parameters<TChain, TAccount, TChainOverride>,
): Promise<bigint> {
  const { tokenAddress, to, amount, chainId, ...txParameters } = parameters

  return estimateContractGas(client, {
    abi: superchainTokenBridgeAbi,
    address: contracts.superchainTokenBridge.address,
    functionName: 'sendERC20',
    args: [tokenAddress, to, amount, BigInt(chainId)],
    ...txParameters,
  } as EstimateContractGasParameters)
}

/**
 * Simulate contract call for {@link sendSuperchainERC20}
 * @category Actions
 * @param client - L2 Client
 * @param parameters - {@link SendSuperchainERC20Parameters}
 * @returns contract return value - {@link SendSuperchainERC20ContractReturnType}
 */
export async function simulateSendSuperchainERC20<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
  TChainOverride extends Chain | undefined = undefined,
>(
  client: Client<Transport, TChain, TAccount>,
  parameters: SendSuperchainERC20Parameters<TChain, TAccount, TChainOverride>,
): Promise<SendSuperchainERC20ContractReturnType> {
  const { account, tokenAddress, to, amount, chainId } = parameters

  const res = await simulateContract(client, {
    account,
    abi: superchainTokenBridgeAbi,
    address: contracts.superchainTokenBridge.address,
    chain: client.chain,
    functionName: 'sendERC20',
    args: [tokenAddress, to, amount, BigInt(chainId)],
  } as SimulateContractParameters)

  return res.result as SendSuperchainERC20ContractReturnType
}
