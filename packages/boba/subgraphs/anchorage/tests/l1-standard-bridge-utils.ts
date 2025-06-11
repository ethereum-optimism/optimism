import { newMockEvent } from "matchstick-as"
import { ethereum, Address, BigInt, Bytes } from "@graphprotocol/graph-ts"
import {
  ERC20BridgeFinalized,
  ERC20BridgeInitiated,
  ERC20DepositInitiated,
  ERC20WithdrawalFinalized,
  ETHBridgeFinalized,
  ETHBridgeInitiated,
  ETHDepositInitiated,
  ETHWithdrawalFinalized,
  Initialized
} from "../generated/L1StandardBridge/L1StandardBridge"

export function createERC20BridgeFinalizedEvent(
  localToken: Address,
  remoteToken: Address,
  from: Address,
  to: Address,
  amount: BigInt,
  extraData: Bytes
): ERC20BridgeFinalized {
  let erc20BridgeFinalizedEvent = changetype<ERC20BridgeFinalized>(
    newMockEvent()
  )

  erc20BridgeFinalizedEvent.parameters = new Array()

  erc20BridgeFinalizedEvent.parameters.push(
    new ethereum.EventParam(
      "localToken",
      ethereum.Value.fromAddress(localToken)
    )
  )
  erc20BridgeFinalizedEvent.parameters.push(
    new ethereum.EventParam(
      "remoteToken",
      ethereum.Value.fromAddress(remoteToken)
    )
  )
  erc20BridgeFinalizedEvent.parameters.push(
    new ethereum.EventParam("from", ethereum.Value.fromAddress(from))
  )
  erc20BridgeFinalizedEvent.parameters.push(
    new ethereum.EventParam("to", ethereum.Value.fromAddress(to))
  )
  erc20BridgeFinalizedEvent.parameters.push(
    new ethereum.EventParam("amount", ethereum.Value.fromUnsignedBigInt(amount))
  )
  erc20BridgeFinalizedEvent.parameters.push(
    new ethereum.EventParam("extraData", ethereum.Value.fromBytes(extraData))
  )

  return erc20BridgeFinalizedEvent
}

export function createERC20BridgeInitiatedEvent(
  localToken: Address,
  remoteToken: Address,
  from: Address,
  to: Address,
  amount: BigInt,
  extraData: Bytes
): ERC20BridgeInitiated {
  let erc20BridgeInitiatedEvent = changetype<ERC20BridgeInitiated>(
    newMockEvent()
  )

  erc20BridgeInitiatedEvent.parameters = new Array()

  erc20BridgeInitiatedEvent.parameters.push(
    new ethereum.EventParam(
      "localToken",
      ethereum.Value.fromAddress(localToken)
    )
  )
  erc20BridgeInitiatedEvent.parameters.push(
    new ethereum.EventParam(
      "remoteToken",
      ethereum.Value.fromAddress(remoteToken)
    )
  )
  erc20BridgeInitiatedEvent.parameters.push(
    new ethereum.EventParam("from", ethereum.Value.fromAddress(from))
  )
  erc20BridgeInitiatedEvent.parameters.push(
    new ethereum.EventParam("to", ethereum.Value.fromAddress(to))
  )
  erc20BridgeInitiatedEvent.parameters.push(
    new ethereum.EventParam("amount", ethereum.Value.fromUnsignedBigInt(amount))
  )
  erc20BridgeInitiatedEvent.parameters.push(
    new ethereum.EventParam("extraData", ethereum.Value.fromBytes(extraData))
  )

  return erc20BridgeInitiatedEvent
}

export function createERC20DepositInitiatedEvent(
  l1Token: Address,
  l2Token: Address,
  from: Address,
  to: Address,
  amount: BigInt,
  extraData: Bytes
): ERC20DepositInitiated {
  let erc20DepositInitiatedEvent = changetype<ERC20DepositInitiated>(
    newMockEvent()
  )

  erc20DepositInitiatedEvent.parameters = new Array()

  erc20DepositInitiatedEvent.parameters.push(
    new ethereum.EventParam("l1Token", ethereum.Value.fromAddress(l1Token))
  )
  erc20DepositInitiatedEvent.parameters.push(
    new ethereum.EventParam("l2Token", ethereum.Value.fromAddress(l2Token))
  )
  erc20DepositInitiatedEvent.parameters.push(
    new ethereum.EventParam("from", ethereum.Value.fromAddress(from))
  )
  erc20DepositInitiatedEvent.parameters.push(
    new ethereum.EventParam("to", ethereum.Value.fromAddress(to))
  )
  erc20DepositInitiatedEvent.parameters.push(
    new ethereum.EventParam("amount", ethereum.Value.fromUnsignedBigInt(amount))
  )
  erc20DepositInitiatedEvent.parameters.push(
    new ethereum.EventParam("extraData", ethereum.Value.fromBytes(extraData))
  )

  return erc20DepositInitiatedEvent
}

