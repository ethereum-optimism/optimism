import { getLogger } from '@eth-optimism/core-utils'
import {
  DEFAULT_OPCODE_WHITELIST_MASK,
  L2_TO_L1_MESSAGE_PASSER_OVM_ADDRESS,
} from '../constants'

const log = getLogger('environment')

/**
 * Class to contain all environment variables referenced by the rollup full node
 * to consolidate access / updates and default values.
 */
export class Environment {
  public static getOrThrow<T>(
    fun: (defaultValue?: T) => T,
    defaultValue?: T,
    logValue: boolean = true
  ): T {
    const res = fun(defaultValue)
    if (!res) {
      throw Error(
        `Expected Environment variable not set. Error calling Environment.${fun.name}()`
      )
    }
    if (logValue) {
      log.info(`Environment: ${fun.name} = ${res}`)
    }
    return res
  }

  public static clearDataKey(defaultValue?: string) {
    return process.env.CLEAR_DATA_KEY || defaultValue
  }

  // Server Type Config
  public static isRoutingServer(defaultValue?: boolean) {
    return !!process.env.IS_ROUTING_SERVER || defaultValue
  }
  public static isTranasactionNode(defaultValue?: boolean) {
    return !!process.env.IS_TRANSACTION_NODE || defaultValue
  }
  public static isReadOnlyNode(defaultValue?: boolean) {
    return !!process.env.IS_READ_ONLY_NODE || defaultValue
  }

  // Routing Server Config
  public static transactionNodeUrl(defaultValue?: string) {
    return process.env.TRANSACTION_NODE_URL || defaultValue
  }
  public static readOnlyNodeUrl(defaultValue?: string) {
    return process.env.READ_ONLY_NODE_URL || defaultValue
  }
  public static maxNonTransactionRequestsPerUnitTime(defaultValue?: number) {
    return process.env.MAX_NON_TRANSACTION_REQUESTS_PER_UNIT_TIME
      ? parseInt(process.env.MAX_NON_TRANSACTION_REQUESTS_PER_UNIT_TIME, 10)
      : defaultValue
  }
  public static maxTransactionsPerUnitTime(defaultValue?: number) {
    return process.env.MAX_TRANSACTIONS_PER_UNIT_TIME
      ? parseInt(process.env.MAX_TRANSACTIONS_PER_UNIT_TIME, 10)
      : defaultValue
  }
  public static requestLimitPeriodMillis(defaultValue?: number) {
    return process.env.REQUEST_LIMIT_PERIOD_MILLIS
      ? parseInt(process.env.REQUEST_LIMIT_PERIOD_MILLIS, 10)
      : defaultValue
  }
  public static contractDeployerAddress(defaultValue?: string) {
    return process.env.CONTRACT_DEPLOYER_ADDRESS || defaultValue
  }
  public static transactionToAddressWhitelist(defaultValue: string[] = []) {
    return process.env.COMMA_SEPARATED_TO_ADDRESS_WHITELIST
      ? process.env.COMMA_SEPARATED_TO_ADDRESS_WHITELIST.split(',')
      : defaultValue
  }
  public static rateLimitWhitelistIpAddresses(defaultValue: string[] = []) {
    return process.env.COMMA_SEPARATED_RATE_LIMIT_WHITELISTED_IPS
      ? process.env.COMMA_SEPARATED_RATE_LIMIT_WHITELISTED_IPS.split(',')
      : defaultValue
  }

  // L2 RPC Server Config
  public static l2RpcServerPersistentDbPath(defaultValue?: string) {
    return process.env.L2_RPC_SERVER_PERSISTENT_DB_PATH || defaultValue
  }
  public static l2RpcServerHost(defaultValue: string = '0.0.0.0'): string {
    return process.env.L2_RPC_SERVER_HOST || defaultValue
  }
  public static l2RpcServerPort(defaultValue: number = 8545): number {
    return process.env.L2_RPC_SERVER_PORT
      ? parseInt(process.env.L2_RPC_SERVER_PORT, 10)
      : defaultValue
  }
  public static noL1Node(defaultValue?: boolean) {
    return !!process.env.NO_L1_NODE || defaultValue
  }
  public static noL1ToL2TransactionProcessor(defaultValue?: boolean) {
    return !!process.env.NO_L1_TO_L2_TX_PROCESSOR || defaultValue
  }

  // Local Node Config
  public static opcodeWhitelistMask(
    defaultValue: string = DEFAULT_OPCODE_WHITELIST_MASK
  ): string {
    return process.env.OPCODE_WHITELIST_MASK || defaultValue
  }
  public static localL2NodePersistentDbPath(defaultValue?: string) {
    return process.env.LOCAL_L2_NODE_PERSISTENT_DB_PATH || defaultValue
  }
  public static localL1NodePersistentDbPath(defaultValue?: string): string {
    return process.env.LOCAL_L1_NODE_PERSISTENT_DB_PATH || defaultValue
  }

