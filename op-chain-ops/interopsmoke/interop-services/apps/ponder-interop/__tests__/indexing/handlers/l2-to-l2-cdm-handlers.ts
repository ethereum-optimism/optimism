import { contracts } from '@eth-optimism/viem'
import type {
  CrossDomainMessage,
  MessageIdentifier,
} from '@eth-optimism/viem/types/interop'
import {
  encodeMessagePayload,
  hashCrossDomainMessage,
} from '@eth-optimism/viem/utils/interop'
import type { Log } from 'viem'
import { hashMessageIdentifier } from '../../../src/utils/hashMessageIdentifier'

// Extracted handler functions for testing

export async function handleSentMessage(event: any, context: any) {
  const cdm: CrossDomainMessage = {
    source: BigInt(context.chain.id),
    destination: event.args.destination,
    nonce: event.args.messageNonce,
    sender: event.args.sender as `0x${string}`,
    target: event.args.target as `0x${string}`,
    message: event.args.message as `0x${string}`,
    log: event.log as Log,
  }

  const messageIdentifier: MessageIdentifier = {
    origin: contracts.l2ToL2CrossDomainMessenger.address,
    chainId: cdm.source,
    logIndex: BigInt(event.log.logIndex),
    blockNumber: event.block.number,
    timestamp: BigInt(event.block.timestamp),
  }

  await context.db.insert('sentMessages').values({
    messageIdentifierHash: hashMessageIdentifier(messageIdentifier),
    messageHash: hashCrossDomainMessage(cdm),

    // message
    source: cdm.source,
    destination: cdm.destination,
    nonce: cdm.nonce,
    sender: cdm.sender,
    target: cdm.target,
    message: cdm.message,

    // log fields
    logIndex: BigInt(event.log.logIndex),
    logPayload: encodeMessagePayload(event.log as Log),

    // general info
    timestamp: event.block.timestamp,
    blockNumber: event.block.number,
    transactionHash: event.transaction.hash,
    txOrigin: event.transaction.from,
  })
}

export async function handleRelayedMessageV1(event: any, context: any) {
  await context.db.insert('relayedMessages').values({
    messageHash: event.args.messageHash,

    // metadata
    relayer: event.transaction.from,

    // log fields
    logIndex: BigInt(event.log.logIndex),
    logPayload: encodeMessagePayload(event.log as Log),

    // general info
    timestamp: event.block.timestamp,
    blockNumber: event.block.number,
    transactionHash: event.transaction.hash,
  })
}

export async function handleRelayedMessageV2(event: any, context: any) {
  await context.db.insert('relayedMessages').values({
    messageHash: event.args.messageHash,

    // metadata
    relayer: event.transaction.from,

    // log fields
    logIndex: BigInt(event.log.logIndex),
    logPayload: encodeMessagePayload(event.log as Log),

    // general info
    timestamp: event.block.timestamp,
    blockNumber: event.block.number,
    transactionHash: event.transaction.hash,
  })
}