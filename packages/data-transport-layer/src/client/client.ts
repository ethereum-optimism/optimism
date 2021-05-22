// Only load if not in browser.
import { isNode } from 'browser-or-node'

declare var window: any

/* tslint:disable-next-line:no-var-requires */
const fetch = isNode ? require('node-fetch') : window.fetch

import {
  EnqueueResponse,
  StateRootBatchResponse,
  StateRootResponse,
  SyncingResponse,
  TransactionBatchResponse,
  TransactionResponse,
} from '../types'

export class L1DataTransportClient {
  public _chainId:number
  constructor(private url: string) {this._chainId=0}
  
  public setChainId(chainId:number){
    this._chainId=chainId
  }

  public async syncing(): Promise<SyncingResponse> {
    return this._get(`/eth/syncing/${this._chainId}`)
  }

  public async getEnqueueByIndex(index: number): Promise<EnqueueResponse> {
    return this._get(`/enqueue/index/${index}/${this._chainId}`)
  }

  public async getLatestEnqueue(): Promise<EnqueueResponse> {
    return this._get(`/enqueue/latest/${this._chainId}`)
  }

  public async getTransactionByIndex(
    index: number
  ): Promise<TransactionResponse> {
    return this._get(`/transaction/index/${index}/${this._chainId}`)
  }

  public async getLatestTransacton(): Promise<TransactionResponse> {
    return this._get(`/transaction/latest/${this._chainId}`)
  }

  public async getTransactionBatchByIndex(
    index: number
  ): Promise<TransactionBatchResponse> {
    return this._get(`/batch/transaction/index/${index}/${this._chainId}`)
  }

  public async getLatestTransactionBatch(): Promise<TransactionBatchResponse> {
    return this._get(`/batch/transaction/latest/${this._chainId}`)
  }

  public async getStateRootByIndex(index: number): Promise<StateRootResponse> {
    return this._get(`/stateroot/index/${index}/${this._chainId}`)
  }

  public async getLatestStateRoot(): Promise<StateRootResponse> {
    return this._get(`/stateroot/latest/${this._chainId}`)
  }

  public async getStateRootBatchByIndex(
    index: number
  ): Promise<StateRootBatchResponse> {
    return this._get(`/batch/stateroot/index/${index}/${this._chainId}`)
  }

  public async getLatestStateRootBatch(): Promise<StateRootBatchResponse> {
    return this._get(`/batch/stateroot/latest/${this._chainId}`)
  }

  private async _get<TResponse>(endpoint: string): Promise<TResponse> {
    return (await fetch(`${this.url}${endpoint}`)).json()
  }
}
