/* External Imports */
import {
  BaseDB,
  DB,
  getLevelInstance,
  newInMemoryDB,
} from '@eth-optimism/core-db'
import { ExpressHttpServer, getLogger, Logger } from '@eth-optimism/core-utils'
import { L1ToL2TransactionProcessor } from '@eth-optimism/rollup-core'
import { JsonRpcProvider } from 'ethers/providers'

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
}

/**
 * Runs a fullnode.
 * @param testFullnode Whether or not this is a test.
 * @returns The array of fullnode instance, L2ToL1MessageSubmitter
 */
export const runFullnode = async (
  testFullnode: boolean = false
): Promise<FullnodeContext> => {
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

  const db: DB = testFullnode
    ? newInMemoryDB()
    : new BaseDB(getLevelInstance(Environment.l2RpcServerPersistentDbPath()))

  const l1ToL2TransactionProcessor: L1ToL2TransactionProcessor = await L1ToL2TransactionProcessor.create(
    db,
    [fullnodeHandler]
  )

  fullnodeRpcServer.listen()

  const baseUrl = `http://${Environment.l2RpcServerHost()}:${port}`
  log.info(`Listening at ${baseUrl}`)

  return {
    fullnodeHandler,
    fullnodeRpcServer,
    l2ToL1MessageSubmitter,
    l1ToL2TransactionProcessor,
  }
}
