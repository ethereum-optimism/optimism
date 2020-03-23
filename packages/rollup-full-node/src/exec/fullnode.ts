/* External Imports */
import { ExpressHttpServer, getLogger, Logger } from '@eth-optimism/core-utils'
import { JsonRpcProvider } from 'ethers/providers'

/* Internal Imports */
import {
  FullnodeRpcServer,
  DefaultWeb3Handler,
  TestWeb3Handler,
} from '../app'
import {L2ToL1MessageSubmitter} from '../types'

const log: Logger = getLogger('rollup-fullnode')

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
  const host = '0.0.0.0'
  const port = 8545

  log.info(`Starting L2 fullnode in ${testFullnode ? 'TEST' : 'LIVE'} mode`)

  if (process.env.L2_WEB3_URL) {
    log.info(`Connecting to L2 web3 URL: ${process.env.L2_WEB3_URL}`)
    provider = new JsonRpcProvider(process.env.L2_WEB3_URL)
  }

  const fullnodeHandler = testFullnode
    ? await TestWeb3Handler.create(provider)
    : await DefaultWeb3Handler.create(provider)
  const fullnodeRpcServer = new FullnodeRpcServer(fullnodeHandler, host, port)

  fullnodeRpcServer.listen()

  const baseUrl = `http://${host}:${port}`
  log.info(`Listening at ${baseUrl}`)

  const messageSubmitter: L2ToL1MessageSubmitter = await runMessageSubmitter()

  return [fullnodeRpcServer, messageSubmitter]
}


const runMessageSubmitter = async (): Promise<L2ToL1MessageSubmitter> => {
  return undefined
}