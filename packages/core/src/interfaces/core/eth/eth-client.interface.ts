import BigNum = require('bn.js')

import {
  TransactionReceipt,
  EventFilter,
  EventLog,
  Abi,
  Contract,
} from '../../common'
import Web3 from 'web3/types'

/**
 * EthClient exposes an API for interacting with Ethereum.
 */
export interface EthClient {
  readonly web3: Web3

  /**
   * @returns `true` if connected to the node, `false` otherwise.
   */
  connected(): Promise<boolean>
}
