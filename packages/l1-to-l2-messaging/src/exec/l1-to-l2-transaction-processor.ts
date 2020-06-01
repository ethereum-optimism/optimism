import {
  BaseDB,
  DB,
  EthereumEventProcessor,
  getLevelInstance,
  newInMemoryDB,
} from '@eth-optimism/core-db'
import { getLogger } from '@eth-optimism/core-utils'
import {
  Environment,
  L1NodeContext,
  L1ToL2TransactionEventName,
  L1ToL2TransactionListener,
  L1ToL2TransactionProcessor,
} from '@eth-optimism/rollup-core'

const log = getLogger('l1-to-l2-transaction-processor')

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
