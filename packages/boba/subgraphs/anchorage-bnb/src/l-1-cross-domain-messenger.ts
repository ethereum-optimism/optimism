import {
  FailedRelayedMessage as FailedRelayedMessageEvent,
  Initialized as InitializedEvent,
  RelayedMessage as RelayedMessageEvent,
  SentMessage as SentMessageEvent,
  SentMessageExtension1 as SentMessageExtension1Event
} from "../generated/L1CrossDomainMessenger/L1CrossDomainMessenger"
import {
  FailedRelayedMessage,
  Initialized,
  RelayedMessage,
  SentMessage,
  SentMessageExtension1
} from "../generated/schema"

export function handleFailedRelayedMessage(
  event: FailedRelayedMessageEvent
): void {
  let entity = new FailedRelayedMessage(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.msgHash = event.params.msgHash

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleInitialized(event: InitializedEvent): void {
  let entity = new Initialized(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.version = event.params.version

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleRelayedMessage(event: RelayedMessageEvent): void {
  let entity = new RelayedMessage(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.msgHash = event.params.msgHash

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleSentMessage(event: SentMessageEvent): void {
  let entity = new SentMessage(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.target = event.params.target
  entity.sender = event.params.sender
  entity.message = event.params.message
  entity.messageNonce = event.params.messageNonce
  entity.gasLimit = event.params.gasLimit

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleSentMessageExtension1(
  event: SentMessageExtension1Event
): void {
  let entity = new SentMessageExtension1(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.sender = event.params.sender
  entity.value = event.params.value

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}
