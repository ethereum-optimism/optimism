import { newMockEvent } from "matchstick-as"
import { ethereum, Address, BigInt, Bytes } from "@graphprotocol/graph-ts"
import {
  DisputeGameCreated,
  ImplementationSet,
  InitBondUpdated,
  Initialized,
  OwnershipTransferred
} from "../generated/DisputeGameFactory/DisputeGameFactory"

export function createDisputeGameCreatedEvent(
  disputeProxy: Address,
  gameType: BigInt,
  rootClaim: Bytes
): DisputeGameCreated {
  let disputeGameCreatedEvent = changetype<DisputeGameCreated>(newMockEvent())

  disputeGameCreatedEvent.parameters = new Array()

  disputeGameCreatedEvent.parameters.push(
    new ethereum.EventParam(
      "disputeProxy",
      ethereum.Value.fromAddress(disputeProxy)
    )
  )
  disputeGameCreatedEvent.parameters.push(
    new ethereum.EventParam(
      "gameType",
      ethereum.Value.fromUnsignedBigInt(gameType)
    )
  )
  disputeGameCreatedEvent.parameters.push(
    new ethereum.EventParam(
      "rootClaim",
      ethereum.Value.fromFixedBytes(rootClaim)
    )
  )

  return disputeGameCreatedEvent
}

export function createImplementationSetEvent(
  impl: Address,
  gameType: BigInt
): ImplementationSet {
  let implementationSetEvent = changetype<ImplementationSet>(newMockEvent())

  implementationSetEvent.parameters = new Array()

  implementationSetEvent.parameters.push(
    new ethereum.EventParam("impl", ethereum.Value.fromAddress(impl))
  )
  implementationSetEvent.parameters.push(
    new ethereum.EventParam(
      "gameType",
      ethereum.Value.fromUnsignedBigInt(gameType)
    )
  )

  return implementationSetEvent
}

export function createInitBondUpdatedEvent(
  gameType: BigInt,
  newBond: BigInt
): InitBondUpdated {
  let initBondUpdatedEvent = changetype<InitBondUpdated>(newMockEvent())

  initBondUpdatedEvent.parameters = new Array()

  initBondUpdatedEvent.parameters.push(
    new ethereum.EventParam(
      "gameType",
      ethereum.Value.fromUnsignedBigInt(gameType)
    )
  )
  initBondUpdatedEvent.parameters.push(
    new ethereum.EventParam(
      "newBond",
      ethereum.Value.fromUnsignedBigInt(newBond)
    )
  )

  return initBondUpdatedEvent
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

export function createOwnershipTransferredEvent(
  previousOwner: Address,
  newOwner: Address
): OwnershipTransferred {
  let ownershipTransferredEvent = changetype<OwnershipTransferred>(
    newMockEvent()
  )

  ownershipTransferredEvent.parameters = new Array()

  ownershipTransferredEvent.parameters.push(
    new ethereum.EventParam(
      "previousOwner",
      ethereum.Value.fromAddress(previousOwner)
    )
  )
  ownershipTransferredEvent.parameters.push(
    new ethereum.EventParam("newOwner", ethereum.Value.fromAddress(newOwner))
  )

  return ownershipTransferredEvent
}
