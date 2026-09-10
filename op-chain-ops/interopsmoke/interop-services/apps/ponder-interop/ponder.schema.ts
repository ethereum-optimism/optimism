// Hack to resolve https://github.com/ponder-sh/ponder/issues/1722
import 'drizzle-orm/pg-core'

import { index, onchainTable, primaryKey } from 'ponder'

export const sentMessages = onchainTable(
  'l2_to_l2_cdm_sent_messages',
  (t) => ({
    // unique identifier
    messageIdentifierHash: t.hex().primaryKey(),

    // parsed message fields
    messageHash: t.hex().notNull(),
    source: t.bigint().notNull(),
    destination: t.bigint().notNull(),
    nonce: t.bigint().notNull(),
    sender: t.hex().notNull(),
    target: t.hex().notNull(),
    message: t.hex().notNull(),

    // log fields
    logIndex: t.bigint().notNull(),
    logPayload: t.hex().notNull(),

    // general fields
    timestamp: t.bigint().notNull(),
    blockNumber: t.bigint().notNull(),
    transactionHash: t.hex().notNull(),
    txOrigin: t.hex().notNull(),
  }),
  (table) => ({
    messageHashIdx: index().on(table.messageHash),
  }),
)

export const relayedMessages = onchainTable(
  'l2_to_l2_cdm_relayed_messages',
  (t) => ({
    // unique identifier
    messageHash: t.hex().primaryKey(),

    // Some unique metadata on the relaying side
    relayer: t.hex().notNull(),

    // log fields
    logIndex: t.bigint().notNull(),
    logPayload: t.hex().notNull(),

    // general fields
    timestamp: t.bigint().notNull(),
    blockNumber: t.bigint().notNull(),
    transactionHash: t.hex().notNull(),
  }),
)

export const gasTankGasProviders = onchainTable(
  'gas_tank_gas_providers',
  (t) => ({
    chainId: t.bigint().notNull(),
    address: t.hex().notNull(),
    balance: t.bigint().notNull(),
    lastUpdatedAt: t.bigint().notNull(),
  }),
  (table) => ({
    pk: primaryKey({ columns: [table.chainId, table.address] }),
  }),
)

export const gasTankPendingWithdrawals = onchainTable(
  'gas_tank_pending_withdrawals',
  (t) => ({
    chainId: t.bigint().notNull(),
    address: t.hex().notNull(),
    amount: t.bigint().notNull(),
    initiatedAt: t.bigint().notNull(),
  }),
  (table) => ({
    pk: primaryKey({ columns: [table.chainId, table.address] }),
  }),
)

export const gasTankFlaggedMessages = onchainTable(
  'gas_tank_flagged_messages',
  (t) => ({
    chainId: t.bigint().notNull(),
    gasProvider: t.hex().notNull(),
    originMessageHash: t.hex().notNull(),
    flaggedAt: t.bigint().notNull(),
  }),
  (table) => ({
    pk: primaryKey({
      columns: [table.chainId, table.gasProvider, table.originMessageHash],
    }),
  }),
)

export const gasTankClaimedMessages = onchainTable(
  'gas_tank_claimed_messages',
  (t) => ({
    originMessageHash: t.hex().notNull(),
    chainId: t.bigint().notNull(),
    relayer: t.hex().notNull(),
    gasProvider: t.hex().notNull(),
    amountClaimed: t.bigint().notNull(),
    claimedAt: t.bigint().notNull(),
  }),
  (table) => ({
    pk: primaryKey({
      columns: [table.chainId, table.originMessageHash],
    }),
  }),
)

export const gasTankRelayedMessageReceipts = onchainTable(
  'gas_tank_relayed_message_receipts',
  (t) => ({
    originMessageHash: t.hex().notNull().primaryKey(),
    chainId: t.bigint().notNull(),
    relayer: t.hex().notNull(),
    gasCost: t.bigint().notNull(),
    destinationMessageHashes: t.hex().array().notNull(),
    relayedAt: t.bigint().notNull(),
  }),
)

