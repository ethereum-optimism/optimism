import { BaseProvider } from '@ethersproject/providers'
import { BigNumber } from 'ethers'
import { TypedEvent } from '@eth-optimism/contracts/dist/types/common'

import {
  TransactionBatchEntry,
  TransactionEntry,
  StateRootBatchEntry,
  StateRootEntry,
} from './database-types'
import { TransportDB } from '../db/transport-db'

export type GetExtraDataHandler<TEvent extends TypedEvent, TExtraData> = (
  event?: TEvent,
  l1RpcProvider?: BaseProvider
) => Promise<TExtraData>

export type ParseEventHandler<
  TEvent extends TypedEvent,
  TExtraData,
  TParsedEvent
> = (event: TEvent, extraData: TExtraData, l2ChainId: number) => TParsedEvent

export type StoreEventHandler<TParsedEvent> = (
  parsedEvent: TParsedEvent,
  db: TransportDB
) => Promise<void>

export interface EventHandlerSet<
  TEvent extends TypedEvent,
  TExtraData,
  TParsedEvent
> {
  getExtraData: GetExtraDataHandler<TEvent, TExtraData>
  parseEvent: ParseEventHandler<TEvent, TExtraData, TParsedEvent>
  storeEvent: StoreEventHandler<TParsedEvent>
}

export interface SequencerBatchAppendedExtraData {
  timestamp: number
  blockNumber: number
  submitter: string
  l1TransactionData: string
  l1TransactionHash: string

  // Stuff from TransactionBatchAppended.
  prevTotalElements: BigNumber
  batchIndex: BigNumber
  batchSize: BigNumber
  batchRoot: string
  batchExtraData: string
}

export interface SequencerBatchAppendedParsedEvent {
  transactionBatchEntry: TransactionBatchEntry
  transactionEntries: TransactionEntry[]
}

export interface StateBatchAppendedExtraData {
  timestamp: number
  blockNumber: number
  submitter: string
  l1TransactionHash: string
  l1TransactionData: string
}

export interface StateBatchAppendedParsedEvent {
  stateRootBatchEntry: StateRootBatchEntry
  stateRootEntries: StateRootEntry[]
}
