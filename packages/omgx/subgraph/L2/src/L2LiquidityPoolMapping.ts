import {
  AddLiquidity,
  ClientDepositL2,
  ClientPayL2,
  OwnerRecoverFee,
  WithdrawLiquidity,
  WithdrawReward,
} from '../generated/L2LiquidityPool/L2LiquidityPool'
import {
  LPAddLiquidity,
  LPClientDepositL2,
  LPClientPayL2,
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

export function handleLPClientDepositL2(event: ClientDepositL2): void {
  let id = event.transaction.hash.toHex()
  let clientDepositL2Event = new LPClientDepositL2(id)
  clientDepositL2Event.id = id
  clientDepositL2Event.sender = event.params.sender
  clientDepositL2Event.amount = event.params.receivedAmount.toString()
  clientDepositL2Event.token = event.params.tokenAddress
  clientDepositL2Event.save()
}

export function handleLPClientPayL2(event: ClientPayL2): void {
  let id = event.transaction.hash.toHex()
  let clientPayL2Event = new LPClientPayL2(id)
  clientPayL2Event.id = id
  clientPayL2Event.sender = event.params.sender
  clientPayL2Event.amount = event.params.amount.toString()
  clientPayL2Event.userRewardFee = event.params.userRewardFee.toString()
  clientPayL2Event.ownerRewardFee = event.params.ownerRewardFee.toString()
  clientPayL2Event.totalFee = event.params.totalFee.toString()
  clientPayL2Event.token = event.params.tokenAddress
  clientPayL2Event.save()
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