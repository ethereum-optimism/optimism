/* External Imports */
import { getLogger, Logger } from '@eth-optimism/core-utils'

/* Internal Imports */
import { FullnodeRpcServer, DefaultWeb3Handler } from '../app'

const log: Logger = getLogger('rollup-fullnode')

export const runFullnode = async (): Promise<void> => {
  // TODO Get these from config
  const host = '0.0.0.0'
  const port = 8545

  const fullnodeHandler = await DefaultWeb3Handler.create()
  const fullnodeRpcServer = new FullnodeRpcServer(fullnodeHandler, host, port)

  fullnodeRpcServer.listen()

  const baseUrl = `http://${host}:${port}`
  log.info(`Listening at ${baseUrl}`)
}

// Start Fullnode
runFullnode()
