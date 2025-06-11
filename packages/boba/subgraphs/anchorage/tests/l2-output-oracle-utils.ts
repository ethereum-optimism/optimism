import { newMockEvent } from "matchstick-as"
import { ethereum, Bytes, BigInt } from "@graphprotocol/graph-ts"
import {
  Initialized,
  OutputProposed,
  OutputsDeleted
} from "../generated/L2OutputOracle/L2OutputOracle"

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

export function createOutputProposedEvent(
  outputRoot: Bytes,
  l2OutputIndex: BigInt,
  l2BlockNumber: BigInt,
  l1Timestamp: BigInt
): OutputProposed {
  let outputProposedEvent = changetype<OutputProposed>(newMockEvent())

  outputProposedEvent.parameters = new Array()

  outputProposedEvent.parameters.push(
    new ethereum.EventParam(
      "outputRoot",
      ethereum.Value.fromFixedBytes(outputRoot)
    )
  )
  outputProposedEvent.parameters.push(
    new ethereum.EventParam(
      "l2OutputIndex",
      ethereum.Value.fromUnsignedBigInt(l2OutputIndex)
    )
  )
  outputProposedEvent.parameters.push(
    new ethereum.EventParam(
      "l2BlockNumber",
      ethereum.Value.fromUnsignedBigInt(l2BlockNumber)
    )
  )
  outputProposedEvent.parameters.push(
    new ethereum.EventParam(
      "l1Timestamp",
      ethereum.Value.fromUnsignedBigInt(l1Timestamp)
    )
  )

  return outputProposedEvent
}

export function createOutputsDeletedEvent(
  prevNextOutputIndex: BigInt,
  newNextOutputIndex: BigInt
): OutputsDeleted {
  let outputsDeletedEvent = changetype<OutputsDeleted>(newMockEvent())

  outputsDeletedEvent.parameters = new Array()

  outputsDeletedEvent.parameters.push(
    new ethereum.EventParam(
      "prevNextOutputIndex",
      ethereum.Value.fromUnsignedBigInt(prevNextOutputIndex)
    )
  )
  outputsDeletedEvent.parameters.push(
    new ethereum.EventParam(
      "newNextOutputIndex",
      ethereum.Value.fromUnsignedBigInt(newNextOutputIndex)
    )
  )

  return outputsDeletedEvent
}
