import {
  AssetBalanceWithdrawn as AssetBalanceWithdrawnEvent,
  AssetReceived as AssetReceivedEvent,
  DisbursementFailed as DisbursementFailedEvent,
  DisbursementRetrySuccess as DisbursementRetrySuccessEvent,
  DisbursementSuccess as DisbursementSuccessEvent,
  DisburserTransferred as DisburserTransferredEvent,
  MaxDepositAmountSet as MaxDepositAmountSetEvent,
  MaxTransferAmountPerDaySet as MaxTransferAmountPerDaySetEvent,
  MinDepositAmountSet as MinDepositAmountSetEvent,
  OwnershipTransferred as OwnershipTransferredEvent,
  Paused as PausedEvent,
  TokenSupported as TokenSupportedEvent,
  Unpaused as UnpausedEvent
} from "../generated/LightBridge/LightBridge"
import {
  AssetBalanceWithdrawn,
  AssetReceived,
  DisbursementFailed,
  DisbursementRetrySuccess,
  DisbursementSuccess,
  DisburserTransferred,
  MaxDepositAmountSet,
  MaxTransferAmountPerDaySet,
  MinDepositAmountSet,
  OwnershipTransferred,
  Paused,
  TokenSupported,
  Unpaused
} from "../generated/schema"

export function handleAssetBalanceWithdrawn(
  event: AssetBalanceWithdrawnEvent
): void {
  let entity = new AssetBalanceWithdrawn(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.token = event.params.token
  entity.owner = event.params.owner
  entity.balance = event.params.balance

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleAssetReceived(event: AssetReceivedEvent): void {
  let entity = new AssetReceived(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.token = event.params.token
  entity.sourceChainId = event.params.sourceChainId
  entity.toChainId = event.params.toChainId
  entity.depositId = event.params.depositId
  entity.emitter = event.params.emitter
  entity.amount = event.params.amount

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleDisbursementFailed(event: DisbursementFailedEvent): void {
  let entity = new DisbursementFailed(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.depositId = event.params.depositId
  entity.to = event.params.to
  entity.amount = event.params.amount
  entity.sourceChainId = event.params.sourceChainId

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleDisbursementRetrySuccess(
  event: DisbursementRetrySuccessEvent
): void {
  let entity = new DisbursementRetrySuccess(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.depositId = event.params.depositId
  entity.to = event.params.to
  entity.amount = event.params.amount
  entity.sourceChainId = event.params.sourceChainId

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleDisbursementSuccess(
  event: DisbursementSuccessEvent
): void {
  let entity = new DisbursementSuccess(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.depositId = event.params.depositId
  entity.to = event.params.to
  entity.token = event.params.token
  entity.amount = event.params.amount
  entity.sourceChainId = event.params.sourceChainId

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleDisburserTransferred(
  event: DisburserTransferredEvent
): void {
  let entity = new DisburserTransferred(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.newDisburser = event.params.newDisburser

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleMaxDepositAmountSet(
  event: MaxDepositAmountSetEvent
): void {
  let entity = new MaxDepositAmountSet(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.token = event.params.token
  entity.toChainId = event.params.toChainId
  entity.previousAmount = event.params.previousAmount
  entity.newAmount = event.params.newAmount

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleMaxTransferAmountPerDaySet(
  event: MaxTransferAmountPerDaySetEvent
): void {
  let entity = new MaxTransferAmountPerDaySet(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.token = event.params.token
  entity.toChainId = event.params.toChainId
  entity.previousAmount = event.params.previousAmount
  entity.newAmount = event.params.newAmount

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleMinDepositAmountSet(
  event: MinDepositAmountSetEvent
): void {
  let entity = new MinDepositAmountSet(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.token = event.params.token
  entity.toChainId = event.params.toChainId
  entity.previousAmount = event.params.previousAmount
  entity.newAmount = event.params.newAmount

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleOwnershipTransferred(
  event: OwnershipTransferredEvent
): void {
  let entity = new OwnershipTransferred(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.newOwner = event.params.newOwner

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handlePaused(event: PausedEvent): void {
  let entity = new Paused(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.account = event.params.account

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleTokenSupported(event: TokenSupportedEvent): void {
  let entity = new TokenSupported(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.token = event.params.token
  entity.toChainId = event.params.toChainId
  entity.supported = event.params.supported

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleUnpaused(event: UnpausedEvent): void {
  let entity = new Unpaused(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.account = event.params.account

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}
