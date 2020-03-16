/* External Imports */
import { ExpressHttpServer, getLogger, Logger } from '@eth-optimism/core-utils'
import { createMockProvider } from 'ethereum-waffle'
import { JsonRpcProvider, Web3Provider } from 'ethers/providers'

/* Internal Imports */
import {
  FullnodeRpcServer,
  DefaultWeb3Handler,
  TestWeb3Handler,
  DEFAULT_ETHNODE_GAS_LIMIT,
} from '../app'
const fs = require('fs')
const dns = require('dns')
const http = require('http')
const axios = require('axios').default

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

  log.info(`Starting fullnode in ${testFullnode ? 'TEST' : 'LIVE'} mode..`)
  const backend: JsonRpcProvider = testFullnode
    ? createMockProvider({
        gasLimit: DEFAULT_ETHNODE_GAS_LIMIT,
        allowUnlimitedContractSize: true,
      })
    : new Web3Provider(new JsonRpcProvider('http://localhost:8545'))
  const fullnodeHandler = testFullnode
    ? await TestWeb3Handler.create(backend)
    : await DefaultWeb3Handler.create(backend)
  const fullnodeRpcServer = new FullnodeRpcServer(fullnodeHandler, host, port)

  fullnodeRpcServer.listen()

  const baseUrl = `http://${host}:${port}`
  log.info(`Listening at ${baseUrl}`)

  return fullnodeRpcServer
}
