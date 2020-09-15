/**
 * Copyright 2020, Optimism PBC
 * MIT License
 * https://github.com/ethereum-optimism
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
import { verifyMessage } from '@ethersproject/wallet'
import { Provider } from '@ethersproject/abstract-provider'
import { joinSignature, SignatureLike } from '@ethersproject/bytes'
import { OptimismSigner } from './signer'
import * as utils from './utils'
import { getNetwork, getUrl } from './network'

import pkg = require('../../package.json')
const version = pkg.version
const logger = new Logger(version)

/**
 * The OptimismProvider is an ethers.js JsonRpcProvider that
 * utilizes a new signature hashing scheme meant for usage with
 * the Optimism node. Transactions that are signed with this scheme
 * are sent to a new endpoint `eth_sendRawEthSignTransaction`.
 */

export class OptimismProvider extends JsonRpcProvider {
  private readonly _ethereum: Web3Provider

  constructor(network?: Networkish, provider?: Web3Provider) {
    const net = getNetwork(network)
    const connectionInfo = getUrl(net, network)

    super(connectionInfo)
    this._ethereum = provider

    // Handle properly deriving "from" on the transaction
    const format = this.formatter.transaction
    this.formatter.transaction = (transaction) => {
      const tx = format(transaction)
      const sig = joinSignature(tx as SignatureLike)
      const hash = utils.sighashEthSign(tx)
      // need to concat and hash with
      tx.from = verifyMessage(hash, sig)
      return tx
    }
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
