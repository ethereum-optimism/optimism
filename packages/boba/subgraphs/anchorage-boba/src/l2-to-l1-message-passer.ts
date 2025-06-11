import { BigInt } from "@graphprotocol/graph-ts"
import {
  MessagePassed as MessagePassedEvent,
  WithdrawerBalanceBurnt as WithdrawerBalanceBurntEvent
} from "../generated/L2ToL1MessagePasser/L2ToL1MessagePasser"
import { ETHBridgeInitiated, MessagePassed, WithdrawerBalanceBurnt } from "../generated/schema"

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

  if (event.transaction.value.notEqual(BigInt.fromI32(0))) {
    let bridgeEntity = new ETHBridgeInitiated(event.transaction.hash.concatI32(event.logIndex.toI32()))
    bridgeEntity.amount = event.transaction.value
    bridgeEntity.from = event.transaction.from
    bridgeEntity.to = event.params.target
    bridgeEntity.blockNumber = event.block.number
    bridgeEntity.blockTimestamp = event.block.timestamp
    bridgeEntity.transactionHash = event.transaction.hash

    bridgeEntity.save()
  }

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
