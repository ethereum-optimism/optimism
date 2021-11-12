import {
  Provider,
  TransactionReceipt,
  TransactionResponse,
} from '@ethersproject/abstract-provider'
import { Signer } from '@ethersproject/abstract-signer'
import { Contract, BigNumber } from 'ethers'

/**
 * Represents Optimistic Ethereum contracts, assumed to be connected to their appropriate
 * providers and addresses.
 */
export interface OEContracts {
  l1: {
    L1CrossDomainMessenger: Contract
    L1StandardBridge: Contract
    StateCommitmentChain: Contract
  }
  l2: {
    L2CrossDomainMessenger: Contract
    L2StandardBridge: Contract
  }
}

/**
 * Enum describing the status of a message.
 */
export enum MessageStatus {
  // Only applies to L1 to L2 messages.
  UNCONFIRMED_L1_TO_L2_MESSAGE,

  // These only apply to L2 to L1 messages.
  STATE_ROOT_NOT_PUBLISHED,
  IN_CHALLENGE_PERIOD,

  // Both of these apply to both types of messages.
  READY_FOR_RELAY,
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
 * Describes a message that is sent between L1 and L2. Direction determines where the message was
 * sent from and where it's being sent to.
 */
export interface CrossChainMessage {
  direction: MessageDirection
  sender: string
  target: string
  message: string
  messageNonce: number
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
  raw: CrossChainMessage
}

/**
 * Enum describing the status of a CrossDomainMessage message receipt.
 */
export enum MessageReceiptStatus {
  RELAYED_SUCCEEDED,
  RELAYED_FAILED,
}

/**
 * CrossDomainMessage receipt.
 */
export interface MessageReceipt {
  messageHash: string
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
 * Different allowable network names.
 * TODO: Maybe we need to add a way to specify a custom network name.
 */
export type NetworkName = 'mainnet' | 'kovan' | 'local' | 'unknown'

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
export type ProviderLike = string | Provider

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
