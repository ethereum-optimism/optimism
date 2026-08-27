import type {
  Abi,
  AbiStateMutability,
  Account,
  Address,
  Chain,
  Client,
  ContractFunctionArgs,
  ContractFunctionName,
  DeriveChain,
  EstimateContractGasParameters,
  FormattedTransactionRequest,
  GetChainParameter,
  Hash,
  Transport,
  UnionOmit,
  WriteContractParameters,
} from 'viem'
import { estimateContractGas, writeContract } from 'viem/actions'

import type { GetAccountParameter, UnionEvaluate } from '@/types/utils.js'

export type BaseWriteContractActionParameters<
  chain extends Chain | undefined = Chain | undefined,
  account extends Account | undefined = Account | undefined,
  chainOverride extends Chain | undefined = Chain | undefined,
  derivedChain extends Chain | undefined = DeriveChain<chain, chainOverride>,
> = UnionEvaluate<
  UnionOmit<
    FormattedTransactionRequest<derivedChain>,
    | 'blobs'
    | 'data'
    | 'from'
    | 'gas'
    | 'maxFeePerBlobGas'
    | 'gasPrice'
    | 'to'
    | 'type'
  >
> &
  GetAccountParameter<account, Account | Address> &
  GetChainParameter<chain, chainOverride> & {
    /** Gas limit for transaction execution on the L1. `null` to skip gas estimation & defer calculation to signer. */
    gas?: bigint | null | undefined
  }

export type ContractParameters = {
  abi: Abi
  contractAddress: Address
  contractFunctionName: ContractFunctionName<Abi, AbiStateMutability>
  contractArgs: ContractFunctionArgs<
    Abi,
    AbiStateMutability,
    ContractFunctionName<Abi, AbiStateMutability>
  >
}

export async function baseWriteAction<
  TChain extends Chain | undefined = Chain | undefined,
  TAccount extends Account | undefined = Account | undefined,
  TChainOverride extends Chain | undefined = Chain | undefined,
  TDerivedChain extends Chain | undefined = DeriveChain<TChain, TChainOverride>,
>(
  client: Client<Transport, TChain, TAccount>,
  contractParameters: ContractParameters,
  parameters: BaseWriteContractActionParameters<
    TChain,
    TAccount,
    TChainOverride,
    TDerivedChain
  >,
): Promise<Hash> {
  const { abi, contractAddress, contractFunctionName, contractArgs } =
    contractParameters

  const {
    account,
    chain = client.chain,
    gas,
    maxFeePerGas,
    maxPriorityFeePerGas,
    nonce,
    value,
    accessList,
  } = parameters

  const gas_ =
    typeof gas !== 'bigint' && gas !== null
      ? await estimateContractGas(client, {
          account: account!,
          abi,
          address: contractAddress,
          chain,
          functionName: contractFunctionName,
          args: contractArgs,
          maxFeePerGas,
          maxPriorityFeePerGas,
          nonce,
          value,
          accessList,
        } as EstimateContractGasParameters)
      : gas ?? undefined

  return writeContract(client, {
    account: account!,
    abi,
    address: contractAddress,
    chain,
    functionName: contractFunctionName,
    args: contractArgs,
    gas: gas_,
    maxFeePerGas,
    maxPriorityFeePerGas,
    nonce,
    value,
    accessList,
  } satisfies WriteContractParameters as any)
}
