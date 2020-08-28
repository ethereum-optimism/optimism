/**
 * Optimism Copyright 2020
 * MIT License
 */

// TODO: clean up dead code
import { Logger } from "@ethersproject/logger";
import { Network, Networkish } from "@ethersproject/networks";
import { UrlJsonRpcProvider, JsonRpcSigner, JsonRpcProvider, Web3Provider } from '@ethersproject/providers'
import { defineReadOnly, getStatic } from "@ethersproject/properties";
import { ConnectionInfo } from "@ethersproject/web";
import { Provider } from '@ethersproject/abstract-provider';
import { OptimismSigner } from './signer'
import * as utils from './utils'
import { getNetwork, getUrl } from './network'

import pkg = require('../../package.json')
const version = pkg.version
const logger = new Logger(version);

// TODO edge cases
// static getNetwork
// ens names
// it shouldn't call `get_getChainId` before every call
// when calling the hosted node

export class OptimismProvider extends JsonRpcProvider {
  private readonly _ethereum: Web3Provider

  constructor(network?: Networkish, provider?: Web3Provider) {
    const net = getNetwork(network)
    const connectionInfo = getUrl(net, network)

    super(connectionInfo);
  }

  public get ethereum() {
    return this._ethereum
  }

  public getSigner(address?: string): OptimismSigner | JsonRpcSigner {
    if (this.ethereum) {
      const signer = this.ethereum.getSigner(address);
      return new OptimismSigner(this, signer, address)
    }

    return super.getSigner()
  }

  // `send` takes the literal RPC method name
  public async send(method: string, params: any[]): Promise<any> {
    // if being called from the signer, certain calls need to get through.

    // Prevent certain calls from hitting the public nodes
    if (utils.isBlacklistedMethod(method)) {
      logger.throwError('blacklisted operation', Logger.errors.UNSUPPORTED_OPERATION, {
        operation: method
      });
    }

    return super.send(method, params)
  }

  // TODO: special case:
  //"sendTransaction" -> "eth_sendRawTransaction"

  // `perform` accepts more human-friendly method names that are usually the
  // name of the RPC method without the `eth_` prefix.
  public async perform(method: string, params: any): Promise<any> {
    if (method === 'sendRawTransaction') {
      // TODO:
    }

    return super.perform(method, params)
  }
}
