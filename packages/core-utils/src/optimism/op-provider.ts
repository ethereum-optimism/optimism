import EventEmitter from 'events'

import { BigNumber } from '@ethersproject/bignumber'
import { deepCopy } from '@ethersproject/properties'
import { ConnectionInfo, fetchJson } from '@ethersproject/web'

const getResult = (payload: {
  error?: { code?: number; data?: any; message?: string }
  result?: any
}): any => {
  if (payload.error) {
    const error: any = new Error(payload.error.message)
    error.code = payload.error.code
    error.data = payload.error.data
    throw error
  }
  return payload.result
}

export interface BlockDescriptor {
  hash: string
  number: BigNumber
  parentHash: string
  timestamp: BigNumber
}

export interface L2BlockDescriptor extends BlockDescriptor {
  l1Origin: {
    hash: string
    number: BigNumber
  }
  sequencerNumber: BigNumber
}

export interface SyncStatusResponse {
  currentL1: BlockDescriptor
  headL1: BlockDescriptor
  unsafeL2: L2BlockDescriptor
  safeL2: L2BlockDescriptor
  finalizedL2: L2BlockDescriptor
}

export class OpNodeProvider extends EventEmitter {
  readonly connection: ConnectionInfo
  private _nextId: number = 0

  constructor(url?: ConnectionInfo | string) {
    super()

    if (typeof url === 'string') {
      this.connection = { url }
    } else {
      this.connection = url
    }
  }

  async syncStatus(): Promise<SyncStatusResponse> {
    const result = await this.send('optimism_syncStatus', [])

    return {
      currentL1: {
        hash: result.current_l1.hash,
        number: BigNumber.from(result.current_l1.number),
        parentHash: result.current_l1.parentHash,
        timestamp: BigNumber.from(result.current_l1.timestamp),
      },
      headL1: {
        hash: result.head_l1.hash,
        number: BigNumber.from(result.head_l1.number),
        parentHash: result.head_l1.parentHash,
        timestamp: BigNumber.from(result.head_l1.timestamp),
      },
      unsafeL2: {
        hash: result.unsafe_l2.hash,
        number: BigNumber.from(result.unsafe_l2.number),
        parentHash: result.unsafe_l2.parentHash,
        timestamp: BigNumber.from(result.unsafe_l2.timestamp),
        l1Origin: {
          hash: result.unsafe_l2.l1origin.hash,
          number: BigNumber.from(result.unsafe_l2.l1origin.number),
        },
        sequencerNumber: BigNumber.from(result.unsafe_l2.sequenceNumber),
      },
      safeL2: {
        hash: result.safe_l2.hash,
        number: BigNumber.from(result.safe_l2.number),
        parentHash: result.safe_l2.parentHash,
        timestamp: BigNumber.from(result.safe_l2.timestamp),
        l1Origin: {
          hash: result.safe_l2.l1origin.hash,
          number: BigNumber.from(result.safe_l2.l1origin.number),
        },
        sequencerNumber: BigNumber.from(result.safe_l2.sequenceNumber),
      },
      finalizedL2: {
        hash: result.finalized_l2.hash,
        number: BigNumber.from(result.finalized_l2.number),
        parentHash: result.finalized_l2.parentHash,
        timestamp: BigNumber.from(result.finalized_l2.timestamp),
        l1Origin: {
          hash: result.finalized_l2.l1origin.hash,
          number: BigNumber.from(result.finalized_l2.l1origin.number),
        },
        sequencerNumber: BigNumber.from(result.finalized_l2.sequenceNumber),
      },
    }
  }

  // TODO(tynes): turn the response into a stronger type
  async rollupConfig() {
    const result = await this.send('optimism_rollupConfig', [])
    return result
  }

  send(method: string, params: Array<any>): Promise<any> {
    const request = {
      method,
      params,
      id: this._nextId++,
      jsonrpc: '2.0',
    }

    this.emit('debug', {
      action: 'request',
      request: deepCopy(request),
      provider: this,
    })

    const result = fetchJson(
      this.connection,
      JSON.stringify(request),
      getResult
    ).then(
      (res) => {
        this.emit('debug', {
          action: 'response',
          request,
          response: res,
          provider: this,
        })

        return res
      },
      (error) => {
        this.emit('debug', {
          action: 'response',
          error,
          request,
          provider: this,
        })

        throw error
      }
    )

    return result
  }
}
