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
import { BaseError, createPublicClient, http } from 'viem'
import { parseAccount } from 'viem/accounts'
import {
  estimateContractGas,
  readContract,
  simulateContract,
} from 'viem/actions'

import { optimismMintableERC20Abi, standardBridgeAbi } from '@/abis/index.js'
import type { BaseWriteContractActionParameters } from '@/core/baseWriteAction.js'
import { baseWriteAction } from '@/core/baseWriteAction.js'
import type { GetContractAddressParameter } from '@/types/utils.js'

/**
 * Action to deposit an ERC20 into an OptimismMintableERC20 | OptimismSuperchainERC20 counterpart.
 * Unless `unsafe` is set or the `targetChain` hasn't been specified, the remote token address will
 * be checked to ensure a pair-wise relationship between the tokens. Specify `remoteClient` to use an
 * already constructed client for the destination chain.
 * @category Types
 */
export type DepositERC20Parameters<
  TChain extends Chain | undefined = Chain | undefined,
  TAccount extends Account | undefined = Account | undefined,
  TChainOverride extends Chain | undefined = Chain | undefined,
  TDerivedChain extends Chain | undefined = DeriveChain<TChain, TChainOverride>,
> = GetChainParameter<TChain, TChainOverride> &
  GetContractAddressParameter<TDerivedChain, 'l1StandardBridge'> &
  BaseWriteContractActionParameters<
    TChain,
    TAccount,
    TChainOverride,
    TDerivedChain
  > & {
    /** The address of the ERC20 to bridge */
    tokenAddress: Address
    /** The address of the OptimismMintableERC20 | OptimismSuperchainERC20 to bridge into */
    remoteTokenAddress: Address
    /** The amount of tokens to bridge */
    amount: bigint
    /** The recipient address to bridge to. Defaults to the calling account */
    to?: Address
    /** The minimums gas the relaying message will be executed with */
    minGasLimit?: number
    /** Metadata to attach to the bridged message */
    extraData?: Hex

    /** Client to use for the destination chain safety check */
    remoteClient?: Client
    /** Whether to skip the remote token check on the destination chain */
    unsafe?: boolean
  }

/**
 * Transaction hash of the depositing transaction.
 * @category Types
 */
export type DepositERC20ReturnType = Hash

/**
 * Return type of the StandardBridge bridgeERC20To function.
 * @category Types
 */
export type DepositERC20ContractReturnType = ContractFunctionReturnType<
  typeof standardBridgeAbi,
  'nonpayable',
  'bridgeERC20To'
>

export type DepositERC20RemoteTokenMismatchErrorType =
  DepositERC20RemoteTokenMismatchError & {
    name: 'DepositERC20RemoteTokenMismatchError'
  }
export class DepositERC20RemoteTokenMismatchError extends BaseError {
  constructor(local: Address, remote: Address) {
    super(
      `OptimismMintableERC20 remote token address mismatch. Local: ${local}, Remote: ${remote}`,
    )
  }
}

/**
 * Deposit an ERC20 into an OptimismMintableERC20 | OptimismSuperchainERC20.
 * @category Actions
 * @param client - Client for the depositing chain
 * @param parameters - {@link DepositERC20Parameters}
 * @returns The transaction hash. {@link DepositERC20ReturnType}
 * @example
 * import { depositERC20 } from '@eth-optimism/viem'
 * import { op } from '@eth-optimism/viem/chains'
 *
 * const hash = await depositERC20(client, {
 *   tokenAddress: '0x0000000000000000000000000000000000000000',
 *   remoteTokenAddress: '0x0000000000000000000000000000000000000000',
 *   amount: 1000000000000000000n,
 *   targetChain: op,
 * })
 */
export async function depositERC20<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
  TChainOverride extends Chain | undefined,
>(
  client: Client<Transport, TChain, TAccount>,
  parameters: DepositERC20Parameters<TChain, TAccount, TChainOverride>,
): Promise<Hash> {
  const {
    chain = client.chain,
    to = parameters.to ?? parseAccount(parameters.account!).address,
    minGasLimit = parameters.minGasLimit ?? 0,
    extraData = parameters.extraData ?? '0x',
    tokenAddress: localTokenAddress,
    remoteTokenAddress,
    amount,
    targetChain,
    unsafe,
  } = parameters

  const bridgeAddress = (() => {
    if (parameters.l1StandardBridgeAddress)
      return parameters.l1StandardBridgeAddress
    if (chain) return targetChain!.contracts.l1StandardBridge[chain.id].address
    return Object.values(targetChain!.contracts.l1StandardBridge)[0].address
  })()

  if (!unsafe && targetChain) {
    const remoteClient =
      parameters.remoteClient ??
      createPublicClient({ chain: targetChain, transport: http() })

    const _localTokenAddress = await readContract(remoteClient, {
      abi: optimismMintableERC20Abi,
      address: remoteTokenAddress,
      functionName: 'remoteToken',
    })

    if (_localTokenAddress !== localTokenAddress) {
      throw new DepositERC20RemoteTokenMismatchError(
        localTokenAddress,
        _localTokenAddress,
      )
    }
  }

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
 * Estimate the gas cost of the {@link depositERC20} action.
 * @category Actions
 * @param client - Client for the depositing chain
 * @param parameters - {@link DepositERC20Parameters}
 * @returns The gas cost
 */
export async function estimateDepositERC20Gas<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
  TChainOverride extends Chain | undefined,
>(
  client: Client<Transport, TChain, TAccount>,
  parameters: DepositERC20Parameters<TChain, TAccount, TChainOverride>,
): Promise<bigint> {
  const {
    chain = client.chain,
    to = parameters.to ?? parseAccount(parameters.account!).address,
    minGasLimit = parameters.minGasLimit ?? 0,
    extraData = parameters.extraData ?? '0x',
    tokenAddress: localTokenAddress,
    remoteTokenAddress,
    amount,
    targetChain,
  } = parameters

  const bridgeAddress = (() => {
    if (parameters.l1StandardBridgeAddress)
      return parameters.l1StandardBridgeAddress
    if (chain) return targetChain!.contracts.l1StandardBridge[chain.id].address
    return Object.values(targetChain!.contracts.l1StandardBridge)[0].address
  })()

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
 * Simulate the {@link depositERC20} action.
 * @category Actions
 * @param client - Client for the depositing chain
 * @param parameters - {@link DepositERC20Parameters}
 * @returns The contract functions return value. {@link DepositERC20ContractReturnType}
 */
export async function simulateDepositERC20<
  TChain extends Chain | undefined,
  TAccount extends Account | undefined,
  TChainOverride extends Chain | undefined,
>(
  client: Client<Transport, TChain, TAccount>,
  parameters: DepositERC20Parameters<TChain, TAccount, TChainOverride>,
): Promise<DepositERC20ContractReturnType> {
  const {
    chain = client.chain,
    to = parameters.to ?? parseAccount(parameters.account!).address,
    minGasLimit = parameters.minGasLimit ?? 0n,
    extraData = parameters.extraData ?? '0x',
    tokenAddress: localTokenAddress,
    remoteTokenAddress,
    amount,
    targetChain,
  } = parameters

  const bridgeAddress = (() => {
    if (parameters.l1StandardBridgeAddress)
      return parameters.l1StandardBridgeAddress
    if (chain) return targetChain!.contracts.l1StandardBridge[chain.id].address
    return Object.values(targetChain!.contracts.l1StandardBridge)[0].address
  })()

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

  return result as DepositERC20ContractReturnType
}
