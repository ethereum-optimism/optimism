// Extracted Gas Tank handler functions for testing

export async function handleGasTankDeposit(event: any, context: any) {
  await context.db
    .insert('gasTankGasProviders')
    .values({
      chainId: BigInt(context.chain.id),
      address: event.args.depositor,
      balance: event.args.amount,
      lastUpdatedAt: event.block.timestamp,
    })
    .onConflictDoUpdate((row: any) => ({
      balance: row.balance + event.args.amount,
      lastUpdatedAt: event.block.timestamp,
    }))
}

export async function handleGasTankWithdrawalInitiated(event: any, context: any) {
  await context.db
    .insert('gasTankPendingWithdrawals')
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
}

export async function handleGasTankWithdrawalFinalized(event: any, context: any) {
  await context.db.delete('gasTankPendingWithdrawals', {
    chainId: BigInt(context.chain.id),
    address: event.args.from,
  })

  await context.db
    .update('gasTankGasProviders', {
      chainId: BigInt(context.chain.id),
      address: event.args.from,
    })
    .set((row: any) => ({
      balance: row.balance - event.args.amount,
      lastUpdatedAt: event.block.timestamp,
    }))
}

export async function handleGasTankFlagged(event: any, context: any) {
  await context.db
    .insert('gasTankFlaggedMessages')
    .values({
      chainId: BigInt(context.chain.id),
      gasProvider: event.args.gasProvider,
      originMessageHash: event.args.originMsgHash,
      flaggedAt: event.block.timestamp,
    })
    .onConflictDoNothing()
}

export async function handleGasTankClaimed(event: any, context: any) {
  await context.db
    .update('gasTankGasProviders', {
      chainId: BigInt(context.chain.id),
      address: event.args.gasProvider,
    })
    .set((row: any) => ({
      balance: row.balance - event.args.amount,
      lastUpdatedAt: event.block.timestamp,
    }))

  await context.db.insert('gasTankClaimedMessages').values({
    originMessageHash: event.args.originMsgHash,
    chainId: BigInt(context.chain.id),
    relayer: event.args.relayer,
    gasProvider: event.args.gasProvider,
    amountClaimed: event.args.amount,
    claimedAt: event.block.timestamp,
  })
}

export async function handleGasTankRelayedMessageGasReceipt(event: any, context: any) {
  await context.db.insert('gasTankRelayedMessageReceipts').values({
    originMessageHash: event.args.originMsgHash,
    chainId: BigInt(context.chain.id),
    relayer: event.args.relayer,
    gasCost: event.args.gasCost,
    destinationMessageHashes: [...event.args.destinationMessageHashes],
    relayedAt: event.block.timestamp,
  })
}