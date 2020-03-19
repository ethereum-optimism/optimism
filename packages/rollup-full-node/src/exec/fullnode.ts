/* External Imports */
import { ExpressHttpServer, getLogger, Logger } from '@eth-optimism/core-utils'
import { JsonRpcProvider } from 'ethers/providers'

/* Internal Imports */
import {
  FullnodeRpcServer,
  DefaultWeb3Handler,
  TestWeb3Handler,
  DEFAULT_ETHNODE_GAS_LIMIT,
} from '../app'

const log: Logger = getLogger('rollup-fullnode')

/**
 * Runs a fullnode.
 * @param testFullnode Whether or not this is a test.
 */
export const runFullnode = async (
  testFullnode: boolean = false
): Promise<ExpressHttpServer> => {
  let provider: JsonRpcProvider
  // TODO Get these from config
  const host = '0.0.0.0'
  const port = 8545

  log.info(`Starting fullnode in ${testFullnode ? 'TEST' : 'LIVE'} mode`)

  if (process.env.WEB3_URL) {
    provider = new JsonRpcProvider(process.env.WEB3_URL)
  }

  const fullnodeHandler = testFullnode
    ? await TestWeb3Handler.create(provider)
    : await DefaultWeb3Handler.create(provider)
  const fullnodeRpcServer = new FullnodeRpcServer(fullnodeHandler, host, port)

  fullnodeRpcServer.listen()

  const baseUrl = `http://${host}:${port}`
  log.info(`Listening at ${baseUrl}`)

  return fullnodeRpcServer
}
