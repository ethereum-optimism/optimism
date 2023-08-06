import {
  gasPriceOracleABI,
  gasPriceOracleAddress,
} from '@eth-optimism/contracts-ts'
import {
  getContract,
  createPublicClient,
  http,
  BlockTag,
  Address,
  EstimateGasParameters,
  serializeTransaction,
  encodeFunctionData,
  EncodeFunctionDataParameters,
  TransactionSerializableEIP1559,
  TransactionSerializedEIP1559,
  PublicClient,
} from 'viem'
import * as chains from 'viem/chains'
import { Abi } from 'abitype'

/**
 * Bytes type representing a hex string with a 0x prefix
 * @typedef {`0x${string}`} Bytes
 */
export type Bytes = `0x${string}`

/**
 * Options to query a specific block
 */
type BlockOptions = {
  /**
   * Block number to query from
   */
  blockNumber?: bigint
  /**
   * Block tag to query from
   */
  blockTag?: BlockTag
}

const knownChains = [
  chains.optimism.id,
  chains.goerli.id,
  chains.base,
  chains.baseGoerli.id,
  chains.zora,
  chains.zoraTestnet,
]

/**
 * ClientOptions type
 * @typedef {Object} ClientOptions
 * @property {keyof typeof gasPriceOracleAddress | number} chainId - Chain ID
 * @property {string} [rpcUrl] - RPC URL. If not provided the provider will attempt to use public RPC URLs for the chain
 * @property {chains.Chain['nativeCurrency']} [nativeCurrency] - Native currency. Defaults to ETH
 */
type ClientOptions =
  // for known chains like base don't require an rpcUrl
  | {
      chainId: typeof knownChains[number]
      rpcUrl?: string
      nativeCurrency?: chains.Chain['nativeCurrency']
    }
  | {
      chainId: number
      rpcUrl: string
      nativeCurrency?: chains.Chain['nativeCurrency']
    }
  | PublicClient

/**
 * Options for all GasPriceOracle methods
 */
export type GasPriceOracleOptions = BlockOptions & { client: ClientOptions }

/**
 * Options for specifying the transaction being estimated
 */
export type OracleTransactionParameters<
  TAbi extends Abi | readonly unknown[],
  TFunctionName extends string | undefined = undefined
> = EncodeFunctionDataParameters<TAbi, TFunctionName> &
  Omit<TransactionSerializableEIP1559, 'data' | 'type'>
/**
 * Options for specifying the transaction being estimated
 */
export type GasPriceOracleEstimator = <
  TAbi extends Abi | readonly unknown[],
  TFunctionName extends string | undefined = undefined
>(
  options: OracleTransactionParameters<TAbi, TFunctionName> &
    GasPriceOracleOptions
) => Promise<bigint>

/**
 * Throws an error if fetch is not defined
 * Viem requires fetch
 */
const validateFetch = (): void => {
  if (typeof fetch === 'undefined') {
    throw new Error(
      'No fetch implementation found. Please provide a fetch polyfill. This can be done in NODE by passing in NODE_OPTIONS=--experimental-fetch or by using the isomorphic-fetch npm package'
    )
  }
}

/**
 * Internal helper to serialize a transaction
 */
const transactionSerializer = <
  TAbi extends Abi | readonly unknown[],
  TFunctionName extends string | undefined = undefined
>(
  options: EncodeFunctionDataParameters<TAbi, TFunctionName> &
    Omit<TransactionSerializableEIP1559, 'data'>
): TransactionSerializedEIP1559 => {
  const encodedFunctionData = encodeFunctionData(options)
  const serializedTransaction = serializeTransaction({
    ...options,
    data: encodedFunctionData,
    type: 'eip1559',
  })
  return serializedTransaction as TransactionSerializedEIP1559
}

/**
 * Gets L2 client
 * @example
 * const client = getL2Client({ chainId: 1, rpcUrl: "http://localhost:8545" });
 */
export const getL2Client = (options: ClientOptions): PublicClient => {
  validateFetch()

  if ('chainId' in options && options.chainId) {
    const viemChain = Object.values(chains)?.find(
      (chain) => chain.id === options.chainId
    )
    const rpcUrls = options.rpcUrl
      ? { default: { http: [options.rpcUrl] } }
      : viemChain?.rpcUrls
    if (!rpcUrls) {
      throw new Error(
        `No rpcUrls found for chainId ${options.chainId}.  Please explicitly provide one`
      )
    }
    return createPublicClient({
      chain: {
        id: options.chainId,
        name: viemChain?.name ?? 'op-chain',
        nativeCurrency:
          options.nativeCurrency ??
          viemChain?.nativeCurrency ??
          chains.optimism.nativeCurrency,
        network: viemChain?.network ?? 'Unknown OP Chain',
        rpcUrls,
        explorers:
          (viemChain as typeof chains.optimism)?.blockExplorers ??
          chains.optimism.blockExplorers,
      },
      transport: http(
        options.rpcUrl ?? chains[options.chainId].rpcUrls.public.http[0]
      ),
    })
  }
  return options as PublicClient
}

/**
 * Get gas price Oracle contract
 */
