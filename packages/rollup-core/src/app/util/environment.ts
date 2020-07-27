import { getLogger, logError } from '@eth-optimism/core-utils'
import {
  DEFAULT_OPCODE_WHITELIST_MASK,
  L2_TO_L1_MESSAGE_PASSER_OVM_ADDRESS,
} from '../constants'
import * as fs from 'fs'

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
    if (res === undefined || (typeof res === 'number' && isNaN(res))) {
      throw Error(
        `Expected Environment variable not set. Error calling Environment.${fun.name}()`
      )
    }
    const lowerName: string = fun.name.toLowerCase()
    if (
      logValue &&
      lowerName.indexOf('password') < 0 &&
      lowerName.indexOf('private') < 0
    ) {
      log.info(`Environment: ${fun.name} = ${res}`)
    }
    return res
  }

  public static clearDataKey(defaultValue?: string) {
    return process.env.CLEAR_DATA_KEY || defaultValue
  }

  public static isSequencerStack(defaultValue?: boolean): boolean {
    return !!process.env.IS_SEQUENCER_STACK || defaultValue
  }

  // Microservices to run config
  public static runL1ChainDataPersister(
    defaultValue: boolean = false
  ): boolean {
    return !!process.env.RUN_L1_CHAIN_DATA_PERSISTER || defaultValue
  }
  public static runL2ChainDataPersister(
    defaultValue: boolean = false
  ): boolean {
    return !!process.env.RUN_L2_CHAIN_DATA_PERSISTER || defaultValue
  }
  public static runGethSubmissionQueuer(
    defaultValue: boolean = false
  ): boolean {
    return !!process.env.RUN_GETH_SUBMISSION_QUEUER || defaultValue
  }
  public static runQueuedGethSubmitter(defaultValue: boolean = false): boolean {
    return !!process.env.RUN_QUEUED_GETH_SUBMITTER || defaultValue
  }
  public static runCanonicalChainBatchCreator(
    defaultValue: boolean = false
  ): boolean {
    return !!process.env.RUN_CANONICAL_CHAIN_BATCH_CREATOR || defaultValue
  }
  public static runCanonicalChainBatchSubmitter(
    defaultValue: boolean = false
  ): boolean {
    return !!process.env.RUN_CANONICAL_CHAIN_BATCH_SUBMITTER || defaultValue
  }
  public static runStateCommitmentChainBatchCreator(
    defaultValue: boolean = false
  ): boolean {
    return (
      !!process.env.RUN_STATE_COMMITMENT_CHAIN_BATCH_CREATOR || defaultValue
    )
  }
  public static runStateCommitmentChainBatchSubmitter(
    defaultValue: boolean = false
  ): boolean {
    return (
      !!process.env.RUN_STATE_COMMITMENT_CHAIN_BATCH_SUBMITTER || defaultValue
    )
  }
  public static runFraudDetector(defaultValue: boolean = false): boolean {
    return !!process.env.RUN_FRAUD_DETECTOR || defaultValue
  }

  // L1 Contract Addresses
  public static canonicalTransactionChainContractAddress(
    defaultValue?: string
  ) {
    return (
      process.env.CANONICAL_TRANSACTION_CHAIN_CONTRACT_ADDRESS || defaultValue
    )
  }
  public static stateCommitmentChainContractAddress(defaultValue?: string) {
    return process.env.STATE_COMMITMENT_CHAIN_CONTRACT_ADDRESS || defaultValue
  }
  public static l1ToL2TransactionQueueContractAddress(defaultValue?: string) {
    return (
      process.env.L1_TO_L2_TRANSACTION_QUEUE_CONTRACT_ADDRESS || defaultValue
    )
  }
  public static safetyTransactionQueueContractAddress(defaultValue?: string) {
    return process.env.SAFETY_TRANSACTION_QUEUE_CONTRACT_ADDRESS || defaultValue
  }
  public static l1ToL2TransactionPasserContractAddress(
    defaultValue?: string
  ): string {
    return (
      process.env.L1_TO_L2_TRANSACTION_PASSER_CONTRACT_ADDRESS || defaultValue
    )
  }
  public static l2ToL1MessageReceiverContractAddress(
    defaultValue?: string
  ): string {
    return (
      process.env.L2_TO_L1_MESSAGE_RECEIVER_CONTRACT_ADDRESS || defaultValue
    )
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
  public static finalityDelayInBlocks(defaultValue?: number): number {
    return process.env.FINALITY_DELAY_IN_BLOCKS
      ? parseInt(process.env.FINALITY_DELAY_IN_BLOCKS, 10)
      : defaultValue
  }
  public static l1EarliestBlock(defaultValue: number = 0): number {
    return process.env.L1_EARLIEST_BLOCK
      ? parseInt(process.env.L1_EARLIEST_BLOCK, 10)
      : defaultValue
  }

  // Batch Sizes
  public static canonicalChainMinBatchSize(defaultValue?: number): number {
    return process.env.CANONICAL_CHAIN_MIN_BATCH_SIZE
      ? parseInt(process.env.CANONICAL_CHAIN_MIN_BATCH_SIZE, 10)
      : defaultValue
  }

  public static canonicalChainMaxBatchSize(defaultValue?: number): number {
    return process.env.CANONICAL_CHAIN_MAX_BATCH_SIZE
      ? parseInt(process.env.CANONICAL_CHAIN_MAX_BATCH_SIZE, 10)
      : defaultValue
  }

  public static stateCommitmentChainMinBatchSize(
    defaultValue?: number
  ): number {
    return process.env.STATE_COMMITMENT_CHAIN_MIN_BATCH_SIZE
      ? parseInt(process.env.STATE_COMMITMENT_CHAIN_MIN_BATCH_SIZE, 10)
      : defaultValue
  }

  public static stateCommitmentChainMaxBatchSize(
    defaultValue?: number
  ): number {
    return process.env.STATE_COMMITMENT_CHAIN_MAX_BATCH_SIZE
      ? parseInt(process.env.STATE_COMMITMENT_CHAIN_MAX_BATCH_SIZE, 10)
      : defaultValue
  }

  // Poller periods
  public static canonicalChainBatchCreatorPeriodMillis(
    defaultValue: number = 10_000
  ): number {
    return process.env.CANONICAL_CHAIN_BATCH_CREATOR_PERIOD_MILLIS
      ? parseInt(process.env.CANONICAL_CHAIN_BATCH_CREATOR_PERIOD_MILLIS, 10)
      : defaultValue
  }
  public static canonicalChainBatchSubmitterPeriodMillis(
    defaultValue: number = 10_000
  ): number {
    return process.env.CANONICAL_CHAIN_BATCH_SUBMITTER_PERIOD_MILLIS
      ? parseInt(process.env.CANONICAL_CHAIN_BATCH_SUBMITTER_PERIOD_MILLIS, 10)
      : defaultValue
  }
  public static stateCommitmentChainBatchCreatorPeriodMillis(
    defaultValue: number = 10_000
  ): number {
    return process.env.STATE_COMMITMENT_CHAIN_BATCH_CREATOR_PERIOD_MILLIS
      ? parseInt(
          process.env.STATE_COMMITMENT_CHAIN_BATCH_CREATOR_PERIOD_MILLIS,
          10
        )
      : defaultValue
  }
  public static stateCommitmentChainBatchSubmitterPeriodMillis(
    defaultValue: number = 10_000
  ): number {
    return process.env.STATE_COMMITMENT_CHAIN_BATCH_SUBMITTER_PERIOD_MILLIS
      ? parseInt(
          process.env.STATE_COMMITMENT_CHAIN_BATCH_SUBMITTER_PERIOD_MILLIS,
          10
        )
      : defaultValue
  }
  public static gethSubmissionQueuerPeriodMillis(
    defaultValue: number = 10_000
  ): number {
    return process.env.GETH_SUBMISSION_QUEUER_PERIOD_MILLIS
      ? parseInt(process.env.GETH_SUBMISSION_QUEUER_PERIOD_MILLIS, 10)
      : defaultValue
  }
  public static queuedGethSubmitterPeriodMillis(
    defaultValue: number = 10_000
  ): number {
    return process.env.QUEUED_GETH_SUBMITTER_PERIOD_MILLIS
      ? parseInt(process.env.QUEUED_GETH_SUBMITTER_PERIOD_MILLIS, 10)
      : defaultValue
  }
  public static fraudDetectorPeriodMillis(
    defaultValue: number = 10_000
  ): number {
    return process.env.FRAUD_DETECTOR_PERIOD_MILLIS
      ? parseInt(process.env.FRAUD_DETECTOR_PERIOD_MILLIS, 10)
      : defaultValue
  }

  // Chain Data Persisters
  public static l1ChainDataPersisterLevelDbPath(defaultValue?: string): string {
    return process.env.L1_CHAIN_DATA_PERSISTER_DB_PATH || defaultValue
  }
  public static l2ChainDataPersisterLevelDbPath(defaultValue?: string): string {
    return process.env.L2_CHAIN_DATA_PERSISTER_DB_PATH || defaultValue
  }

  // Postgres Database config
  public static postgresHost(defaultValue?: string): string {
    return process.env.POSTGRES_HOST || defaultValue
  }
  public static postgresPort(defaultValue: number = 5432): number {
    return process.env.POSTGRES_PORT
      ? parseInt(process.env.POSTGRES_PORT, 10)
      : defaultValue
  }
  public static postgresUser(defaultValue?: string): string {
    return process.env.POSTGRES_USER || defaultValue
  }
  public static postgresPassword(defaultValue?: string): string {
    return process.env.POSTGRES_PASSWORD || defaultValue
  }
  public static postgresDatabase(defaultValue?: string): string {
    return process.env.POSTGRES_DATABASE || defaultValue
  }
  public static postgresPoolSize(defaultValue?: number): number {
    return process.env.POSTGRES_CONNECTION_POOL_SIZE
      ? parseInt(process.env.POSTGRES_CONNECTION_POOL_SIZE, 10)
      : defaultValue
  }
  public static postgresUseSsl(defaultValue?: boolean): boolean {
    return !!process.env.POSTGRES_USE_SSL || defaultValue
  }

  // Misc
  public static submitToL2GethPrivateKey(defaultValue?: string): string {
    return process.env.SUBMIT_TO_L2_PRIVATE_KEY || defaultValue
  }
  public static reAlertOnUnresolvedFraudEveryNFraudDetectorRuns(
    defaultValue: number = 10
  ): number {
    return process.env.REALERT_ON_UNRESOLVED_FRAUD_EVERY_N_FRAUD_DETECTOR_RUNS
      ? parseInt(
          process.env.REALERT_ON_UNRESOLVED_FRAUD_EVERY_N_FRAUD_DETECTOR_RUNS,
          10
        )
      : defaultValue
  }

  public static environmentVariablesUpdateFilePath(
    defaultValue: string = '/server/env_var_updates.config'
  ): string {
    return process.env.ENVIRONMENT_VARIABLES_UPDATE_FILE_PATH || defaultValue
  }
}

