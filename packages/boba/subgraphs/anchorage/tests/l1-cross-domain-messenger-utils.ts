import { newMockEvent } from "matchstick-as"
import { ethereum, Bytes, Address, BigInt } from "@graphprotocol/graph-ts"
import {
  FailedRelayedMessage,
  Initialized,
  RelayedMessage,
  SentMessage,
  SentMessageExtension1
} from "../generated/L1CrossDomainMessenger/L1CrossDomainMessenger"

export function createFailedRelayedMessageEvent(
  msgHash: Bytes
): FailedRelayedMessage {
  let failedRelayedMessageEvent = changetype<FailedRelayedMessage>(
    newMockEvent()
  )

  failedRelayedMessageEvent.parameters = new Array()

  failedRelayedMessageEvent.parameters.push(
    new ethereum.EventParam("msgHash", ethereum.Value.fromFixedBytes(msgHash))
  )

  return failedRelayedMessageEvent
}

export function createInitializedEvent(version: i32): Initialized {
  let initializedEvent = changetype<Initialized>(newMockEvent())

  initializedEvent.parameters = new Array()

  initializedEvent.parameters.push(
    new ethereum.EventParam(
      "version",
      ethereum.Value.fromUnsignedBigInt(BigInt.fromI32(version))
    )
  )

  return initializedEvent
}

export function createRelayedMessageEvent(msgHash: Bytes): RelayedMessage {
  let relayedMessageEvent = changetype<RelayedMessage>(newMockEvent())

  relayedMessageEvent.parameters = new Array()

  relayedMessageEvent.parameters.push(
    new ethereum.EventParam("msgHash", ethereum.Value.fromFixedBytes(msgHash))
  )

  return relayedMessageEvent
}

export function createSentMessageEvent(
  target: Address,
  sender: Address,
  message: Bytes,
  messageNonce: BigInt,
  gasLimit: BigInt
): SentMessage {
  let sentMessageEvent = changetype<SentMessage>(newMockEvent())

  sentMessageEvent.parameters = new Array()

  sentMessageEvent.parameters.push(
    new ethereum.EventParam("target", ethereum.Value.fromAddress(target))
  )
  sentMessageEvent.parameters.push(
    new ethereum.EventParam("sender", ethereum.Value.fromAddress(sender))
  )
  sentMessageEvent.parameters.push(
    new ethereum.EventParam("message", ethereum.Value.fromBytes(message))
  )
  sentMessageEvent.parameters.push(
    new ethereum.EventParam(
      "messageNonce",
      ethereum.Value.fromUnsignedBigInt(messageNonce)
    )
  )
  sentMessageEvent.parameters.push(
    new ethereum.EventParam(
      "gasLimit",
      ethereum.Value.fromUnsignedBigInt(gasLimit)
    )
  )

  return sentMessageEvent
}

export function createSentMessageExtension1Event(
  sender: Address,
  value: BigInt
): SentMessageExtension1 {
  let sentMessageExtension1Event = changetype<SentMessageExtension1>(
    newMockEvent()
  )

  sentMessageExtension1Event.parameters = new Array()

  sentMessageExtension1Event.parameters.push(
    new ethereum.EventParam("sender", ethereum.Value.fromAddress(sender))
  )
  sentMessageExtension1Event.parameters.push(
    new ethereum.EventParam("value", ethereum.Value.fromUnsignedBigInt(value))
  )

  return sentMessageExtension1Event
}
