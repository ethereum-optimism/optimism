import { beforeEach, describe, expect, it, vi } from 'vitest'

import {
  handleCallbackRegistered,
  handlePromiseCreated,
  handlePromiseRejected,
  handlePromiseResolved,
  handleResolutionTransferred,
} from './handlers/promise-handlers'

const mockOnConflictDoUpdate = vi.fn().mockResolvedValue(undefined)
const mockValues = vi.fn(() => ({
  onConflictDoUpdate: mockOnConflictDoUpdate,
}))
const mockDb = {
  insert: vi.fn(() => ({ values: mockValues })),
}

function ctx(chainId: number) {
  return { db: mockDb, chain: { id: chainId } }
}

const PROMISE_ID =
  '0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'

describe('Promise + Callback indexing', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('CallbackRegistered inserts into callbacks keyed by the firing chain', async () => {
    const event = {
      args: {
        callbackPromiseId:
          '0xcccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc',
        parentPromiseId: PROMISE_ID,
        callbackType: 0,
      },
      log: { address: '0x1111111111111111111111111111111111111111' },
      block: mockBlock({ timestamp: 1700000000n, number: 10n }),
      transaction: mockTransaction({ hash: '0xreg' }),
    }

    await handleCallbackRegistered(event, ctx(902))

    expect(mockDb.insert).toHaveBeenCalledWith('callbacks')
    expect(mockValues).toHaveBeenCalledWith(
      expect.objectContaining({
        callbackPromiseId: event.args.callbackPromiseId,
        parentPromiseId: PROMISE_ID,
        chainId: 902n,
        target: '0x1111111111111111111111111111111111111111',
        callbackType: 0,
      }),
    )
  })

  it('PromiseCreated upserts a pending promise row on the firing chain', async () => {
    const event = {
      args: {
        promiseId: PROMISE_ID,
        resolver: '0x2222222222222222222222222222222222222222',
      },
      block: mockBlock({ timestamp: 1700000000n, number: 5n }),
      transaction: mockTransaction({ hash: '0xcreate' }),
    }

    await handlePromiseCreated(event, ctx(901))

    expect(mockDb.insert).toHaveBeenCalledWith('promises')
    expect(mockValues).toHaveBeenCalledWith(
      expect.objectContaining({
        promiseId: PROMISE_ID,
        chainId: 901n,
        resolver: '0x2222222222222222222222222222222222222222',
        status: 'pending',
      }),
    )
  })

  // The §10 Q2 mechanism: a resolved row is keyed by the chain the event fired
  // on. When a shared resolution lands on a destination, receiveSharedPromise
  // emits PromiseResolved there, so a resolved row appears on the *destination*
  // chainId — which is how the relayer learns that chain is no longer unshared.
  it('PromiseResolved upserts a resolved row keyed by the destination chain', async () => {
    const event = {
      args: { promiseId: PROMISE_ID },
      block: mockBlock({ timestamp: 1700000100n, number: 20n }),
      transaction: mockTransaction({ hash: '0xresolve' }),
    }

    await handlePromiseResolved(event, ctx(902))

    expect(mockValues).toHaveBeenCalledWith(
      expect.objectContaining({
        promiseId: PROMISE_ID,
        chainId: 902n,
        status: 'resolved',
        resolvedTransactionHash: '0xresolve',
      }),
    )
    expect(mockOnConflictDoUpdate).toHaveBeenCalledWith(
      expect.objectContaining({ status: 'resolved' }),
    )
  })

  it('PromiseRejected upserts a rejected row', async () => {
    const event = {
      args: { promiseId: PROMISE_ID },
      block: mockBlock({ timestamp: 1700000100n, number: 21n }),
      transaction: mockTransaction({ hash: '0xreject' }),
    }

    await handlePromiseRejected(event, ctx(901))

    expect(mockOnConflictDoUpdate).toHaveBeenCalledWith(
      expect.objectContaining({ status: 'rejected' }),
    )
  })

  // ResolutionTransferred must not clobber a terminal status that arrived first
  // (cross-chain ordering): the onConflict update only flips 'pending' →
  // 'transferred'.
  it('ResolutionTransferred only flips a pending row to transferred', async () => {
    const event = {
      args: {
        promiseId: PROMISE_ID,
        destinationChainId: 902n,
        resolver: '0x3333333333333333333333333333333333333333',
      },
      block: mockBlock({ timestamp: 1700000200n, number: 30n }),
      transaction: mockTransaction({ hash: '0xtransfer' }),
    }

    await handleResolutionTransferred(event, ctx(901))

    expect(mockOnConflictDoUpdate).toHaveBeenCalledWith(expect.any(Function))
    const updateFn = mockOnConflictDoUpdate.mock.calls[0][0] as (
      row: any,
    ) => any
    expect(updateFn({ status: 'pending' }).status).toBe('transferred')
    expect(updateFn({ status: 'resolved' }).status).toBe('resolved')
  })
})
