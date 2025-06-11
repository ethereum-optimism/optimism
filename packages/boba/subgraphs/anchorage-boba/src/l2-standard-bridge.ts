import {
  DepositFinalized as DepositFinalizedEvent,
  ERC20BridgeFinalized as ERC20BridgeFinalizedEvent,
  ERC20BridgeInitiated as ERC20BridgeInitiatedEvent,
  ETHBridgeFinalized as ETHBridgeFinalizedEvent,
  ETHBridgeInitiated as ETHBridgeInitiatedEvent,
  Initialized as InitializedEvent,
  WithdrawalInitiated as WithdrawalInitiatedEvent
} from "../generated/L2StandardBridge/L2StandardBridge"
import {
  DepositFinalized,
  ERC20BridgeFinalized,
  ERC20BridgeInitiated,
  ETHBridgeFinalized,
  ETHBridgeInitiated,
  Initialized,
  WithdrawalInitiated
} from "../generated/schema"
import {
  MessagePassed as MessagePassedEvent,
  WithdrawerBalanceBurnt as WithdrawerBalanceBurntEvent
} from "../generated/L2ToL1MessagePasser/L2ToL1MessagePasser"
import { MessagePassed, WithdrawerBalanceBurnt } from "../generated/schema"

export function handleMessagePassed(event: MessagePassedEvent): void {
  let entity = new MessagePassed(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.nonce = event.params.nonce
  entity.sender = event.params.sender
  entity.target = event.params.target
  entity.value = event.params.value
  entity.gasLimit = event.params.gasLimit
  entity.data = event.params.data
  entity.withdrawalHash = event.params.withdrawalHash

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleWithdrawerBalanceBurnt(
  event: WithdrawerBalanceBurntEvent
): void {
  let entity = new WithdrawerBalanceBurnt(
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.amount = event.params.amount

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}


export function handleDepositFinalized(event: DepositFinalizedEvent): void {
  let entity = new DepositFinalized(
    event.transaction.hash.concatI32(event.logIndex.toI32())
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

export function handleERC20BridgeFinalized(
  event: ERC20BridgeFinalizedEvent
): void {
  let entity = new ERC20BridgeFinalized(
    event.transaction.hash.concatI32(event.logIndex.toI32())
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
  event: ERC20BridgeInitiatedEvent
): void {
  let entity = new ERC20BridgeInitiated(
    event.transaction.hash.concatI32(event.logIndex.toI32())
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

export function handleETHBridgeFinalized(event: ETHBridgeFinalizedEvent): void {
  let entity = new ETHBridgeFinalized(
    event.transaction.hash.concatI32(event.logIndex.toI32())
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
    event.transaction.hash.concatI32(event.logIndex.toI32())
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
    event.transaction.hash.concatI32(event.logIndex.toI32())
  )
  entity.version = event.params.version

  entity.blockNumber = event.block.number
  entity.blockTimestamp = event.block.timestamp
  entity.transactionHash = event.transaction.hash

  entity.save()
}

export function handleWithdrawalInitiated(
  event: WithdrawalInitiatedEvent
): void {
  let entity = new WithdrawalInitiated(
    event.transaction.hash.concatI32(event.logIndex.toI32())
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
