// Extracted Promise + Callback handler functions for testing. These mirror the
// inline `ponder.on(...)` bodies in src/index.ts (same convention as
// gas-tank-handlers.ts / l2-to-l2-cdm-handlers.ts). The real handlers are
// type-checked against the generated ponder registry; these let us assert the
// db-call shape without a live database. The unshared-resolved *join* is not
// exercised here — it needs real rows and is verified at the supersim e2e (C2).

const ZERO = '0x0000000000000000000000000000000000000000'

export async function handleCallbackRegistered(event: any, context: any) {
  await context.db.insert('callbacks').values({
    callbackPromiseId: event.args.callbackPromiseId,
    parentPromiseId: event.args.parentPromiseId,
    chainId: BigInt(context.chain.id),
    target: event.log.address,
    callbackType: Number(event.args.callbackType),
    timestamp: event.block.timestamp,
    blockNumber: event.block.number,
    transactionHash: event.transaction.hash,
  })
}

export async function handlePromiseCreated(event: any, context: any) {
  await context.db
    .insert('promises')
    .values({
      promiseId: event.args.promiseId,
      chainId: BigInt(context.chain.id),
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
}

export async function handlePromiseResolved(event: any, context: any) {
  await context.db
    .insert('promises')
    .values({
      promiseId: event.args.promiseId,
      chainId: BigInt(context.chain.id),
      resolver: ZERO,
      status: 'resolved',
      createdAt: event.block.timestamp,
      createdBlockNumber: event.block.number,
      createdTransactionHash: event.transaction.hash,
      resolvedAt: event.block.timestamp,
      resolvedBlockNumber: event.block.number,
      resolvedTransactionHash: event.transaction.hash,
    })
    .onConflictDoUpdate({
      status: 'resolved',
      resolvedAt: event.block.timestamp,
      resolvedBlockNumber: event.block.number,
      resolvedTransactionHash: event.transaction.hash,
    })
}

export async function handlePromiseRejected(event: any, context: any) {
  await context.db
    .insert('promises')
    .values({
      promiseId: event.args.promiseId,
      chainId: BigInt(context.chain.id),
      resolver: ZERO,
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
}

export async function handleResolutionTransferred(event: any, context: any) {
  await context.db
    .insert('promises')
    .values({
      promiseId: event.args.promiseId,
      chainId: BigInt(context.chain.id),
      resolver: event.args.resolver,
      status: 'transferred',
      createdAt: event.block.timestamp,
      createdBlockNumber: event.block.number,
      createdTransactionHash: event.transaction.hash,
      transferredAt: event.block.timestamp,
      transferredBlockNumber: event.block.number,
      transferredTransactionHash: event.transaction.hash,
    })
    .onConflictDoUpdate((row: any) => ({
      status: row.status === 'pending' ? 'transferred' : row.status,
      transferredAt: event.block.timestamp,
      transferredBlockNumber: event.block.number,
      transferredTransactionHash: event.transaction.hash,
    }))
}
