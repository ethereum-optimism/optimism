import {
  Initialized as InitializedEvent,
  TransactionDeposited as TransactionDepositedEvent,
  WithdrawalFinalized as WithdrawalFinalizedEvent,
  WithdrawalProven as WithdrawalProvenEvent,
} from "../generated/OptimismPortal/OptimismPortal"
import {
  Initialized,
  TransactionDeposited,
  WithdrawalFinalized,
  WithdrawalProven,
} from "../generated/schema"

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

export function handleTransactionDeposited(
  event: TransactionDepositedEvent,
): void {
  let entity = new TransactionDeposited(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
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
  event: WithdrawalFinalizedEvent,
): void {
  let entity = new WithdrawalFinalized(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
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
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.withdrawalHash = event.params.withdrawalHash
  entity.from = event.params.from
  entity.to = event.params.to

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}