export function createERC20WithdrawalFinalizedEvent(
  l1Token: Address,
  l2Token: Address,
  from: Address,
  to: Address,
  amount: BigInt,
  extraData: Bytes
): ERC20WithdrawalFinalized {
  let erc20WithdrawalFinalizedEvent = changetype<ERC20WithdrawalFinalized>(
    newMockEvent()
  )

  erc20WithdrawalFinalizedEvent.parameters = new Array()

  erc20WithdrawalFinalizedEvent.parameters.push(
    new ethereum.EventParam("l1Token", ethereum.Value.fromAddress(l1Token))
  )
  erc20WithdrawalFinalizedEvent.parameters.push(
    new ethereum.EventParam("l2Token", ethereum.Value.fromAddress(l2Token))
  )
  erc20WithdrawalFinalizedEvent.parameters.push(
    new ethereum.EventParam("from", ethereum.Value.fromAddress(from))
  )
  erc20WithdrawalFinalizedEvent.parameters.push(
    new ethereum.EventParam("to", ethereum.Value.fromAddress(to))
  )
  erc20WithdrawalFinalizedEvent.parameters.push(
    new ethereum.EventParam("amount", ethereum.Value.fromUnsignedBigInt(amount))
  )
  erc20WithdrawalFinalizedEvent.parameters.push(
    new ethereum.EventParam("extraData", ethereum.Value.fromBytes(extraData))
  )

  return erc20WithdrawalFinalizedEvent
}

export function createETHBridgeFinalizedEvent(
  from: Address,
  to: Address,
  amount: BigInt,
  extraData: Bytes
): ETHBridgeFinalized {
  let ethBridgeFinalizedEvent = changetype<ETHBridgeFinalized>(newMockEvent())

  ethBridgeFinalizedEvent.parameters = new Array()

  ethBridgeFinalizedEvent.parameters.push(
    new ethereum.EventParam("from", ethereum.Value.fromAddress(from))
  )
  ethBridgeFinalizedEvent.parameters.push(
    new ethereum.EventParam("to", ethereum.Value.fromAddress(to))
  )
  ethBridgeFinalizedEvent.parameters.push(
    new ethereum.EventParam("amount", ethereum.Value.fromUnsignedBigInt(amount))
  )
  ethBridgeFinalizedEvent.parameters.push(
    new ethereum.EventParam("extraData", ethereum.Value.fromBytes(extraData))
  )

  return ethBridgeFinalizedEvent
}

export function createETHBridgeInitiatedEvent(
  from: Address,
  to: Address,
  amount: BigInt,
  extraData: Bytes
): ETHBridgeInitiated {
  let ethBridgeInitiatedEvent = changetype<ETHBridgeInitiated>(newMockEvent())

  ethBridgeInitiatedEvent.parameters = new Array()

  ethBridgeInitiatedEvent.parameters.push(
    new ethereum.EventParam("from", ethereum.Value.fromAddress(from))
  )
  ethBridgeInitiatedEvent.parameters.push(
    new ethereum.EventParam("to", ethereum.Value.fromAddress(to))
  )
  ethBridgeInitiatedEvent.parameters.push(
    new ethereum.EventParam("amount", ethereum.Value.fromUnsignedBigInt(amount))
  )
  ethBridgeInitiatedEvent.parameters.push(
    new ethereum.EventParam("extraData", ethereum.Value.fromBytes(extraData))
  )

  return ethBridgeInitiatedEvent
}

export function createETHDepositInitiatedEvent(
  from: Address,
  to: Address,
  amount: BigInt,
  extraData: Bytes
): ETHDepositInitiated {
  let ethDepositInitiatedEvent = changetype<ETHDepositInitiated>(newMockEvent())

  ethDepositInitiatedEvent.parameters = new Array()

  ethDepositInitiatedEvent.parameters.push(
    new ethereum.EventParam("from", ethereum.Value.fromAddress(from))
  )
  ethDepositInitiatedEvent.parameters.push(
    new ethereum.EventParam("to", ethereum.Value.fromAddress(to))
  )
  ethDepositInitiatedEvent.parameters.push(
    new ethereum.EventParam("amount", ethereum.Value.fromUnsignedBigInt(amount))
  )
  ethDepositInitiatedEvent.parameters.push(
    new ethereum.EventParam("extraData", ethereum.Value.fromBytes(extraData))
  )

  return ethDepositInitiatedEvent
}

export function createETHWithdrawalFinalizedEvent(
  from: Address,
  to: Address,
  amount: BigInt,
  extraData: Bytes
): ETHWithdrawalFinalized {
  let ethWithdrawalFinalizedEvent = changetype<ETHWithdrawalFinalized>(
    newMockEvent()
  )

  ethWithdrawalFinalizedEvent.parameters = new Array()

  ethWithdrawalFinalizedEvent.parameters.push(
    new ethereum.EventParam("from", ethereum.Value.fromAddress(from))
  )
  ethWithdrawalFinalizedEvent.parameters.push(
    new ethereum.EventParam("to", ethereum.Value.fromAddress(to))
  )
  ethWithdrawalFinalizedEvent.parameters.push(
    new ethereum.EventParam("amount", ethereum.Value.fromUnsignedBigInt(amount))
  )
  ethWithdrawalFinalizedEvent.parameters.push(
    new ethereum.EventParam("extraData", ethereum.Value.fromBytes(extraData))
  )

  return ethWithdrawalFinalizedEvent
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
