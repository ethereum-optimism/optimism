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
import { BigInt, Bytes, ethereum } from "@graphprotocol/graph-ts"

/**
 * Extracts l2BlockNumber from extraData bytes
 * The l2BlockNumber is encoded in the last 8 bytes of the extraData
 */
function extractL2BlockNumberFromExtraData(extraData: Bytes): BigInt | null {
  if (extraData.length < 8) {
    return null
  }

  // Get the last 8 bytes
  let startIndex = extraData.length - 8
  let l2BlockNumberBytes = new Array<i32>(8)
  for (let i = 0; i < 8; i++) {
    l2BlockNumberBytes[i] = extraData[startIndex + i]
  }

  // Convert bytes to BigInt (big-endian)
  let l2BlockNumber = BigInt.fromString("0")
  for (let i = 0; i < l2BlockNumberBytes.length; i++) {
    l2BlockNumber = l2BlockNumber.leftShift(8).plus(BigInt.fromI32(l2BlockNumberBytes[i]))
  }

  return l2BlockNumber
}

/**
 * Extracts l2BlockNumber from transaction input using ABI decoding
 */
function extractL2BlockNumberFromTxInput(txInput: Bytes): BigInt | null {
  if (txInput.length < 4) { // At least function selector
    return null
  }

  // Skip the 4-byte function selector to get the parameters
  let paramsData = Bytes.fromUint8Array(txInput.subarray(4))

  // Decode the parameters: (uint32, bytes32, bytes)
  let decoded = ethereum.decode("(uint32,bytes32,bytes)", paramsData)
  if (decoded) {
    // Extract extraData (third parameter)
    let extraData = decoded.toTuple()[2].toBytes()
    return extractL2BlockNumberFromExtraData(extraData)
  }

  return null
}

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

  // Extract l2BlockNumber from transaction input using ABI decoding
  entity.l2BlockNumber = extractL2BlockNumberFromTxInput(event.transaction.input)

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

  // Create template instances based on game type
  // You'll need to determine which game type corresponds to which template
  if (event.params.gameType.equals(BigInt.fromI32(0))) {
    // Assuming game type 0 is FaultDisputeGame
    FaultDisputeGame.create(event.params.disputeProxy)
  } else if (event.params.gameType.equals(BigInt.fromI32(1))) {
    // Assuming game type 1 is PermissionedDisputeGame
    PermissionedDisputeGame.create(event.params.disputeProxy)
  }
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
