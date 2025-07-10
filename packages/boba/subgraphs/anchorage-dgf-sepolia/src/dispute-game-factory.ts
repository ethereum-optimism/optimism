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
import { FaultDisputeGame, PermissionedDisputeGame } from "../generated/templates"
import { FaultDisputeGame as FaultDisputeGameContract } from "../generated/templates/FaultDisputeGame/FaultDisputeGame"
import { BigInt, Bytes, ethereum } from "@graphprotocol/graph-ts"
import { log } from '@graphprotocol/graph-ts'

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
  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  // Watch out! We are using the FaultGameDispute contract here
  // but other game types might not match its ABI.  We tolerate
  // reversion below, but don't re-use this contract with additional
  // methods without realizing PermissionedDisputeGame must support
  // the ABI as well
  let contract = FaultDisputeGameContract.bind(event.params.disputeProxy)
  let l2BlockNumberResult = contract.try_l2BlockNumber()
  if (!l2BlockNumberResult.reverted) {
    entity.l2BlockNumber = l2BlockNumberResult.value
  } else {
    log.warning("could not resolve l2BlockNumber for {}", [event.params.disputeProxy.toHexString()])
  }
  entity.save()

  // Create template instances based on game type
  // You'll need to determine which game type corresponds to which template
  if (event.params.gameType.equals(BigInt.fromI32(0))) {
    log.info("Creating fault dispute gametype for {}", [event.params.disputeProxy.toHexString()])
    FaultDisputeGame.create(event.params.disputeProxy)
  } else if (event.params.gameType.equals(BigInt.fromI32(1))) {
    // Assuming game type 1 is PermissionedDisputeGame
    log.info("Creating permissioned dispute gametype for {}", [event.params.disputeProxy.toHexString()])
    PermissionedDisputeGame.create(event.params.disputeProxy)
  } else {
    log.warning("Unsupported gametype {} for {}", [event.params.gameType.toHexString(), event.params.disputeProxy.toHexString()])
  }

  // Update the latest index entity
  if (latestEntity == null) {
    latestEntity = new DisputeGameCreatedIndex(Bytes.fromUTF8("latest"))
  }
  latestEntity.index = newIndex
  latestEntity.save()

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
