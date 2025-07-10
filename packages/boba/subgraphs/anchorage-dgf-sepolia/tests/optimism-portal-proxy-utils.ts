import { newMockEvent } from "matchstick-as"
import { ethereum, Address, BigInt, Bytes } from "@graphprotocol/graph-ts"
import {
  DisputeGameBlacklisted,
  Initialized,
  RespectedGameTypeSet,
  TransactionDeposited,
  WithdrawalFinalized,
  WithdrawalProven,
  WithdrawalProvenExtension1
} from "../generated/OptimismPortal2/OptimismPortal2"

export function createDisputeGameBlacklistedEvent(
  disputeGame: Address
): DisputeGameBlacklisted {
  let disputeGameBlacklistedEvent = changetype<DisputeGameBlacklisted>(
    newMockEvent()
  )

  disputeGameBlacklistedEvent.parameters = new Array()

  disputeGameBlacklistedEvent.parameters.push(
    new ethereum.EventParam(
      "disputeGame",
      ethereum.Value.fromAddress(disputeGame)
    )
  )

  return disputeGameBlacklistedEvent
}

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

export function createRespectedGameTypeSetEvent(
  newGameType: BigInt,
  updatedAt: BigInt
): RespectedGameTypeSet {
  let respectedGameTypeSetEvent = changetype<RespectedGameTypeSet>(
    newMockEvent()
  )

  respectedGameTypeSetEvent.parameters = new Array()

  respectedGameTypeSetEvent.parameters.push(
    new ethereum.EventParam(
      "newGameType",
      ethereum.Value.fromUnsignedBigInt(newGameType)
    )
  )
  respectedGameTypeSetEvent.parameters.push(
    new ethereum.EventParam(
      "updatedAt",
      ethereum.Value.fromUnsignedBigInt(updatedAt)
    )
  )

  return respectedGameTypeSetEvent
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

export function createWithdrawalProvenExtension1Event(
  withdrawalHash: Bytes,
  proofSubmitter: Address
): WithdrawalProvenExtension1 {
  let withdrawalProvenExtension1Event = changetype<WithdrawalProvenExtension1>(
    newMockEvent()
  )

  withdrawalProvenExtension1Event.parameters = new Array()

  withdrawalProvenExtension1Event.parameters.push(
    new ethereum.EventParam(
      "withdrawalHash",
      ethereum.Value.fromFixedBytes(withdrawalHash)
    )
  )
  withdrawalProvenExtension1Event.parameters.push(
    new ethereum.EventParam(
      "proofSubmitter",
      ethereum.Value.fromAddress(proofSubmitter)
    )
  )

  return withdrawalProvenExtension1Event
}
