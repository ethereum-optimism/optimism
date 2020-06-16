/* External Imports */
import {
  BaseDB,
  DB,
  getLevelInstance,
  newInMemoryDB,
} from '@eth-optimism/core-db'
import {
  ExpressHttpServer,
  getLogger,
  logError,
  Logger,
  SimpleClient,
} from '@eth-optimism/core-utils'
import {
  Environment,
  initializeL1Node,
  initializeL2Node,
  L1NodeContext,
  L2NodeContext,
} from '@eth-optimism/rollup-core'
import cors = require('cors')

import { JsonRpcProvider } from 'ethers/providers'
import * as fs from 'fs'
import * as rimraf from 'rimraf'

/* Internal Imports */
import {
  FullnodeRpcServer,
  DefaultWeb3Handler,
  TestWeb3Handler,
  RoutingHandler,
  DefaultL2ToL1MessageSubmitter,
  NoOpL2ToL1MessageSubmitter,
  NoOpAccountRateLimiter,
  DefaultAccountRateLimiter,
} from '../app'
import {
  AccountRateLimiter,
  FullnodeHandler,
  L2ToL1MessageSubmitter,
  Web3Handler,
} from '../types'

const log: Logger = getLogger('rollup-fullnode')

export interface FullnodeContext {
  fullnodeHandler: FullnodeHandler & Web3Handler
  fullnodeRpcServer: ExpressHttpServer
  l2ToL1MessageSubmitter: L2ToL1MessageSubmitter
  l1NodeContext: L1NodeContext
}

/**
 * Runs the configured server.
 * This will either start a
 * * Router - handles rate limiting and distribute load between read-only and transaction processing nodes
 * * Transaction Node - a full node that will be sent transactions and requests tightly-coupled with transactions.
 * * Read-only Node - a full node that will only be sent requests that read state but don't modify it.
 *
 * @param testFullnode Whether or not this is a test.
 * @returns The array of fullnode instance, L2ToL1MessageSubmitter
 */
export const runFullnode = async (
  testFullnode: boolean = false
): Promise<FullnodeContext> => {
  if (
    !!Environment.isTranasactionNode() ||
    (!Environment.isRoutingServer() && !Environment.isReadOnlyNode())
  ) {
    log.info(`Starting Transaction Node`)
    return startTransactionNode(testFullnode)
  }
  if (Environment.isRoutingServer()) {
    log.info(`Starting Routing Server`)
    return startRoutingServer()
  }

  log.info(`Starting Read-only Node`)
  return startReadOnlyNode(testFullnode)
}

/**
 * Starts a routing server that handles rate limiting and routes
 * requests to the configured transaction node and readonly node.
 *
 * @returns The L2NodeContext with undefined values for everything except for handler and server.
 */
const startRoutingServer = async (): Promise<FullnodeContext> => {
  if (
    (!!Environment.maxNonTransactionRequestsPerUnitTime() ||
      !!Environment.maxTransactionsPerUnitTime() ||
      !!Environment.requestLimitPeriodMillis()) &&
    !(
      !!Environment.maxNonTransactionRequestsPerUnitTime() &&
      !!Environment.maxTransactionsPerUnitTime() &&
      !!Environment.requestLimitPeriodMillis()
    )
  ) {
    throw new Error(
      'Routing server rate limiting is partially configured. Please configure all of MAX_NON_TRANSACTION_REQUESTS_PER_UNIT_TIME, MAX_TRANSACTIONS_PER_UNIT_TIME, REQUEST_LIMIT_PERIOD_MILLIS or none of them.'
    )
  }

  const rateLimiter: AccountRateLimiter = !Environment.maxTransactionsPerUnitTime()
    ? new NoOpAccountRateLimiter()
    : new DefaultAccountRateLimiter(
        Environment.maxNonTransactionRequestsPerUnitTime(),
        Environment.maxTransactionsPerUnitTime(),
        Environment.requestLimitPeriodMillis()
      )

  const fullnodeHandler = new RoutingHandler(
    new SimpleClient(Environment.getOrThrow(Environment.transactionNodeUrl)),
    new SimpleClient(Environment.getOrThrow(Environment.readOnlyNodeUrl)),
    Environment.contractDeployerAddress(),
    rateLimiter,
    Environment.rateLimitWhitelistIpAddresses(),
    Environment.transactionToAddressWhitelist()
  )
  const fullnodeRpcServer = new FullnodeRpcServer(
    fullnodeHandler,
    Environment.getOrThrow(Environment.l2RpcServerHost),
    Environment.getOrThrow(Environment.l2RpcServerPort),
    [cors]
  )

  fullnodeRpcServer.listen()

  const baseUrl = `http://${Environment.l2RpcServerHost()}:${Environment.l2RpcServerPort()}`
  log.info(`Listening at ${baseUrl}`)

  setInterval(() => {
    updateEnvironmentVariables('/server/env_var_updates.config')
  }, 179_000)

  return {
    fullnodeHandler: undefined,
    fullnodeRpcServer,
    l2ToL1MessageSubmitter: undefined,
    l1NodeContext: undefined,
  }
}

