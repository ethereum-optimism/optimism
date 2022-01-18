import {
  Provider,
  TransactionReceipt,
  TransactionResponse,
} from '@ethersproject/abstract-provider'
import { Signer } from '@ethersproject/abstract-signer'
import { Contract, BigNumber, Overrides } from 'ethers'

/**
 * L1 contract references.
 */
export interface OEL1Contracts {
  AddressManager: Contract
  L1CrossDomainMessenger: Contract
  L1StandardBridge: Contract
  StateCommitmentChain: Contract
  CanonicalTransactionChain: Contract
  BondManager: Contract
}

/**
 * L2 contract references.
 */
export interface OEL2Contracts {
  L2CrossDomainMessenger: Contract
  L2StandardBridge: Contract
  OVM_L1BlockNumber: Contract
  OVM_L2ToL1MessagePasser: Contract
  OVM_DeployerWhitelist: Contract
  OVM_ETH: Contract
  OVM_GasPriceOracle: Contract
  OVM_SequencerFeeVault: Contract
  WETH: Contract
}

/**
 * Represents Optimism contracts, assumed to be connected to their appropriate
 * providers and addresses.
 */
export interface OEContracts {
  l1: OEL1Contracts
  l2: OEL2Contracts
}

/**
 * Convenience type for something that looks like the L1 OE contract interface but could be
 * addresses instead of actual contract objects.
 */
export type OEL1ContractsLike = {
  [K in keyof OEL1Contracts]: AddressLike
}

/**
 * Convenience type for something that looks like the L2 OE contract interface but could be
 * addresses instead of actual contract objects.
 */
export type OEL2ContractsLike = {
  [K in keyof OEL2Contracts]: AddressLike
}

/**
 * Convenience type for something that looks like the OE contract interface but could be
 * addresses instead of actual contract objects.
 */
export interface OEContractsLike {
  l1: OEL1ContractsLike
  l2: OEL2ContractsLike
}

/**
 * Represents list of custom bridges.
 */
export interface CustomBridges {
  l1: {
    [name: string]: Contract
  }
  l2: {
    [name: string]: Contract
  }
}

/**
 * Something that looks like the list of custom bridges.
 */
export interface CustomBridgesLike {
  l1: {
    [K in keyof CustomBridges['l1']]: AddressLike
  }
  l2: {
    [K in keyof CustomBridges['l2']]: AddressLike
  }
}

/**
 * Enum describing the status of a message.
 */
export enum MessageStatus {
  /**
   * Message is an L1 to L2 message and has not been processed by the L2.
   */
  UNCONFIRMED_L1_TO_L2_MESSAGE,

  /**
   * Message is an L1 to L2 message and the transaction to execute the message failed.
   * When this status is returned, you will need to resend the L1 to L2 message, probably with a
   * higher gas limit.
   */
  FAILED_L1_TO_L2_MESSAGE,

  /**
   * Message is an L2 to L1 message and no state root has been published yet.
   */
  STATE_ROOT_NOT_PUBLISHED,

  /**
   * Message is an L2 to L1 message and awaiting the challenge period.
   */
  IN_CHALLENGE_PERIOD,

  /**
   * Message is ready to be relayed.
   */
  READY_FOR_RELAY,

  /**
   * Message has been relayed.
   */
  RELAYED,
}

/**
 * Enum describing the direction of a message.
 */
export enum MessageDirection {
  L1_TO_L2,
  L2_TO_L1,
}

/**
 * Partial message that needs to be signed and executed by a specific signer.
 */
export interface CrossChainMessageRequest {
  direction: MessageDirection
  target: string
  message: string
  l2GasLimit: NumberLike
}

/**
 * Core components of a cross chain message.
 */
export interface CoreCrossChainMessage {
  sender: string
  target: string
  message: string
  messageNonce: number
}

/**
 * Describes a message that is sent between L1 and L2. Direction determines where the message was
 * sent from and where it's being sent to.
 */
export interface CrossChainMessage extends CoreCrossChainMessage {
  direction: MessageDirection
  logIndex: number
  blockNumber: number
  transactionHash: string
}

/**
 * Describes a token withdrawal or deposit, along with the underlying raw cross chain message
 * behind the deposit or withdrawal.
 */
export interface TokenBridgeMessage {
  direction: MessageDirection
  from: string
  to: string
  l1Token: string
  l2Token: string
  amount: BigNumber
  data: string
  logIndex: number
  blockNumber: number
  transactionHash: string
}

/**
 * Enum describing the status of a CrossDomainMessage message receipt.
 */
export enum MessageReceiptStatus {
  RELAYED_FAILED,
  RELAYED_SUCCEEDED,
}

/**
 * CrossDomainMessage receipt.
 */
export interface MessageReceipt {
  receiptStatus: MessageReceiptStatus
  transactionReceipt: TransactionReceipt
}

/**
 * Header for a state root batch.
 */
export interface StateRootBatchHeader {
  batchIndex: BigNumber
  batchRoot: string
  batchSize: BigNumber
  prevTotalElements: BigNumber
  extraData: string
}

/**
 * State root batch, including header and actual state roots.
 */
export interface StateRootBatch {
  header: StateRootBatchHeader
  stateRoots: string[]
}

/**
 * Extended Ethers overrides object with an l2GasLimit field.
 * Only meant to be used for L1 to L2 messages, since L2 to L1 messages don't have a specified gas
 * limit field (gas used depends on the amount of gas provided).
 */
export type L1ToL2Overrides = Overrides & {
  l2GasLimit: NumberLike
}

/**
 * Stuff that can be coerced into a transaction.
 */
export type TransactionLike = string | TransactionReceipt | TransactionResponse

/**
 * Stuff that can be coerced into a message.
 */
export type MessageLike =
  | CrossChainMessage
  | TransactionLike
  | TokenBridgeMessage

/**
 * Stuff that can be coerced into a provider.
 */
export type ProviderLike = string | Provider | any

/**
 * Stuff that can be coerced into a signer.
 */
export type SignerLike = string | Signer

/**
 * Stuff that can be coerced into a signer or provider.
 */
export type SignerOrProviderLike = SignerLike | ProviderLike

/**
 * Stuff that can be coerced into an address.
 */
export type AddressLike = string | Contract

/**
 * Stuff that can be coerced into a number.
 */
export type NumberLike = string | number | BigNumber
