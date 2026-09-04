import { contracts } from '@eth-optimism/viem'
import type {
  CrossDomainMessage,
  MessageIdentifier,
} from '@eth-optimism/viem/types/interop'
import {
  encodeMessagePayload,
  hashCrossDomainMessage,
} from '@eth-optimism/viem/utils/interop'
import { ponder } from 'ponder:registry'
import {
  callbacks,
  deposits,
  gasTankClaimedMessages,
  gasTankFlaggedMessages,
  gasTankGasProviders,
  gasTankPendingWithdrawals,
  gasTankRelayedMessageReceipts,
  promises,
  relayedMessages,
  ResolutionTransferred,
  sentMessages,
} from 'ponder:schema'
import { type Log } from 'viem'

import { hashMessageIdentifier } from '@/utils/hashMessageIdentifier.js'

ponder.on('L2ToL2CDM:SentMessage', async ({ event, context }) => {
  const cdm = {
    source: BigInt(context.chain.id),
    destination: event.args.destination,
    nonce: event.args.messageNonce,
    sender: event.args.sender,
    target: event.args.target,
    message: event.args.message,
    log: event.log as Log,
  } satisfies CrossDomainMessage

  const messageIdentifier: MessageIdentifier = {
    origin: contracts.l2ToL2CrossDomainMessenger.address,
    chainId: cdm.source,
    logIndex: BigInt(event.log.logIndex),
    blockNumber: event.block.number,
    timestamp: BigInt(event.block.timestamp),
  }

  await context.db.insert(sentMessages).values({
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
})

ponder.on(
  'L2ToL2CDM:RelayedMessage(uint256 indexed source, uint256 indexed messageNonce, bytes32 indexed messageHash)',
  async ({ event, context }) => {
    await context.db.insert(relayedMessages).values({
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
  },
)

ponder.on(
  'L2ToL2CDM:RelayedMessage(uint256 indexed source, uint256 indexed messageNonce, bytes32 indexed messageHash, bytes32 returnDataHash)',
  async ({ event, context }) => {
    await context.db.insert(relayedMessages).values({
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
  },
)

ponder.on('GasTank:Deposit', async ({ event, context }) => {
  await context.db
    .insert(gasTankGasProviders)
    .values({
      chainId: BigInt(context.chain.id),
      address: event.args.depositor,
      balance: event.args.amount,
      lastUpdatedAt: event.block.timestamp,
    })
    .onConflictDoUpdate((row) => ({
      balance: row.balance + event.args.amount,
      lastUpdatedAt: event.block.timestamp,
    }))
})

ponder.on('GasTank:WithdrawalInitiated', async ({ event, context }) => {
  await context.db
    .insert(gasTankPendingWithdrawals)
    .values({
      chainId: BigInt(context.chain.id),
      address: event.args.from,
      amount: event.args.amount,
      initiatedAt: event.block.timestamp,
    })
    .onConflictDoUpdate({
      amount: event.args.amount,
      initiatedAt: event.block.timestamp,
    })
})

ponder.on('GasTank:WithdrawalFinalized', async ({ event, context }) => {
  await context.db.delete(gasTankPendingWithdrawals, {
    chainId: BigInt(context.chain.id),
    address: event.args.from,
  })

  await context.db
    .update(gasTankGasProviders, {
      chainId: BigInt(context.chain.id),
      address: event.args.from,
    })
    .set((row) => ({
      balance: row.balance - event.args.amount,
      lastUpdatedAt: event.block.timestamp,
    }))
})

ponder.on('GasTank:Flagged', async ({ event, context }) => {
  await context.db
    .insert(gasTankFlaggedMessages)
    .values({
      chainId: BigInt(context.chain.id),
      gasProvider: event.args.gasProvider,
      originMessageHash: event.args.originMsgHash,
      flaggedAt: event.block.timestamp,
    })
    .onConflictDoNothing()
})

ponder.on('GasTank:Claimed', async ({ event, context }) => {
  await context.db
    .update(gasTankGasProviders, {
      chainId: BigInt(context.chain.id),
      address: event.args.gasProvider,
    })
    .set((row) => ({
      balance: row.balance - event.args.amount,
      lastUpdatedAt: event.block.timestamp,
    }))

  await context.db.insert(gasTankClaimedMessages).values({
    originMessageHash: event.args.originMsgHash,
    chainId: BigInt(context.chain.id),
    relayer: event.args.relayer,
    gasProvider: event.args.gasProvider,
    amountClaimed: event.args.amount,
    claimedAt: event.block.timestamp,
  })
})

ponder.on('GasTank:RelayedMessageGasReceipt', async ({ event, context }) => {
  await context.db.insert(gasTankRelayedMessageReceipts).values({
    originMessageHash: event.args.originMsgHash,
    chainId: BigInt(context.chain.id),
    relayer: event.args.relayer,
    gasCost: event.args.gasCost,
    destinationMessageHashes: [...event.args.destinationMessageHashes],
    relayedAt: event.block.timestamp,
  })
})

// MostBasicRelayDeposit event handler
ponder.on('MostBasicRelayDeposit:Deposited', async ({ event, context }) => {
  await context.db
    .insert(deposits)
    .values({
      chainId: BigInt(context.chain.id),
      depositor: event.args.depositor,
      balance: event.args.amount,
      lastDepositAt: event.block.timestamp,
      lastDepositBlockNumber: event.block.number,
      lastDepositTransactionHash: event.transaction.hash,
    })
    .onConflictDoUpdate((row) => ({
      balance: row.balance + event.args.amount,
      lastDepositAt: event.block.timestamp,
      lastDepositBlockNumber: event.block.number,
      lastDepositTransactionHash: event.transaction.hash,
    }))
})

// ---------------------------------------------------------------------------
// Promise + Callback indexing. Feeds the relayer's /promises/pending +
// /promises/unshared-resolved endpoints and the viewer's /promise-chains.
// ---------------------------------------------------------------------------

// Twin.executeCallback selector — when a callback routes through a Twin, the
// real target/selector live in the Twin's `dispatches` mapping.
const EXECUTE_CALLBACK_SELECTOR = '0x3bb52210'

ponder.on('Callback:CallbackRegistered', async ({ event, context }) => {
  const callbackPromiseId = event.args.callbackPromiseId

  // The event carries only ids + type; target/selector live in the callbacks
  // mapping. Read them now — Callback.resolve() deletes the entry after it runs.
  let target: `0x${string}` = event.log.address
  let selector: `0x${string}` | null = null
  try {
    const callbackData = await context.client.readContract({
      address: event.log.address,
      abi: [
        {
          type: 'function',
          name: 'getCallback',
          stateMutability: 'view',
          inputs: [{ name: 'callbackPromiseId', type: 'bytes32' }],
          outputs: [
            {
              name: 'callbackData',
              type: 'tuple',
              components: [
                { name: 'parentPromiseId', type: 'bytes32' },
                { name: 'target', type: 'address' },
                { name: 'selector', type: 'bytes4' },
                { name: 'callbackType', type: 'uint8' },
                { name: 'registrant', type: 'address' },
                { name: 'sourceChain', type: 'uint256' },
              ],
            },
          ],
        },
      ] as const,
      functionName: 'getCallback',
      args: [callbackPromiseId],
    })
    target = callbackData.target
    selector = callbackData.selector
  } catch (err) {
    console.warn(`Failed to read callback data for ${callbackPromiseId}:`, err)
  }

  // Twin dispatch: resolve the real target/selector behind executeCallback.
  let realTarget: `0x${string}` | null = null
  let realSelector: `0x${string}` | null = null
  let isDelegateScript: boolean | null = null
  if (selector === EXECUTE_CALLBACK_SELECTOR) {
    try {
      const dispatch = await context.client.readContract({
        address: target, // the Twin
        abi: [
          {
            type: 'function',
            name: 'dispatches',
            stateMutability: 'view',
            inputs: [{ name: 'callbackPromiseId', type: 'bytes32' }],
            outputs: [
              { name: 'target', type: 'address' },
              { name: 'selector', type: 'bytes4' },
              { name: 'delegateScript', type: 'bool' },
            ],
          },
        ] as const,
        functionName: 'dispatches',
        args: [callbackPromiseId],
      })
      realTarget = dispatch[0]
      realSelector = dispatch[1]
      isDelegateScript = dispatch[2]
    } catch (err) {
      console.warn(
        `Failed to read Twin dispatch for ${callbackPromiseId}:`,
        err,
      )
    }
  }

  await context.db.insert(callbacks).values({
    callbackPromiseId,
    parentPromiseId: event.args.parentPromiseId,
    chainId: BigInt(context.chain.id),
    target,
    selector,
    callbackType: Number(event.args.callbackType),
    realTarget,
    realSelector,
    isDelegateScript,
    timestamp: event.block.timestamp,
    blockNumber: event.block.number,
    transactionHash: event.transaction.hash,
  })
})

// PromiseCreated fires both on create() and on receiveResolverTransfer()
// (transfer arrival). PK is (chainId, promiseId) so different chains get
// separate rows; on conflict (re-index) update the resolver.
ponder.on('Promise:PromiseCreated', async ({ event, context }) => {
  const chainId = BigInt(context.chain.id)
  await context.db
    .insert(promises)
    .values({
      promiseId: event.args.promiseId,
      chainId,
      resolver: event.args.resolver,
      status: 'pending',
      createdAt: event.block.timestamp,
      createdBlockNumber: event.block.number,
      createdTransactionHash: event.transaction.hash,
    })
    .onConflictDoUpdate({
      resolver: event.args.resolver,
      status: 'pending',
      createdAt: event.block.timestamp,
      createdBlockNumber: event.block.number,
      createdTransactionHash: event.transaction.hash,
    })
})

// PromiseResolved fires from resolve() on the local chain AND from
// receiveSharedPromise() on destination chains. The destination may never have
// seen PromiseCreated for this promise, so we upsert. A resolved row appearing
// on the destination chain is exactly the signal the relayer uses to drop that
// chain from `unshared-resolved`.
ponder.on('Promise:PromiseResolved', async ({ event, context }) => {
  const chainId = BigInt(context.chain.id)
  const returnData = event.args.returnData ?? null
  const resolvedBy = event.transaction.from
  await context.db
    .insert(promises)
    .values({
      promiseId: event.args.promiseId,
      chainId,
      resolver: '0x0000000000000000000000000000000000000000',
      status: 'resolved',
      createdAt: event.block.timestamp,
      createdBlockNumber: event.block.number,
      createdTransactionHash: event.transaction.hash,
      resolvedAt: event.block.timestamp,
      resolvedBlockNumber: event.block.number,
      resolvedTransactionHash: event.transaction.hash,
      returnData,
      resolvedBy,
    })
    .onConflictDoUpdate({
      status: 'resolved',
      resolvedAt: event.block.timestamp,
      resolvedBlockNumber: event.block.number,
      resolvedTransactionHash: event.transaction.hash,
      returnData,
      resolvedBy,
    })
})

// PromiseRejected fires from reject() and from receiveSharedPromise().
ponder.on('Promise:PromiseRejected', async ({ event, context }) => {
  const chainId = BigInt(context.chain.id)
  await context.db
    .insert(promises)
    .values({
      promiseId: event.args.promiseId,
      chainId,
      resolver: '0x0000000000000000000000000000000000000000',
      status: 'rejected',
      createdAt: event.block.timestamp,
      createdBlockNumber: event.block.number,
      createdTransactionHash: event.transaction.hash,
      resolvedAt: event.block.timestamp,
      resolvedBlockNumber: event.block.number,
      resolvedTransactionHash: event.transaction.hash,
    })
    .onConflictDoUpdate({
      status: 'rejected',
      resolvedAt: event.block.timestamp,
      resolvedBlockNumber: event.block.number,
      resolvedTransactionHash: event.transaction.hash,
    })
})

ponder.on('Promise:ResolutionTransferred', async ({ event, context }) => {
  const chainId = BigInt(context.chain.id)
  const promiseId = event.args.promiseId

  // Raw event row — source of truth for the promise's home (destination) chain,
  // used by /promise-chains.
  await context.db.insert(ResolutionTransferred).values({
    promiseId,
    destinationChainId: event.args.destinationChainId,
    resolver: event.args.resolver,
    chainId,
    timestamp: event.block.timestamp,
    blockNumber: event.block.number,
    transactionHash: event.transaction.hash,
  })

  // Mark transferred on the origin chain — but only from 'pending'. If the
  // destination's PromiseResolved/PromiseRejected was already indexed
  // (cross-chain ordering), don't overwrite the terminal status. Upsert so a
  // row always exists even if PromiseCreated hasn't been seen here yet.
  await context.db
    .insert(promises)
    .values({
      promiseId,
      chainId,
      resolver: event.args.resolver,
      status: 'transferred',
      createdAt: event.block.timestamp,
      createdBlockNumber: event.block.number,
      createdTransactionHash: event.transaction.hash,
      transferredAt: event.block.timestamp,
      transferredBlockNumber: event.block.number,
      transferredTransactionHash: event.transaction.hash,
    })
    .onConflictDoUpdate((row) => ({
      status: row.status === 'pending' ? 'transferred' : row.status,
      transferredAt: event.block.timestamp,
      transferredBlockNumber: event.block.number,
      transferredTransactionHash: event.transaction.hash,
    }))
})