/**
 * Updates process environment variables from provided update file
 * if any variables are updated.
 *
 * @param updateFilePath The path to the file from which to read env var updates.
 */
export const updateEnvironmentVariables = (
  updateFilePath: string = Environment.environmentVariablesUpdateFilePath()
) => {
  try {
    fs.readFile(updateFilePath, 'utf8', (error, data) => {
      try {
        let changesExist: boolean = false
        if (!!error) {
          logError(
            log,
            `Error reading environment variable updates from ${updateFilePath}`,
            error
          )
          return
        }

        const lines = data.split('\n')
        for (const rawLine of lines) {
          if (!rawLine) {
            continue
          }
          const line = rawLine.trim()
          if (!line || line.startsWith('#')) {
            continue
          }

          const varAssignmentSplit = line.split('=')
          if (varAssignmentSplit.length !== 2) {
            log.error(
              `Invalid updated env variable line: ${line}. Expected some_var_name=somevalue`
            )
            continue
          }
          const deletePlaceholder = '$DELETE$'
          const key = varAssignmentSplit[0].trim()
          const value = varAssignmentSplit[1].trim()
          if (value === deletePlaceholder && !!process.env[key]) {
            delete process.env[key]
            log.info(`Updated process.env.${key} to have no value.`)
            changesExist = true
          } else if (
            value !== process.env[key] &&
            value !== deletePlaceholder
          ) {
            process.env[key] = value
            log.info(`Updated process.env.${key} to have value ${value}.`)
            changesExist = true
          }
        }
      } catch (e) {
        logError(
          log,
          `Error updating environment variables from ${updateFilePath}`,
          e
        )
      }
    })
  } catch (e) {
    logError(
      log,
      `Error updating environment variables from ${updateFilePath}`,
      e
    )
  }
}
