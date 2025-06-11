import {
  DisputeGameBlacklisted as DisputeGameBlacklistedEvent,
  Initialized as InitializedEvent,
  RespectedGameTypeSet as RespectedGameTypeSetEvent,
  TransactionDeposited as TransactionDepositedEvent,
  WithdrawalFinalized as WithdrawalFinalizedEvent,
  WithdrawalProven as WithdrawalProvenEvent,
  WithdrawalProvenExtension1 as WithdrawalProvenExtension1Event
} from "../generated/OptimismPortal/OptimismPortal"
import {
  DisputeGameBlacklisted,
  Initialized,
  RespectedGameTypeSet,
  TransactionDeposited,
  WithdrawalFinalized,
  WithdrawalProven,
  WithdrawalProvenExtension1
} from "../generated/schema"

export function handleDisputeGameBlacklisted(
  event: DisputeGameBlacklistedEvent
): void {
  let entity = new DisputeGameBlacklisted(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.disputeGame = event.params.disputeGame

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

export function handleRespectedGameTypeSet(
  event: RespectedGameTypeSetEvent
): void {
  let entity = new RespectedGameTypeSet(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.newGameType = event.params.newGameType
  entity.updatedAt = event.params.updatedAt

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleTransactionDeposited(
  event: TransactionDepositedEvent
): void {
  let entity = new TransactionDeposited(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.from = event.params.from
  entity.to = event.params.to
  entity.version = event.params.version
  entity.opaqueData = event.params.opaqueData

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleWithdrawalFinalized(
  event: WithdrawalFinalizedEvent
): void {
  let entity = new WithdrawalFinalized(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.withdrawalHash = event.params.withdrawalHash
  entity.success = event.params.success

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleWithdrawalProven(event: WithdrawalProvenEvent): void {
  let entity = new WithdrawalProven(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.withdrawalHash = event.params.withdrawalHash
  entity.from = event.params.from
  entity.to = event.params.to

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleWithdrawalProvenExtension1(
  event: WithdrawalProvenExtension1Event
): void {
  let entity = new WithdrawalProvenExtension1(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.withdrawalHash = event.params.withdrawalHash
  entity.proofSubmitter = event.params.proofSubmitter

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}
