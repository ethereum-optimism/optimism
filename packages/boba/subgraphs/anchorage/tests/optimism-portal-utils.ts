import { newMockEvent } from "matchstick-as"
import { ethereum, Address, BigInt, Bytes } from "@graphprotocol/graph-ts"
import {
  Initialized,
  TransactionDeposited,
  WithdrawalFinalized,
  WithdrawalProven
} from "../generated/OptimismPortal/OptimismPortal"

export function createInitializedEvent(version: i32): Initialized {
  let initializedEvent = changetype<Initialized>(newMockEvent())

  initializedEvent.parameters = new Array()

  initializedEvent.parameters.push(
    new ethereum.EventParam(
      "version",
      ethereum.Value.fromUnsignedBigInt(BigInt.fromI32(version))
    )
  )

  return initializedEvent
}

export function createTransactionDepositedEvent(
  from: Address,
  to: Address,
  version: BigInt,
  opaqueData: Bytes
): TransactionDeposited {
  let transactionDepositedEvent = changetype<TransactionDeposited>(
    newMockEvent()
  )

  transactionDepositedEvent.parameters = new Array()

  transactionDepositedEvent.parameters.push(
    new ethereum.EventParam("from", ethereum.Value.fromAddress(from))
  )
  transactionDepositedEvent.parameters.push(
    new ethereum.EventParam("to", ethereum.Value.fromAddress(to))
  )
  transactionDepositedEvent.parameters.push(
    new ethereum.EventParam(
      "version",
      ethereum.Value.fromUnsignedBigInt(version)
    )
  )
  transactionDepositedEvent.parameters.push(
    new ethereum.EventParam("opaqueData", ethereum.Value.fromBytes(opaqueData))
  )

  return transactionDepositedEvent
}

export function createWithdrawalFinalizedEvent(
  withdrawalHash: Bytes,
  success: boolean
): WithdrawalFinalized {
  let withdrawalFinalizedEvent = changetype<WithdrawalFinalized>(newMockEvent())

  withdrawalFinalizedEvent.parameters = new Array()

  withdrawalFinalizedEvent.parameters.push(
    new ethereum.EventParam(
      "withdrawalHash",
      ethereum.Value.fromFixedBytes(withdrawalHash)
    )
  )
  withdrawalFinalizedEvent.parameters.push(
    new ethereum.EventParam("success", ethereum.Value.fromBoolean(success))
  )

  return withdrawalFinalizedEvent
}

export function createWithdrawalProvenEvent(
  withdrawalHash: Bytes,
  from: Address,
  to: Address
): WithdrawalProven {
  let withdrawalProvenEvent = changetype<WithdrawalProven>(newMockEvent())

  withdrawalProvenEvent.parameters = new Array()

  withdrawalProvenEvent.parameters.push(
    new ethereum.EventParam(
      "withdrawalHash",
      ethereum.Value.fromFixedBytes(withdrawalHash)
    )
  )
  withdrawalProvenEvent.parameters.push(
    new ethereum.EventParam("from", ethereum.Value.fromAddress(from))
  )
  withdrawalProvenEvent.parameters.push(
    new ethereum.EventParam("to", ethereum.Value.fromAddress(to))
  )

  return withdrawalProvenEvent
}
