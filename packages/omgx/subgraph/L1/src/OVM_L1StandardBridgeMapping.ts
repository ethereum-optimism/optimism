import {
  ERC20DepositInitiated,
  ERC20WithdrawalFinalized,
  ETHDepositInitiated,
  ETHWithdrawalFinalized
} from '../generated/OVM_L1StandardBridge/OVM_L1StandardBridge'
import {
  SBERC20DepositInitiated,
  SBERC20WithdrawalFinalized,
  SBETHDepositInitiated,
  SBETHWithdrawalFinalized
} from '../generated/schema'

export function handleSBERC20DepositInitiated(event: ERC20DepositInitiated): void {
  let id = event.transaction.hash.toHex()
  let erc20DepositInitiated = new SBERC20DepositInitiated(id)
  erc20DepositInitiated.id = id
  erc20DepositInitiated.l1Token = event.params._l1Token
  erc20DepositInitiated.l2Token = event.params._l2Token
  erc20DepositInitiated.from = event.params._from
  erc20DepositInitiated.l1Token = event.params._l1Token
  erc20DepositInitiated.to = event.params._to
  erc20DepositInitiated.amount = event.params._amount.toString()
  erc20DepositInitiated.data = event.params._data
  erc20DepositInitiated.save()
}

export function handleSBERC20WithdrawalFinalized(event: ERC20WithdrawalFinalized): void {
  let id = event.transaction.hash.toHex()
  let erc20WithdrawalFinalized = new SBERC20WithdrawalFinalized(id)
  erc20WithdrawalFinalized.id = id
  erc20WithdrawalFinalized.l1Token = event.params._l1Token
  erc20WithdrawalFinalized.l2Token = event.params._l2Token
  erc20WithdrawalFinalized.from = event.params._from
  erc20WithdrawalFinalized.l1Token = event.params._l1Token
  erc20WithdrawalFinalized.to = event.params._to
  erc20WithdrawalFinalized.amount = event.params._amount.toString()
  erc20WithdrawalFinalized.data = event.params._data
  erc20WithdrawalFinalized.save()
}

export function handleSBETHDepositInitiated(event: ETHDepositInitiated): void {
  let id = event.transaction.hash.toHex()
  let ethDepositInitiated = new SBETHDepositInitiated(id)
  ethDepositInitiated.id = id
  ethDepositInitiated.from = event.params._from
  ethDepositInitiated.to = event.params._to
  ethDepositInitiated.amount = event.params._amount.toString()
  ethDepositInitiated.data = event.params._data
  ethDepositInitiated.save()
}

export function handleSBETHWithdrawalFinalized(event: ETHWithdrawalFinalized): void {
  let id = event.transaction.hash.toHex()
  let ethWithdrawalFinalized = new SBETHWithdrawalFinalized(id)
  ethWithdrawalFinalized.id = id
  ethWithdrawalFinalized.from = event.params._from
  ethWithdrawalFinalized.to = event.params._to
  ethWithdrawalFinalized.amount = event.params._amount.toString()
  ethWithdrawalFinalized.data = event.params._data
  ethWithdrawalFinalized.save()
}
