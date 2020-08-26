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

import pkg = require('../../package.json')
const version = pkg.version
const logger = new Logger(version);

// TODO: maybe change this to JsonRpcProvider
export class OptimismProvider extends UrlJsonRpcProvider {
  private _ethereum: Web3Provider

  constructor(network?: Networkish, provider?: Web3Provider) {
    // Must construct with `new`
    logger.checkAbstract(new.target, OptimismProvider)

    super(network);

    this._ethereum = provider || null;
  }

  public get ethereum() {
    return this._ethereum
  }

  public getSigner(address?: string): OptimismSigner {
    if (this.ethereum) {
      const signer = this.ethereum.getSigner(address);
      return new OptimismSigner(this, signer, address)
    }

    logger.throwError("no web3 provider", Logger.errors.UNSUPPORTED_OPERATION, {
      operation: "getSigner"
    });
  }

  public async send(method: string, params: any[]): Promise<any> {
    if (utils.isBlacklistedMethod(method)) {
      logger.throwError('blacklisted operation', Logger.errors.UNSUPPORTED_OPERATION, {
        operation: method
      });
    }

    return super.send(method, params)
  }

  // TODO: special case:
  //"sendTransaction" -> "eth_sendRawTransaction"
  public async perform(method: string, params: any): Promise<any> {
    super.perform(method, params)
  }

  // Based on the newtork, return the public URL of the optimism nodes
  public static getUrl(network: Network, apiKey: any): string | ConnectionInfo {
    let host: string = null
    switch (network ? network.name : 'unknown') {
      case 'dev':
        host = 'localhost:8546'
        break
      default:
        logger.throwError("unsupported network", Logger.errors.INVALID_ARGUMENT, {
          argument: "network",
          value: network
        });
    }

    const connection: ConnectionInfo = {
      url: `http://${host}`
    };

    return connection
  }
}
