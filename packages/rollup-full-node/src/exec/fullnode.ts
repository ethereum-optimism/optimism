/* External Imports */
import { ExpressHttpServer, getLogger, Logger } from '@eth-optimism/core-utils'
import { promisify } from 'util'
import { createMockProvider } from 'ethereum-waffle'
import { Contract, ethers, utils, Wallet } from 'ethers'
import { readFile as readFileAsync } from 'fs';
import { JsonRpcProvider, Web3Provider } from 'ethers/providers'
import axios from 'axios'

/* Internal Imports */
import {
  FullnodeRpcServer,
  DefaultWeb3Handler,
  TestWeb3Handler,
  DEFAULT_ETHNODE_GAS_LIMIT,
} from '../app'

const log: Logger = getLogger('rollup-fullnode')
const readFile = promisify(readFileAsync);

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
    : new JsonRpcProvider('http://geth:8545')

  await new Promise(r => setTimeout(r, 3000))
  const privateKey = await readFile("/root/.ethereum/private_key.txt");
  const wallet = new Wallet(`0x${privateKey}`, backend)
  log.info(`Address: ${(await wallet.getAddress()).toString()}`)
  log.info(`Balance: ${(await wallet.getBalance()).toString()}`)
  const tx = await wallet.sendTransaction({
    to: '0xf45b372480bb2eb803a4d99a8e935ff2d8e9adf5',
    value: 1
  });
  log.info('Sent in Transaction: ' + tx.hash);
  log.info(`testFullnode: ${testFullnode}`)
  const fullnodeHandler = testFullnode
    ? await TestWeb3Handler.create(backend)
    : await DefaultWeb3Handler.create(backend)
  const fullnodeRpcServer = new FullnodeRpcServer(fullnodeHandler, host, port)

  fullnodeRpcServer.listen()

  const baseUrl = `http://${host}:${port}`
  log.info(`Listening at ${baseUrl}`)

  return fullnodeRpcServer
}