export const getGasPriceOracleContract = (params: ClientOptions) => {
  return getContract({
    address: gasPriceOracleAddress['420'],
    abi: gasPriceOracleABI,
    publicClient: getL2Client(params),
  })
}

/**
 * Returns the base fee
 * @returns {Promise<bigint>} - The base fee
 * @example
 * const baseFeeValue = await baseFee(params);
 */
export const baseFee = async ({
  client,
  blockNumber,
  blockTag,
}: GasPriceOracleOptions): Promise<bigint> => {
  const contract = getGasPriceOracleContract(client)
  return contract.read.baseFee({ blockNumber, blockTag })
}

/**
 * Returns the decimals used in the scalar
 * @example
 * const decimalsValue = await decimals(params);
 */
export const decimals = async ({
  client,
  blockNumber,
  blockTag,
}: GasPriceOracleOptions): Promise<bigint> => {
  const contract = getGasPriceOracleContract(client)
  return contract.read.decimals({ blockNumber, blockTag })
}

/**
 * Returns the gas price
 * @example
 * const gasPriceValue = await gasPrice(params);
 */
export const gasPrice = async ({
  client,
  blockNumber,
  blockTag,
}: GasPriceOracleOptions): Promise<bigint> => {
  const contract = getGasPriceOracleContract(client)
  return contract.read.gasPrice({ blockNumber, blockTag })
}

/**
 * Computes the L1 portion of the fee based on the size of the rlp encoded input
 * transaction, the current L1 base fee, and the various dynamic parameters.
 * @example
 * const L1FeeValue = await getL1Fee(data, params);
 */
export const getL1Fee: GasPriceOracleEstimator = async (options) => {
  const data = transactionSerializer(options)
  const contract = getGasPriceOracleContract(options.client)
  return contract.read.getL1Fee([data], {
    blockNumber: options.blockNumber,
    blockTag: options.blockTag,
  })
}

/**
 * Returns the L1 gas used
 * @example
 */
export const getL1GasUsed: GasPriceOracleEstimator = async (options) => {
  const data = transactionSerializer(options)
  const contract = getGasPriceOracleContract(options.client)
  return contract.read.getL1GasUsed([data], {
    blockNumber: options.blockNumber,
    blockTag: options.blockTag,
  })
}

/**
 * Returns the L1 base fee
 * @example
 * const L1BaseFeeValue = await l1BaseFee(params);
 */
export const l1BaseFee = async ({
  client,
  blockNumber,
  blockTag,
}: GasPriceOracleOptions): Promise<bigint> => {
  const contract = getGasPriceOracleContract(client)
  return contract.read.l1BaseFee({ blockNumber, blockTag })
}

/**
 * Returns the overhead
 * @example
 * const overheadValue = await overhead(params);
 */
export const overhead = async ({
  client,
  blockNumber,
  blockTag,
}: GasPriceOracleOptions): Promise<bigint> => {
  const contract = getGasPriceOracleContract(client)
  return contract.read.overhead({ blockNumber, blockTag })
}

/**
 * Returns the current fee scalar
 * @example
 * const scalarValue = await scalar(params);
 */
export const scalar = async ({
  client,
  ...params
}: GasPriceOracleOptions): Promise<bigint> => {
  const contract = getGasPriceOracleContract(client)
  return contract.read.scalar(params)
}

/**
 * Returns the version
 * @example
 * const versionValue = await version(params);
 */
export const version = async ({
  client,
  ...params
}: GasPriceOracleOptions): Promise<string> => {
  const contract = getGasPriceOracleContract(client)
  return contract.read.version(params)
}

export type EstimateFeeParams = {
  /**
   * The transaction call data as a 0x-prefixed hex string
   */
  data: Bytes
  /**
   * The address of the account that will be sending the transaction
   */
  account: Address
} & GasPriceOracleOptions &
  Omit<EstimateGasParameters, 'data' | 'account'>

export type EstimateFees = <
  TAbi extends Abi | readonly unknown[],
  TFunctionName extends string | undefined = undefined
>(
  options: OracleTransactionParameters<TAbi, TFunctionName> &
    GasPriceOracleOptions &
    Omit<EstimateGasParameters, 'data'>
) => Promise<bigint>
/**
 * Estimates gas for an L2 transaction including the l1 fee
 */
export const estimateFees: EstimateFees = async (options) => {
  const client = getL2Client(options.client)
  const encodedFunctionData = encodeFunctionData({
    abi: options.abi,
    args: options.args,
    functionName: options.functionName,
  } as EncodeFunctionDataParameters)
  const [l1Fee, l2Gas, l2GasPrice] = await Promise.all([
    getL1Fee({
      ...options,
      // account must be undefined or else viem will return undefined
      account: undefined as any,
    }),
    client.estimateGas({
      to: options.to,
      account: options.account,
      accessList: options.accessList,
      blockNumber: options.blockNumber,
      blockTag: options.blockTag,
      data: encodedFunctionData,
      value: options.value,
    } as EstimateGasParameters<typeof chains.optimism>),
    client.getGasPrice(),
  ])
  return l1Fee + l2Gas * l2GasPrice
}
