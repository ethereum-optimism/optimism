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
  L1ToL2TransactionProcessor,
} from '@eth-optimism/rollup-core'

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
} from '../app'
import {
  FullnodeHandler,
  L1NodeContext,
  L2ToL1MessageSubmitter,
  Web3Handler,
} from '../types'
import { DefaultL2ToL1MessageSubmitter } from '../app/message-submitter'

const log: Logger = getLogger('rollup-fullnode')

export interface FullnodeContext {
  fullnodeHandler: FullnodeHandler & Web3Handler
  fullnodeRpcServer: ExpressHttpServer
  l2ToL1MessageSubmitter: L2ToL1MessageSubmitter
  l1ToL2TransactionProcessor: L1ToL2TransactionProcessor
  l1NodeContext: L1NodeContext
}

/**
 * Runs a fullnode.
 * @param testFullnode Whether or not this is a test.
 * @returns The array of fullnode instance, L2ToL1MessageSubmitter
 */
export const runFullnode = async (
  testFullnode: boolean = false
): Promise<FullnodeContext> => {
  initializeDBPaths()

  let provider: JsonRpcProvider
  // TODO Get these from config
  const port: number = Environment.l2RpcServerPort()

  log.info(`Connecting to L1 fullnode.`)
  const l1NodeContext: L1NodeContext = await initializeL1Node()

  const l2ToL1MessageSubmitter: L2ToL1MessageSubmitter = await DefaultL2ToL1MessageSubmitter.create(
    l1NodeContext.sequencerWallet,
    l1NodeContext.l2ToL1MessageReceiver
  )

  log.info(`Starting L2 fullnode in ${testFullnode ? 'TEST' : 'LIVE'} mode`)

  if (Environment.l2NodeWeb3Url()) {
    log.info(`Connecting to L2 web3 URL: ${Environment.l2NodeWeb3Url()}`)
    provider = new JsonRpcProvider(Environment.l2NodeWeb3Url())
  }

  const fullnodeHandler = testFullnode
    ? await TestWeb3Handler.create(l2ToL1MessageSubmitter, provider)
    : await DefaultWeb3Handler.create(l2ToL1MessageSubmitter, provider)
  const fullnodeRpcServer = new FullnodeRpcServer(
    fullnodeHandler,
    Environment.l2RpcServerHost(),
    port
  )

  const db: DB = getDB(testFullnode)

  const l1ToL2TransactionProcessor: L1ToL2TransactionProcessor = await L1ToL2TransactionProcessor.create(
    db,
    EthereumEventProcessor.getEventID(
      l1NodeContext.l1ToL2TransactionPasser.address,
      L1ToL2TransactionEventName
    ),
    [fullnodeHandler]
  )

  // TODO: Figure out earliest block # when necessary
  const eventProcessor = new EthereumEventProcessor(db)
  await eventProcessor.subscribe(
    l1NodeContext.l1ToL2TransactionPasser,
    L1ToL2TransactionEventName,
    l1ToL2TransactionProcessor
  )

  fullnodeRpcServer.listen()

  const baseUrl = `http://${Environment.l2RpcServerHost()}:${port}`
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
 * Initializes filesystem DB paths. This will also purge all data if the `CLEAR_DATA_KEY` has changed.
 */
const initializeDBPaths = () => {
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