  // L2 Config
  public static l2NodeWeb3Url(defaultValue?: string): string {
    return process.env.L2_NODE_WEB3_URL || defaultValue
  }
  public static l2WalletPrivateKey(defaultValue?: string): string {
    return process.env.L2_WALLET_PRIVATE_KEY || defaultValue
  }
  public static l2WalletMnemonic(defaultValue?: string): string {
    return process.env.L2_WALLET_MNEMONIC || defaultValue
  }
  public static l2WalletPrivateKeyPath(defaultValue?: string): string {
    return process.env.L2_WALLET_PRIVATE_KEY_PATH || defaultValue
  }
  public static l2ExecutionManagerAddress(defaultValue?: string): string {
    return process.env.L2_EXECUTION_MANAGER_ADDRESS || defaultValue
  }
  public static l2ToL1MessagePasserOvmAddress(
    defaultValue = L2_TO_L1_MESSAGE_PASSER_OVM_ADDRESS
  ): string {
    return process.env.L2_TO_L1_MESSAGE_PASSER_OVM_ADDRESS || defaultValue
  }
  public static localL2NodePort(defaultValue: number = 9876): number {
    return process.env.LOCAL_L2_NODE_PORT
      ? parseInt(process.env.LOCAL_L2_NODE_PORT, 10)
      : defaultValue
  }

  // L1 Infura
  public static l1NodeInfuraNetwork(defaultValue?: string): string {
    return process.env.L1_NODE_INFURA_NETWORK || defaultValue
  }
  public static l1NodeInfuraProjectId(defaultValue?: string): string {
    return process.env.L1_NODE_INFURA_PROJECT_ID || defaultValue
  }

  // L1 Config
  public static l1NodeWeb3Url(defaultValue?: string): string {
    return process.env.L1_NODE_WEB3_URL || defaultValue
  }
  public static localL1NodePort(defaultValue: number = 7545): number {
    return process.env.LOCAL_L1_NODE_PORT
      ? parseInt(process.env.LOCAL_L1_NODE_PORT, 10)
      : defaultValue
  }
  // TODO: remove default when this matters
  public static sequencerMnemonic(
    defaultValue: string = 'rebel talent argue catalog maple duty file taxi dust hire funny steak'
  ): string {
    return process.env.L1_SEQUENCER_MNEMONIC || defaultValue
  }
  public static sequencerPrivateKey(defaultValue?: string): string {
    return process.env.L1_SEQUENCER_PRIVATE_KEY || defaultValue
  }
  public static l1ToL2TransactionPasserAddress(defaultValue?: string): string {
    return process.env.L1_TO_L2_TRANSACTION_PASSER_ADDRESS || defaultValue
  }
  public static l2ToL1MessageReceiverAddress(defaultValue?: string): string {
    return process.env.L2_TO_L1_MESSAGE_RECEIVER_ADDRESS || defaultValue
  }
  public static l2ToL1MessageFinalityDelayInBlocks(
    defaultValue: number = 0
  ): number {
    return process.env.L2_TO_L1_MESSAGE_FINALITY_DELAY_IN_BLOCKS
      ? parseInt(process.env.L2_TO_L1_MESSAGE_FINALITY_DELAY_IN_BLOCKS, 10)
      : defaultValue
  }
  public static l1EarliestBlock(defaultValue: number = 0): number {
    return process.env.L1_EARLIEST_BLOCK
      ? parseInt(process.env.L1_EARLIEST_BLOCK, 10)
      : defaultValue
  }

  // L1 to L2 Tx Processor Config
  public static l1ToL2TxProcessorPersistentDbPath(defaultValue?: string) {
    return process.env.L1_TO_L2_TX_PROCESSOR_PERSISTENT_DB_PATH || defaultValue
  }
  public static l1ToL2TxProcessorPrivateKey(defaultValue?: string) {
    return process.env.L1_TO_L2_TX_PROCESSOR_PRIVATE_KEY || defaultValue
  }

  // L1 to L2 Tx Batch Processor Config
  public static transactionBatchSubmissionToAddress(defaultValue?: string) {
    return process.env.TRANSACTION_BATCH_SUBMISSION_TO_ADDRESS || defaultValue
  }
  public static transactionBatchSubmissionMethodId(defaultValue?: string) {
    return process.env.TRANSACTION_BATCH_SUBMISSION_METHOD_ID || defaultValue
  }

}
