/* External Imports */
import {
  BaseDB,
  DB,
  EthereumEventProcessor,
  getLevelInstance,
  newInMemoryDB,
} from '@eth-optimism/core-db'
import { ExpressHttpServer, getLogger, Logger } from '@eth-optimism/core-utils'
import {
  L1ToL2TransactionEventName,
  L1ToL2TransactionListener,
  L1ToL2TransactionProcessor,
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
  Environment,
  initializeL1Node,
  RoutingHandler,
  DefaultL2ToL1MessageSubmitter,
  NoOpL2ToL1MessageSubmitter,
  initializeL2Node,
} from '../app'
import {
  FullnodeHandler,
  L1NodeContext,
  L2NodeContext,
  L2ToL1MessageSubmitter,
  Web3Handler,
} from '../types'

const log: Logger = getLogger('rollup-fullnode')

export interface FullnodeContext {
  fullnodeHandler: FullnodeHandler & Web3Handler
  fullnodeRpcServer: ExpressHttpServer
  l2ToL1MessageSubmitter: L2ToL1MessageSubmitter
  l1ToL2TransactionProcessor: L1ToL2TransactionProcessor
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
  const fullnodeHandler = new RoutingHandler(
    Environment.transactionNodeUrl(),
    Environment.readOnlyNodeUrl(),
    Environment.maxNonTransactionRequestsPerUnitTime(),
    Environment.maxTransactionsPerUnitTime(),
    Environment.requestLimitPeriodMillis(),
    Environment.contractDeployerAddress(),
    Environment.commaSeparatedToAddressWhitelist('').split(',')
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
    fullnodeHandler: undefined,
    fullnodeRpcServer,
    l2ToL1MessageSubmitter: undefined,
    l1ToL2TransactionProcessor: undefined,
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

  const l1ToL2TransactionProcessor: L1ToL2TransactionProcessor = await getL1ToL2TransactionProcessor(
    testFullnode,
    l1NodeContext,
    fullnodeHandler
  )

  fullnodeRpcServer.listen()

  const baseUrl = `http://${Environment.l2RpcServerHost()}:${Environment.l2RpcServerPort()}`
  log.info(`Listening at ${baseUrl}`)

  return {
    fullnodeHandler,
    fullnodeRpcServer,
    l2ToL1MessageSubmitter,
    l1ToL2TransactionProcessor,
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
    l1ToL2TransactionProcessor: undefined,
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
 * Gets an L1ToL2TransactionProcessor based on configuration and the provided arguments.
 *
 * Notably this will return undefined if configuration says not to connect to the L1 node.
 *
 * @param testFullnode Whether or not this is a test full node.
 * @param l1NodeContext The L1 node context.
 * @param listener The listener to listen to the processor.
 * @returns The L1ToL2TransactionProcessor or undefined.
 */
const getL1ToL2TransactionProcessor = async (
  testFullnode: boolean,
  l1NodeContext: L1NodeContext,
  listener: L1ToL2TransactionListener
): Promise<L1ToL2TransactionProcessor> => {
  if (Environment.noL1Node()) {
    return undefined
  }

  const db: DB = getDB(testFullnode)
  const l1ToL2TransactionProcessor: L1ToL2TransactionProcessor = await L1ToL2TransactionProcessor.create(
    db,
    EthereumEventProcessor.getEventID(
      l1NodeContext.l1ToL2TransactionPasser.address,
      L1ToL2TransactionEventName
    ),
    [listener]
  )

  const earliestBlock = Environment.l1EarliestBlock()

  const eventProcessor = new EthereumEventProcessor(db, earliestBlock)
  await eventProcessor.subscribe(
    l1NodeContext.l1ToL2TransactionPasser,
    L1ToL2TransactionEventName,
    l1ToL2TransactionProcessor
  )

  return l1ToL2TransactionProcessor
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
