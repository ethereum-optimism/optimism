import type { Event } from 'ethers'

interface TypedEvent<TArgsArray extends Array<any> = any, TArgsObject = any>
  extends Event {
  args: TArgsArray & TArgsObject
}

export interface AttestationCreatedEventObject {
  creator: string
  about: string
  key: string
  val: string
}

export type AttestationCreatedEvent = TypedEvent<
  [string, string, string, string],
  AttestationCreatedEventObject
>
