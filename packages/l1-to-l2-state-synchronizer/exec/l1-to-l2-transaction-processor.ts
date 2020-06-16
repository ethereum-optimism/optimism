/* External Imports */
import {
  BaseDB,
  DB,
  EthereumEventProcessor,
  getLevelInstance,
  newInMemoryDB,
} from '@eth-optimism/core-db'
import { add0x, getLogger, logError } from '@eth-optimism/core-utils'
import {
  Environment,
  initializeL1Node,
  L1NodeContext,
  L1ToL2TransactionProcessor,
  L1ToL2TransactionEventName,
  L1ToL2TransactionListener,
  L1ToL2TransactionListenerSubmitter,
  CHAIN_ID,
} from '@eth-optimism/rollup-core'

import { JsonRpcProvider, Provider } from 'ethers/providers'
import * as fs from 'fs'
import * as rimraf from 'rimraf'
import { Wallet } from 'ethers'
import { getWallets } from 'ethereum-waffle'

const log = getLogger('l1-to-l2-transaction-processor')

export const runTest = async (
  l1Provider?: Provider,
  l2Provider?: JsonRpcProvider
): Promise<L1ToL2TransactionProcessor> => {
  return run(true, l1Provider, l2Provider)
}

export const run = async (
  testFullNode: boolean = false,
  l1Provider?: Provider,
  l2Provider?: JsonRpcProvider
): Promise<L1ToL2TransactionProcessor> => {
  initializeDBPaths(testFullNode)

  let l1NodeContext: L1NodeContext
  log.info(`Attempting to connect to L1 Node.`)
  try {
    l1NodeContext = await initializeL1Node(true, l1Provider)
  } catch (e) {
    logError(log, 'Error connecting to L1 Node', e)
    throw e
  }

  let provider: JsonRpcProvider = l2Provider
  if (!provider && !!Environment.l2NodeWeb3Url()) {
    log.info(`Connecting to L2 web3 URL: ${Environment.l2NodeWeb3Url()}`)
    provider = new JsonRpcProvider(Environment.l2NodeWeb3Url(), CHAIN_ID)
  }

  const l2TransactionListenerSubmitter = new L1ToL2TransactionListenerSubmitter(
    getWallet(provider),
    provider
  )

  return getL1ToL2TransactionProcessor(
    testFullNode,
    l1NodeContext,
    l2TransactionListenerSubmitter
  )
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
    if (!Environment.stateSynchronizerPersistentDbPath()) {
      log.error(
        `No L1_TO_L2_TX_PROCESSOR_PERSISTENT_DB_PATH environment variable present. Please set one!`
      )
      process.exit(1)
    }

    return new BaseDB(
      getLevelInstance(Environment.stateSynchronizerPersistentDbPath())
    )
  }
}

/**
 * Gets the wallet to use to interact with the L2 node. This may be configured via
 * private key file specified through environment variables. If not it is assumed
 * that a local test provider is being used, from which the wallet may be fetched.
 *
 * @param provider The provider with which the wallet will be associated.
 * @returns The wallet to use with the L2 node.
 */
const getWallet = (provider: JsonRpcProvider): Wallet => {
  let wallet: Wallet
  if (!!Environment.stateSynchronizerPrivateKey()) {
    wallet = new Wallet(
      add0x(Environment.stateSynchronizerPrivateKey()),
      provider
    )
    log.info(
      `Initialized L1-to-L2 Tx processor wallet from private key. Address: ${wallet.address}`
    )
  } else {
    wallet = getWallets(provider)[0]
    log.info(
      `Getting wallet from provider. First wallet private key: [${wallet.privateKey}`
    )
  }

  if (!wallet) {
    const msg: string = `Wallet not created! Specify the L1_TO_L2_TX_PROCESSOR_PRIVATE_KEY environment variable to set one!`
    log.error(msg)
    throw Error(msg)
  } else {
    log.info(`L1-to-L2 Tx processor wallet created. Address: ${wallet.address}`)
  }

  return wallet
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
      rimraf.sync(`${Environment.stateSynchronizerPersistentDbPath()}/{*,.*}`)
      log.info(
        `L2 RPC Server data purged from '${Environment.stateSynchronizerPersistentDbPath()}/{*,.*}'`
      )
      makeDataDirectory()
    }
  }
}

/**
 * Makes the data directory for this full node and adds a clear data key file if it is configured to use one.
 */
const makeDataDirectory = () => {
  fs.mkdirSync(Environment.stateSynchronizerPersistentDbPath(), {
    recursive: true,
  })
  if (Environment.clearDataKey()) {
    fs.writeFileSync(getClearDataFilePath(), '')
  }
}

const getClearDataFilePath = () => {
  return `${Environment.stateSynchronizerPersistentDbPath()}/.clear_data_key_${Environment.clearDataKey()}`
}

if (typeof require !== 'undefined' && require.main === module) {
  run()
}
