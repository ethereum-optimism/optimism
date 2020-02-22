/* External Imports */
import { ExpressHttpServer, getLogger, Logger } from '@eth-optimism/core-utils'

/* Internal Imports */
import { FullnodeRpcServer, DefaultWeb3Handler } from '../app'
import { TestWeb3Handler } from '../app/test-handler'

const log: Logger = getLogger('rollup-fullnode')

/**
 * Runs a fullnode.
 * @param testFullnode Whether or not this is a test.
 */
export const runFullnode = async (
  testFullnode: boolean = false
): Promise<ExpressHttpServer> => {
  // TODO Get these from config
  const host = '0.0.0.0'
  const port = 8545

  const fullnodeHandler = testFullnode
    ? await TestWeb3Handler.create()
    : await DefaultWeb3Handler.create()
  const fullnodeRpcServer = new FullnodeRpcServer(fullnodeHandler, host, port)

  fullnodeRpcServer.listen()

  const baseUrl = `http://${host}:${port}`
  log.info(`Listening at ${baseUrl}`)

  return fullnodeRpcServer
}
