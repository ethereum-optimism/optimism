import * as W3 from 'web3'
const Web3 = require('web3') // tslint:disable-line

import { EthClient } from '../../../interfaces'

/**
 * Simple EthClient implementation that uses Web3 over
 * HTTP under the hood.
 */
export class Web3EthClient implements EthClient {
  public readonly web3: W3.default

  /**
   * Creates the client.
   * @param endpoint HTTP endpoint to connect to.
   */
  constructor(endpoint = 'http://127.0.0.1:8545') {
    this.web3 = new Web3(endpoint)
  }

  /**
   * @returns `true` if connected via web3, `false` otherwise.
   */
  public async connected(): Promise<boolean> {
    try {
      await this.web3.eth.net.isListening()
      return true
    } catch (e) {
      return false
    }
  }
}
