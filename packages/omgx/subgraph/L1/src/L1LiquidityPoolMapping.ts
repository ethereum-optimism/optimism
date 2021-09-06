import {
  AddLiquidity,
  ClientDepositL1,
  ClientPayL1,
  OwnerRecoverFee,
  WithdrawLiquidity,
  WithdrawReward,
} from '../generated/L1LiquidityPool/L1LiquidityPool'
import {
  LPAddLiquidity,
  LPClientDepositL1,
  LPClientPayL1,
  LPOwnerRecoverFee,
  LPWithdrawLiquidity,
  LPWithdrawReward,
} from '../generated/schema'

export function handleLPAddLiquidity(event: AddLiquidity): void {
  let id = event.transaction.hash.toHex()
  let addLiquidityEvent = new LPAddLiquidity(id)
  addLiquidityEvent.id = id
  addLiquidityEvent.sender = event.params.sender
  addLiquidityEvent.amount = event.params.amount.toString()
  addLiquidityEvent.token = event.params.tokenAddress
  addLiquidityEvent.save()
}

export function handleLPClientDepositL1(event: ClientDepositL1): void {
  let id = event.transaction.hash.toHex()
  let ClientDepositL1Event = new LPClientDepositL1(id)
  ClientDepositL1Event.id = id
  ClientDepositL1Event.sender = event.params.sender
  ClientDepositL1Event.amount = event.params.receivedAmount.toString()
  ClientDepositL1Event.token = event.params.tokenAddress
  ClientDepositL1Event.save()
}

export function handleLPClientPayL1(event: ClientPayL1): void {
  let id = event.transaction.hash.toHex()
  let ClientPayL1Event = new LPClientPayL1(id)
  ClientPayL1Event.id = id
  ClientPayL1Event.sender = event.params.sender
  ClientPayL1Event.amount = event.params.amount.toString()
  ClientPayL1Event.userRewardFee = event.params.userRewardFee.toString()
  ClientPayL1Event.ownerRewardFee = event.params.ownerRewardFee.toString()
  ClientPayL1Event.totalFee = event.params.totalFee.toString()
  ClientPayL1Event.token = event.params.tokenAddress
  ClientPayL1Event.save()
}

export function handleLPOwnerRecoverFee(event: OwnerRecoverFee): void {
  let id = event.transaction.hash.toHex()
  let ownerRecoverFeeEvent = new LPOwnerRecoverFee(id)
  ownerRecoverFeeEvent.id = id
  ownerRecoverFeeEvent.sender = event.params.sender
  ownerRecoverFeeEvent.receiver = event.params.receiver
  ownerRecoverFeeEvent.amount = event.params.amount.toString()
  ownerRecoverFeeEvent.token = event.params.tokenAddress
  ownerRecoverFeeEvent.save()
}

export function handleLPWithdrawLiquidity(event: WithdrawLiquidity): void {
  let id = event.transaction.hash.toHex()
  let withdrawLiquidityEvent = new LPWithdrawLiquidity(id)
  withdrawLiquidityEvent.id = id
  withdrawLiquidityEvent.sender = event.params.sender
  withdrawLiquidityEvent.receiver = event.params.receiver
  withdrawLiquidityEvent.amount = event.params.amount.toString()
  withdrawLiquidityEvent.token = event.params.tokenAddress
  withdrawLiquidityEvent.save()
}

export function handleWithdrawReward(event: WithdrawReward): void {
  let id = event.transaction.hash.toHex()
  let withdrawReward = new LPWithdrawReward(id)
  withdrawReward.id = id
  withdrawReward.sender = event.params.sender
  withdrawReward.receiver = event.params.receiver
  withdrawReward.amount = event.params.amount.toString()
  withdrawReward.token = event.params.tokenAddress
  withdrawReward.save()
}