/**
 * Starts a transaction node, which includes
 *
 * @param testFullnode Whether or not this node is in test mode (allowing test RPC methods).
 */
const startTransactionNode = async (
  testFullnode: boolean
): Promise<FullnodeContext> => {
  initializeDBPaths(testFullnode)

  let provider: JsonRpcProvider

  let l1NodeContext: L1NodeContext
  let l2ToL1MessageSubmitter: L2ToL1MessageSubmitter
  if (!!Environment.noL1Node()) {
    log.info(`Not connecting to L1 node per configuration.`)
    l2ToL1MessageSubmitter = new NoOpL2ToL1MessageSubmitter()
  } else {
    log.info(`Connecting to L1 fullnode.`)
    l1NodeContext = await initializeL1Node()
    l2ToL1MessageSubmitter = await DefaultL2ToL1MessageSubmitter.create(
      l1NodeContext.sequencerWallet,
      l1NodeContext.l2ToL1MessageReceiver
    )
  }

  log.info(
    `Starting L2 TRANSACTION PROCESSING SERVER in ${
      testFullnode ? 'TEST' : 'LIVE'
    } mode`
  )

  if (!!Environment.l2NodeWeb3Url()) {
    log.info(`Connecting to L2 web3 URL: ${Environment.l2NodeWeb3Url()}`)
    provider = new JsonRpcProvider(Environment.l2NodeWeb3Url())
  }

  const fullnodeHandler = testFullnode
    ? await TestWeb3Handler.create(l2ToL1MessageSubmitter, provider)
    : await DefaultWeb3Handler.create(l2ToL1MessageSubmitter, provider)
  const fullnodeRpcServer = new FullnodeRpcServer(
    fullnodeHandler,
    Environment.l2RpcServerHost(),
    Environment.l2RpcServerPort(),
    [cors]
  )

  fullnodeRpcServer.listen()

  const baseUrl = `http://${Environment.l2RpcServerHost()}:${Environment.l2RpcServerPort()}`
  log.info(`Listening at ${baseUrl}`)

  return {
    fullnodeHandler,
    fullnodeRpcServer,
    l2ToL1MessageSubmitter,
    l1NodeContext,
  }
}

/**
 * Starts a read-only node. This will only handle reading from the wrapped node and will
 * not deploy any contracts on L1 or L2 or process transactions.
 *
 * @param testFullnode Whether or not this is a test full node, exposing test RPC methods.
 * @returns The test FullnodeContext with undefined values for everything except for handler and server.
 */
