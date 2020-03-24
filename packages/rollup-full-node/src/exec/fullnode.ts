/* External Imports */
import { BaseDB, DB, newInMemoryDB } from '@eth-optimism/core-db'
import { ExpressHttpServer, getLogger, Logger } from '@eth-optimism/core-utils'
import { L2ToL1MessageReceiverContractDefinition } from '@eth-optimism/ovm'

import Level from 'level'
import { JsonRpcProvider, Provider, Web3Provider } from 'ethers/providers'

/* Internal Imports */
import {
  FullnodeRpcServer,
  DefaultWeb3Handler,
  TestWeb3Handler,
  startLocalL1Node,
} from '../app'
import { L2ToL1MessageSubmitter } from '../types'
import { Contract } from 'ethers'
import { DefaultL2ToL1MessageSubmitter } from '../app/message-submitter'

const log: Logger = getLogger('rollup-fullnode')

const l1NodeLevelDBPath: string =
  process.env.L1_NODE_LEVELDB_PATH || '/leveldb/l1_fullnode'
const rollupNodeHost: string = process.env.ROLLUP_NODE_HOST || '0.0.0.0'
const rollupNodePort: string = process.env.ROLLUP_NODE_PORT || '8545'
const localL1NodePort: string = process.env.DEFAULT_L1_NODE_PORT || '7545'
const layer1Web3Url: string = process.env.L1_WEB3_URL
const layer2Web3Url: string = process.env.L2_WEB3_URL

/**
 * Runs a fullnode.
 * @param testFullnode Whether or not this is a test.
 * @returns The array of fullnode instance, L2ToL1MessageSubmitter
 */
export const runFullnode = async (
  testFullnode: boolean = false
): Promise<[ExpressHttpServer, L2ToL1MessageSubmitter]> => {
  let provider: JsonRpcProvider
  // TODO Get these from config
  const port = parseInt(rollupNodePort, 10)

  const messageSubmitter: L2ToL1MessageSubmitter = await runMessageSubmitter()

  log.info(`Starting L2 fullnode in ${testFullnode ? 'TEST' : 'LIVE'} mode`)

  if (layer2Web3Url) {
    log.info(`Connecting to L2 web3 URL: ${layer2Web3Url}`)
    provider = new JsonRpcProvider(layer2Web3Url)
  }

  const fullnodeHandler = testFullnode
    ? await TestWeb3Handler.create(messageSubmitter, provider)
    : await DefaultWeb3Handler.create(messageSubmitter, provider)
  const fullnodeRpcServer = new FullnodeRpcServer(
    fullnodeHandler,
    rollupNodeHost,
    port
  )

  fullnodeRpcServer.listen()

  const baseUrl = `http://${rollupNodeHost}:${port}`
  log.info(`Listening at ${baseUrl}`)

  return [fullnodeRpcServer, messageSubmitter]
}

const runMessageSubmitter = async (): Promise<L2ToL1MessageSubmitter> => {
  log.info(`Connecting to L1 fullnode.`)

  let db: DB
  let provider: JsonRpcProvider
  if (layer1Web3Url) {
    log.info(`Connecting to L1 web3 URL: ${layer1Web3Url}`)
    provider = new JsonRpcProvider(layer1Web3Url)

    db = new BaseDB(
      new Level(l1NodeLevelDBPath, {
        keyEncoding: 'binary',
        valueEncoding: 'binary',
      }),
      256
    )
  } else {
    log.info(`Deploying local L1 node on port ${localL1NodePort}`)
    provider = await startLocalL1Node(parseInt(localL1NodePort, 10))
    db = newInMemoryDB()
  }

  const messageReceiverContractAddress: string =
    process.env.L2_TO_L1_MESSAGE_RECEIVER_ADDRESS
  const messageReceiverContract = new Contract(
    messageReceiverContractAddress,
    L2ToL1MessageReceiverContractDefinition.abi,
    provider
  )

  return DefaultL2ToL1MessageSubmitter.create(db, messageReceiverContract)
}
