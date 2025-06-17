import {
  DisputeGameCreated as DisputeGameCreatedEvent,
  ImplementationSet as ImplementationSetEvent,
  InitBondUpdated as InitBondUpdatedEvent,
  Initialized as InitializedEvent,
  OwnershipTransferred as OwnershipTransferredEvent,
} from "../generated/DisputeGameFactory/DisputeGameFactory"
import {
  DisputeGameCreated,
  ImplementationSet,
  InitBondUpdated,
  Initialized,
  OwnershipTransferred,
  DisputeGameCreatedIndex,
} from "../generated/schema"
import { FaultDisputeGame } from "../generated/templates"
import { BigInt, Bytes } from "@graphprotocol/graph-ts"

export function handleDisputeGameCreated(event: DisputeGameCreatedEvent): void {
  if (!event.params.disputeProxy) {
    return
  }

  // so we can retrieve it in our FaultDisputeGame subgraph
  let entity = new DisputeGameCreated(event.params.disputeProxy)

  let newIndex = BigInt.fromI32(0)
  let latestEntity = DisputeGameCreatedIndex.load(Bytes.fromUTF8("latest"))
  if (latestEntity != null) {
    newIndex = latestEntity.index.plus(BigInt.fromI32(1))
  }
  entity.index = newIndex

  entity.disputeProxy = event.params.disputeProxy
  entity.gameType = event.params.gameType
  entity.rootClaim = event.params.rootClaim
  entity.resolvedStatus = 0

  /** @DEV needs implementation */
  // let data = event.transaction.input.toHex()
  // let l2BlockNumberHex = data.slice(2 + 2 * 0x54, 2 + 2 * (0x54 + 32))
  entity.rootClaim = event.params.rootClaim
  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash
  entity.save()

  // Update the latest index entity
  if (latestEntity == null) {
    latestEntity = new DisputeGameCreatedIndex(Bytes.fromUTF8("latest"))
  }
  latestEntity.index = newIndex
  latestEntity.save()

  // Create a new FaultDisputeGame template instance
  FaultDisputeGame.create(event.params.disputeProxy)
}

export function handleImplementationSet(event: ImplementationSetEvent): void {
  let entity = new ImplementationSet(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.impl = event.params.impl
  entity.gameType = event.params.gameType

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleInitBondUpdated(event: InitBondUpdatedEvent): void {
  let entity = new InitBondUpdated(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.gameType = event.params.gameType
  entity.newBond = event.params.newBond

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleInitialized(event: InitializedEvent): void {
  let entity = new Initialized(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.version = event.params.version

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleOwnershipTransferred(
  event: OwnershipTransferredEvent,
): void {
  let entity = new OwnershipTransferred(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.previousOwner = event.params.previousOwner
  entity.newOwner = event.params.newOwner

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}