const startReadOnlyNode = async (
  testFullnode: boolean
): Promise<FullnodeContext> => {
  log.info(
    `Starting L2 READ ONLY SERVER in ${testFullnode ? 'TEST' : 'LIVE'} mode`
  )

  let provider: JsonRpcProvider
  if (Environment.l2NodeWeb3Url()) {
    log.info(`Connecting to L2 web3 URL: ${Environment.l2NodeWeb3Url()}`)
    provider = new JsonRpcProvider(Environment.l2NodeWeb3Url())
  }

  const l2NodeContext: L2NodeContext = await initializeL2Node(provider, true)

  const noOpMessageSubmitter: L2ToL1MessageSubmitter = new NoOpL2ToL1MessageSubmitter()
  const fullnodeHandler = testFullnode
    ? await TestWeb3Handler.create(
        noOpMessageSubmitter,
        provider,
        l2NodeContext
      )
    : await DefaultWeb3Handler.create(
        noOpMessageSubmitter,
        provider,
        l2NodeContext
      )
  const fullnodeRpcServer = new FullnodeRpcServer(
    fullnodeHandler,
    Environment.l2RpcServerHost(),
    Environment.l2RpcServerPort(),
    [cors]
  )

  fullnodeRpcServer.listen()

  const baseUrl = `http://${Environment.l2RpcServerHost()}:${Environment.l2RpcServerPort()}`
  log.info(`Listening at ${baseUrl}`)

  return {
    fullnodeHandler,
    fullnodeRpcServer,
    l2ToL1MessageSubmitter: undefined,
    l1NodeContext: undefined,
  }
}

/**
 * Initializes filesystem DB paths. This will also purge all data if the `CLEAR_DATA_KEY` has changed.
 */
const initializeDBPaths = (isTestMode: boolean) => {
  if (isTestMode) {
    return
  }

  if (!fs.existsSync(Environment.l2RpcServerPersistentDbPath())) {
    makeDataDirectory()
  } else {
    if (Environment.clearDataKey() && !fs.existsSync(getClearDataFilePath())) {
      log.info(`Detected change in CLEAR_DATA_KEY. Purging data...`)
      rimraf.sync(`${Environment.l2RpcServerPersistentDbPath()}/{*,.*}`)
      log.info(
        `L2 RPC Server data purged from '${Environment.l2RpcServerPersistentDbPath()}/{*,.*}'`
      )
      if (Environment.localL1NodePersistentDbPath()) {
        rimraf.sync(`${Environment.localL1NodePersistentDbPath()}/{*,.*}`)
        log.info(
          `Local L1 node data purged from '${Environment.localL1NodePersistentDbPath()}/{*,.*}'`
        )
      }
      if (Environment.localL2NodePersistentDbPath()) {
        rimraf.sync(`${Environment.localL2NodePersistentDbPath()}/{*,.*}`)
        log.info(
          `Local L2 node data purged from '${Environment.localL2NodePersistentDbPath()}/{*,.*}'`
        )
      }
      makeDataDirectory()
    }
  }
}

/**
 * Updates process environment variables from provided update file
 * if any variables are updated.
 *
 * @param updateFilePath The path to the file from which to read env var updates.
 */
const updateEnvironmentVariables = (updateFilePath: string) => {
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

/**
 * Gets the appropriate db for this node to use based on whether or not this is run in test mode.
 *
 * @param isTestMode Whether or not it is test mode.
 * @returns The constructed DB instance.
 */
const getDB = (isTestMode: boolean = false): DB => {
  if (isTestMode) {
    return newInMemoryDB()
  } else {
    if (!Environment.l2RpcServerPersistentDbPath()) {
      log.error(
        `No L2_RPC_SERVER_PERSISTENT_DB_PATH environment variable present. Please set one!`
      )
      process.exit(1)
    }

    return new BaseDB(
      getLevelInstance(Environment.l2RpcServerPersistentDbPath())
    )
  }
}

/**
 * Makes the data directory for this full node and adds a clear data key file if it is configured to use one.
 */
const makeDataDirectory = () => {
  fs.mkdirSync(Environment.l2RpcServerPersistentDbPath(), { recursive: true })
  if (Environment.clearDataKey()) {
    fs.writeFileSync(getClearDataFilePath(), '')
  }
}

const getClearDataFilePath = () => {
  return `${Environment.l2RpcServerPersistentDbPath()}/.clear_data_key_${Environment.clearDataKey()}`
}
