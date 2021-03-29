import { JsonRpcProvider } from '@ethersproject/providers'
import { TransportDB } from '../db/transport-db'
import { TypedEthersEvent } from './event-types'

export type GetExtraDataHandler<TEventArgs, TExtraData> = (
  event?: TypedEthersEvent<TEventArgs>,
  l1RpcProvider?: JsonRpcProvider
) => Promise<TExtraData>

export type ParseEventHandler<TEventArgs, TExtraData, TParsedEvent> = (
  event: TypedEthersEvent<TEventArgs>,
  extraData: TExtraData
) => TParsedEvent

export type StoreEventHandler<TParsedEvent> = (
  parsedEvent: TParsedEvent,
  db: TransportDB
) => Promise<void>

export interface EventHandlerSet<TEventArgs, TExtraData, TParsedEvent> {
  getExtraData: GetExtraDataHandler<TEventArgs, TExtraData>
  parseEvent: ParseEventHandler<TEventArgs, TExtraData, TParsedEvent>
  storeEvent: StoreEventHandler<TParsedEvent>
}
