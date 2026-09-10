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
import { parseAccount } from 'viem/accounts'
import {
  estimateContractGas,
  readContract,
  simulateContract,
} from 'viem/actions'

import { optimismMintableERC20Abi, standardBridgeAbi } from '@/abis/index.js'
import { contracts } from '@/contracts.js'
import type { BaseWriteContractActionParameters } from '@/core/baseWriteAction.js'
import { baseWriteAction } from '@/core/baseWriteAction.js'

/**
 * Parameters to withdraw an OptimismMintableERC20 | OptimismSuperchainERC20 into the remote token.
 * @category Types
 */
export type WithdrawOptimismERC20Parameters<
  TChain extends Chain | undefined = Chain | undefined,
  TAccount extends Account | undefined = Account | undefined,
  TChainOverride extends Chain | undefined = Chain | undefined,
  TDerivedChain extends Chain | undefined = DeriveChain<TChain, TChainOverride>,
> = GetChainParameter<TChain, TChainOverride> &
  BaseWriteContractActionParameters<
    TChain,
    TAccount,
    TChainOverride,
    TDerivedChain
  > & {
    /** The address of the OptimismMintablERC20 token to withdraw */
    tokenAddress: Address
    /** The token amount to bridge */
    amount: bigint
    /** The recipient address to bridge to. Defaults to the calling account */
    to?: Address
    /** The minimums gas the relaying message will be executed with */
    minGasLimit?: number
    /** Metadata to attach to the bridged message */
    extraData?: Hex
    /** The address of the StandardBridge to use. Defaults to the L2StandardBridge Predeploy */
    bridgeAddress?: Address
  }

/**
 * Transaction hash of the withdrawing transaction.
 * @category Types
 */
export type WithdrawOptimismERC20ReturnType = Hash

/**
 * Return type of the StandardBridge bridgeERC20To function.
 * @category Types
 */
export type WithdrawOptimismERC20ContractReturnType =
  ContractFunctionReturnType<
    typeof standardBridgeAbi,
    'nonpayable',
    'bridgeERC20To'
  >

/**
 * Action to withdraw an OptimismMintableERC20 | OptimismSuperchainERC20 into its remote ERC20.
 * @category Actions
 * @param client - Client for the withdrawing chain
 * @param parameters - {@link WithdrawOptimismERC20Parameters}
 * @returns The hash of the withdrawing transaction
 * @example
 * import { withdrawOptimismERC20 } from '@eth-optimism/viem'
 * import { op } from '@eth-optimism/viem/chains'
 *
 * const client = createPublicClient({ chain: op, transport: http() })
 * const hash = await withdrawOptimismERC20(client, {
 *   tokenAddress: '0x0000000000000000000000000000000000000000',
 *   amount: 1000000000000000000n,
 * })
 */
export async function withdrawOptimismERC20<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
  TChainOverride extends Chain | undefined,
>(
  client: Client<Transport, TChain, TAccount>,
  parameters: WithdrawOptimismERC20Parameters<TChain, TAccount, TChainOverride>,
): Promise<Hash> {
  const {
    to = parameters.to ?? parseAccount(parameters.account!).address,
    minGasLimit = parameters.minGasLimit ?? 0,
    extraData = parameters.extraData ?? '0x',
    tokenAddress: localTokenAddress,
    amount,
  } = parameters

  const remoteTokenAddress = await readContract(client, {
    abi: optimismMintableERC20Abi,
    address: localTokenAddress,
    functionName: 'remoteToken',
  })

  const bridgeAddress =
    parameters.bridgeAddress ?? contracts.l2StandardBridge.address

  return baseWriteAction(
    client,
    {
      abi: standardBridgeAbi,
      contractAddress: bridgeAddress,
      contractFunctionName: 'bridgeERC20To',
      contractArgs: [
        localTokenAddress,
        remoteTokenAddress,
        to,
        amount,
        minGasLimit,
        extraData,
      ],
    },
    parameters as BaseWriteContractActionParameters,
  )
}

/**
 * Estimate the gas cost of the {@link withdrawOptimismERC20} action.
 * @category Actions
 * @param client - Client for the withdrawing chain
 * @param parameters - {@link WithdrawOptimismERC20Parameters}
 * @returns The gas cost
 */
export async function estimateWithdrawOptimismERC20Gas<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
  TChainOverride extends Chain | undefined,
>(
  client: Client<Transport, TChain, TAccount>,
  parameters: WithdrawOptimismERC20Parameters<TChain, TAccount, TChainOverride>,
) {
  const {
    to = parameters.to ?? parseAccount(parameters.account!).address,
    minGasLimit = parameters.minGasLimit ?? 0,
    extraData = parameters.extraData ?? '0x',
    tokenAddress: localTokenAddress,
    amount,
  } = parameters

  const remoteTokenAddress = await readContract(client, {
    abi: optimismMintableERC20Abi,
    address: localTokenAddress,
    functionName: 'remoteToken',
  })

  const bridgeAddress =
    parameters.bridgeAddress ?? contracts.l2StandardBridge.address

  return estimateContractGas(client, {
    abi: standardBridgeAbi,
    address: bridgeAddress,
    functionName: 'bridgeERC20To',
    args: [
      localTokenAddress,
      remoteTokenAddress,
      to,
      amount,
      minGasLimit,
      extraData,
    ],
    ...parameters,
  } as EstimateContractGasParameters)
}

/**
 * Simulate the {@link withdrawOptimismERC20} action.
 * @category Actions
 * @param client - Client for the withdrawing chain
 * @param parameters - {@link WithdrawOptimismERC20Parameters}
 * @returns The simulated transaction
 */
export async function simulateWithdrawOptimismERC20<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
  TChainOverride extends Chain | undefined,
>(
  client: Client<Transport, TChain, TAccount>,
  parameters: WithdrawOptimismERC20Parameters<TChain, TAccount, TChainOverride>,
): Promise<WithdrawOptimismERC20ContractReturnType> {
  const {
    to = parameters.to ?? parseAccount(parameters.account!).address,
    minGasLimit = parameters.minGasLimit ?? 0,
    extraData = parameters.extraData ?? '0x',
    tokenAddress: localTokenAddress,
    amount,
  } = parameters

  const remoteTokenAddress = await readContract(client, {
    abi: optimismMintableERC20Abi,
    address: localTokenAddress,
    functionName: 'remoteToken',
  })

  const bridgeAddress =
    parameters.bridgeAddress ?? contracts.l2StandardBridge.address

  const { result } = await simulateContract(client, {
    abi: standardBridgeAbi,
    address: bridgeAddress,
    functionName: 'bridgeERC20To',
    args: [
      localTokenAddress,
      remoteTokenAddress,
      to,
      amount,
      minGasLimit,
      extraData,
    ],
    ...parameters,
  } as SimulateContractParameters)

  return result as WithdrawOptimismERC20ContractReturnType
}
