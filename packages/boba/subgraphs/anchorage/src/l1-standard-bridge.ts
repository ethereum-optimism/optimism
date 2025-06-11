import {
  ERC20BridgeFinalized as ERC20BridgeFinalizedEvent,
  ERC20BridgeInitiated as ERC20BridgeInitiatedEvent,
  ERC20DepositInitiated as ERC20DepositInitiatedEvent,
  ERC20WithdrawalFinalized as ERC20WithdrawalFinalizedEvent,
  ETHBridgeFinalized as ETHBridgeFinalizedEvent,
  ETHBridgeInitiated as ETHBridgeInitiatedEvent,
  ETHDepositInitiated as ETHDepositInitiatedEvent,
  ETHWithdrawalFinalized as ETHWithdrawalFinalizedEvent,
  Initialized as InitializedEvent,
} from "../generated/L1StandardBridge/L1StandardBridge"
import {
  ERC20BridgeFinalized,
  ERC20BridgeInitiated,
  ERC20DepositInitiated,
  ERC20WithdrawalFinalized,
  ETHBridgeFinalized,
  ETHBridgeInitiated,
  ETHDepositInitiated,
  ETHWithdrawalFinalized,
  Initialized,
} from "../generated/schema"

export function handleERC20BridgeFinalized(
  event: ERC20BridgeFinalizedEvent,
): void {
  let entity = new ERC20BridgeFinalized(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.localToken = event.params.localToken
  entity.remoteToken = event.params.remoteToken
  entity.from = event.params.from
  entity.to = event.params.to
  entity.amount = event.params.amount
  entity.extraData = event.params.extraData

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleERC20BridgeInitiated(
  event: ERC20BridgeInitiatedEvent,
): void {
  let entity = new ERC20BridgeInitiated(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.localToken = event.params.localToken
  entity.remoteToken = event.params.remoteToken
  entity.from = event.params.from
  entity.to = event.params.to
  entity.amount = event.params.amount
  entity.extraData = event.params.extraData

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleERC20DepositInitiated(
  event: ERC20DepositInitiatedEvent,
): void {
  let entity = new ERC20DepositInitiated(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.l1Token = event.params.l1Token
  entity.l2Token = event.params.l2Token
  entity.from = event.params.from
  entity.to = event.params.to
  entity.amount = event.params.amount
  entity.extraData = event.params.extraData

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleERC20WithdrawalFinalized(
  event: ERC20WithdrawalFinalizedEvent,
): void {
  let entity = new ERC20WithdrawalFinalized(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.l1Token = event.params.l1Token
  entity.l2Token = event.params.l2Token
  entity.from = event.params.from
  entity.to = event.params.to
  entity.amount = event.params.amount
  entity.extraData = event.params.extraData

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleETHBridgeFinalized(event: ETHBridgeFinalizedEvent): void {
  let entity = new ETHBridgeFinalized(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.from = event.params.from
  entity.to = event.params.to
  entity.amount = event.params.amount
  entity.extraData = event.params.extraData

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleETHBridgeInitiated(event: ETHBridgeInitiatedEvent): void {
  let entity = new ETHBridgeInitiated(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.from = event.params.from
  entity.to = event.params.to
  entity.amount = event.params.amount
  entity.extraData = event.params.extraData

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleETHDepositInitiated(
  event: ETHDepositInitiatedEvent,
): void {
  let entity = new ETHDepositInitiated(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.from = event.params.from
  entity.to = event.params.to
  entity.amount = event.params.amount
  entity.extraData = event.params.extraData

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleETHWithdrawalFinalized(
  event: ETHWithdrawalFinalizedEvent,
): void {
  let entity = new ETHWithdrawalFinalized(
    event.transaction.hash.concatI32(event.logIndex.toI32()),
  )
  entity.from = event.params.from
  entity.to = event.params.to
  entity.amount = event.params.amount
  entity.extraData = event.params.extraData

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
