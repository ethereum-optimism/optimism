/**
 * Optimism Copyright 2020
 * MIT License
 */

import { Logger } from '@ethersproject/logger'
import { Network, Networkish } from '@ethersproject/networks'
import {
  UrlJsonRpcProvider,
  JsonRpcSigner,
  JsonRpcProvider,
  Web3Provider,
} from '@ethersproject/providers'
import { defineReadOnly, getStatic } from '@ethersproject/properties'
import { ConnectionInfo } from '@ethersproject/web'
import { Provider } from '@ethersproject/abstract-provider'
import { OptimismSigner } from './signer'
import * as utils from './utils'
import { getNetwork, getUrl } from './network'

import pkg = require('../../package.json')
const version = pkg.version
const logger = new Logger(version)

// TODO edge cases
// static getNetwork
// it shouldn't call `get_getChainId` before every call
// when calling the hosted node

export class OptimismProvider extends JsonRpcProvider {
  private readonly _ethereum: Web3Provider

  constructor(network?: Networkish, provider?: Web3Provider) {
    const net = getNetwork(network)
    const connectionInfo = getUrl(net, network)

    super(connectionInfo)
    this._ethereum = provider
  }

  public get ethereum() {
    return this._ethereum
  }

  public getSigner(address?: string): OptimismSigner {
    if (this.ethereum) {
      return new OptimismSigner(this.ethereum, this, address)
    }

    logger.throwError(
      'no web3 instance provided',
      Logger.errors.UNSUPPORTED_OPERATION,
      {
        operation: 'getSigner',
      }
    )
  }

  // `send` takes the literal RPC method name. The signer cannot use this
  // codepath, it is for querying an optimism node.
  public async send(method: string, params: any[]): Promise<any> {
    // Prevent certain calls from hitting the public nodes
    if (utils.isBlacklistedMethod(method)) {
      logger.throwError(
        'blacklisted operation',
        Logger.errors.UNSUPPORTED_OPERATION,
        {
          operation: method,
        }
      )
    }

    return super.send(method, params)
  }

  public prepareRequest(method: string, params: any): [string, any[]] {
    switch (method) {
      case 'sendTransaction':
      case 'sendEthSignTransaction':
        return ['eth_sendRawEthSignTransaction', [params.signedTransaction]]
    }

    return super.prepareRequest(method, params)
  }

  public async perform(method: string, params: any): Promise<any> {
    return super.perform(method, params)
  }
}