export const deposits = onchainTable(
  'relay_deposits',
  (t) => ({
    chainId: t.bigint().notNull(),
    depositor: t.hex().notNull(),
    balance: t.bigint().notNull(),
    lastDepositAt: t.bigint().notNull(),
    lastDepositBlockNumber: t.bigint().notNull(),
    lastDepositTransactionHash: t.hex().notNull(),
  }),
  (table) => ({
    pk: primaryKey({ columns: [table.chainId, table.depositor] }),
    depositorIdx: index().on(table.depositor),
  }),
)

// Cross-chain callbacks registered against a parent promise. Indexed from
// Callback:CallbackRegistered. Keyed (chainId, callbackPromiseId); the
// parentPromiseId index powers the unshared-resolved join.
export const callbacks = onchainTable(
  'callbacks',
  (t) => ({
    callbackPromiseId: t.hex().notNull(),
    parentPromiseId: t.hex().notNull(),
    chainId: t.bigint().notNull(),
    target: t.hex().notNull(),
    // Callback target's 4-byte selector, read from getCallback() at index time.
    selector: t.hex(),
    callbackType: t.integer().notNull(), // 0 = Then, 1 = Catch

    // Twin dispatch: when target.selector is Twin.executeCallback, these hold
    // the real target/selector behind the twin (null for direct callbacks).
    realTarget: t.hex(),
    realSelector: t.hex(),
    isDelegateScript: t.boolean(),

    // general fields
    timestamp: t.bigint().notNull(),
    blockNumber: t.bigint().notNull(),
    transactionHash: t.hex().notNull(),
  }),
  (table) => ({
    pk: primaryKey({ columns: [table.chainId, table.callbackPromiseId] }),
    parentIdx: index().on(table.parentPromiseId),
  }),
)

// Unified promises table for tracking complete lifecycle.
// PK is (chainId, promiseId) because the same promise can exist on multiple
// chains in different states (e.g. 'transferred' on origin, 'resolved' on a
// destination after receiveSharedPromise). The relayer derives "already
// shared to chain X" from a resolved row existing on chain X.
export const promises = onchainTable(
  'promises',
  (t) => ({
    // Global unique promise ID: keccak256(abi.encode(chainId, nonce))
    promiseId: t.hex().notNull(),

    // Chain where this promise exists
    chainId: t.bigint().notNull(),

    // Promise creator
    resolver: t.hex().notNull(),

    // Promise status: 'pending', 'resolved', 'rejected', 'transferred'
    status: t.text().notNull().default('pending'),

    // Creation info
    createdAt: t.bigint().notNull(),
    createdBlockNumber: t.bigint().notNull(),
    createdTransactionHash: t.hex().notNull(),

    // Transfer info (set when ResolutionTransferred fires on origin)
    transferredAt: t.bigint(),
    transferredBlockNumber: t.bigint(),
    transferredTransactionHash: t.hex(),

    // Resolution info (if resolved/rejected)
    resolvedAt: t.bigint(),
    resolvedBlockNumber: t.bigint(),
    resolvedTransactionHash: t.hex(),

    // Resolution payload + who resolved it (from PromiseResolved).
    returnData: t.hex(),
    resolvedBy: t.hex(),
  }),
  (table) => ({
    pk: primaryKey({ columns: [table.chainId, table.promiseId] }),
    resolverIdx: index().on(table.resolver),
    statusIdx: index().on(table.status),
  }),
)

// Raw ResolutionTransferred events. Source of truth for a promise's "home"
// chain (transfer destination), used by /promise-chains. Keyed (chainId,
// promiseId) = the origin chain + promise.
export const ResolutionTransferred = onchainTable(
  'resolution_transferred',
  (t) => ({
    promiseId: t.hex().notNull(),
    destinationChainId: t.bigint().notNull(),
    resolver: t.hex().notNull(),
    chainId: t.bigint().notNull(),

    // general fields
    timestamp: t.bigint().notNull(),
    blockNumber: t.bigint().notNull(),
    transactionHash: t.hex().notNull(),
  }),
  (table) => ({
    pk: primaryKey({ columns: [table.chainId, table.promiseId] }),
    promiseIdIdx: index().on(table.promiseId),
  }),
)
