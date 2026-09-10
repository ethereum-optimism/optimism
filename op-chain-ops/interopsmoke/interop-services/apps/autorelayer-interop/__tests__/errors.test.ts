import {
  type Abi,
  BaseError,
  ContractFunctionRevertedError,
  encodeErrorResult,
} from 'viem'
import { describe, expect, it } from 'vitest'

import {
  classifyL2ToL2CdmRevert,
  classifyRevert,
  decodeRelayRevert,
  L2_TO_L2_CDM_ERROR_REASONS,
  RelayError,
} from '@/errors.js'

const abi = [
  {
    type: 'error',
    name: 'MessageAlreadyRelayed',
    inputs: [],
  },
  {
    type: 'error',
    name: 'InsufficientBalance',
    inputs: [{ name: 'available', type: 'uint256' }],
  },
] as const satisfies Abi

describe('RelayError', () => {
  it('carries stage, reason, and cause', () => {
    const cause = new Error('underlying failure')
    const err = new RelayError({
      stage: 'simulation',
      reason: 'already_relayed',
      cause,
    })

    expect(err.stage).toBe('simulation')
    expect(err.reason).toBe('already_relayed')
    expect(err.cause).toBe(cause)
    expect(err.name).toBe('RelayError')
  })

  it('formats a default message from stage and reason', () => {
    const err = new RelayError({
      stage: 'broadcast',
      reason: 'submission_failed',
    })
    expect(err.message).toBe('relay failed at broadcast: submission_failed')
  })

  it('respects a custom message override', () => {
    const err = new RelayError({
      stage: 'execution',
      reason: 'already_relayed',
      message: 'another relayer beat us to it',
    })
    expect(err.message).toBe('another relayer beat us to it')
  })

  it('preserves instanceof across the prototype chain', () => {
    const err = new RelayError({ stage: 'simulation', reason: 'unknown' })
    expect(err).toBeInstanceOf(RelayError)
    expect(err).toBeInstanceOf(Error)
  })
})

describe('decodeRelayRevert', () => {
  it('returns undefined for non-viem errors', () => {
    expect(decodeRelayRevert(new Error('boom'), abi)).toBeUndefined()
    expect(decodeRelayRevert('string', abi)).toBeUndefined()
    expect(decodeRelayRevert(undefined, abi)).toBeUndefined()
  })

  it('returns undefined for a viem error with no revert in its chain', () => {
    const err = new BaseError('network flake')
    expect(decodeRelayRevert(err, abi)).toBeUndefined()
  })

  it('decodes a named custom error via viem-provided data', () => {
    const reverted = new ContractFunctionRevertedError({
      abi,
      data: encodeErrorResult({ abi, errorName: 'MessageAlreadyRelayed' }),
      functionName: 'relayMessage',
    })
    const wrapper = new BaseError('simulation failed', { cause: reverted })

    const result = decodeRelayRevert(wrapper, abi)
    expect(result).toEqual({
      errorName: 'MessageAlreadyRelayed',
      args: undefined,
    })
  })

  it('decodes a custom error with args', () => {
    const reverted = new ContractFunctionRevertedError({
      abi,
      data: encodeErrorResult({
        abi,
        errorName: 'InsufficientBalance',
        args: [42n],
      }),
      functionName: 'relayMessage',
    })

    const result = decodeRelayRevert(reverted, abi)
    expect(result?.errorName).toBe('InsufficientBalance')
    expect(result?.args).toEqual([42n])
  })
})

describe('classifyRevert', () => {
  const reasonMap = { MessageAlreadyRelayed: 'already_relayed' }

  it('returns undefined for non-revert errors', () => {
    expect(classifyRevert(new Error('boom'), abi, reasonMap)).toBeUndefined()
  })

  it('maps a known error name to its reason', () => {
    const reverted = new ContractFunctionRevertedError({
      abi,
      data: encodeErrorResult({ abi, errorName: 'MessageAlreadyRelayed' }),
      functionName: 'relayMessage',
    })
    expect(classifyRevert(reverted, abi, reasonMap)).toEqual({
      errorName: 'MessageAlreadyRelayed',
      reason: 'already_relayed',
    })
  })

  it('falls back to unknown for decoded-but-unmapped error names', () => {
    const reverted = new ContractFunctionRevertedError({
      abi,
      data: encodeErrorResult({
        abi,
        errorName: 'InsufficientBalance',
        args: [0n],
      }),
      functionName: 'relayMessage',
    })
    expect(classifyRevert(reverted, abi, reasonMap)).toEqual({
      errorName: 'InsufficientBalance',
      reason: 'unknown',
    })
  })
})

describe('classifyL2ToL2CdmRevert', () => {
  it('recognizes MessageAlreadyRelayed', () => {
    expect(L2_TO_L2_CDM_ERROR_REASONS.MessageAlreadyRelayed).toBe(
      'already_relayed',
    )
  })

  it('returns undefined for non-revert errors', () => {
    expect(classifyL2ToL2CdmRevert(new Error('network flake'))).toBeUndefined()
  })

  it('classifies op-supervisor RPC rejections as rpc_rejected', async () => {
    const { InvalidParamsRpcError, TransactionExecutionError } = await import(
      'viem'
    )
    // Mirrors the real shape produced by viem's sendTransaction when supersim's
    // RPC rejects an executing message at admission. The "failed tx simulation"
    // marker is what viem surfaces in the message body.
    const rpcError = new InvalidParamsRpcError(
      new Error(
        'Invalid parameters were provided to the RPC method.\n\n' +
          'Details: failed tx simulation',
      ),
    )
    const wrapped = new TransactionExecutionError(rpcError, {
      account: { address: '0x0000000000000000000000000000000000000001' } as any,
    })
    expect(classifyL2ToL2CdmRevert(wrapped)).toEqual({
      errorName: 'RpcRejected',
      reason: 'rpc_rejected',
    })
  })

  it('does not classify InvalidParamsRpcError without the supersim marker', async () => {
    const { InvalidParamsRpcError } = await import('viem')
    const rpcError = new InvalidParamsRpcError(
      new Error('Invalid parameters were provided to the RPC method.'),
    )
    expect(classifyL2ToL2CdmRevert(rpcError)).toBeUndefined()
  })
})

describe('classifyGasTankRevert (R4: parity with the CDM path)', () => {
  it('classifies op-supervisor RPC rejections as rpc_rejected', async () => {
    const { classifyGasTankRevert } = await import('@/errors.js')
    const { InvalidParamsRpcError, TransactionExecutionError } = await import(
      'viem'
    )
    // Same supervisor refusal shape as the CDM test above. On the gas-tank
    // path this classified 'unknown' → ~43 min of count-based retries,
    // instead of permanent on first failure like the CDM path.
    const rpcError = new InvalidParamsRpcError(
      new Error(
        'Invalid parameters were provided to the RPC method.\n\n' +
          'Details: failed tx simulation',
      ),
    )
    const wrapped = new TransactionExecutionError(rpcError, {
      account: { address: '0x0000000000000000000000000000000000000001' } as any,
    })
    expect(classifyGasTankRevert(wrapped)).toEqual({
      errorName: 'RpcRejected',
      reason: 'rpc_rejected',
    })
  })

  it('maps GasTank custom errors to reasons (map must not be an empty stub)', async () => {
    const { GAS_TANK_ERROR_REASONS } = await import('@/errors.js')
    // The GasTank contract's relay/claim custom errors, per the gasTankAbi
    // the relayer simulates against.
    expect(GAS_TANK_ERROR_REASONS.InsufficientBalance).toBe(
      'insufficient_gas_tank_balance',
    )
    expect(GAS_TANK_ERROR_REASONS.AlreadyClaimed).toBe('already_claimed')
    expect(GAS_TANK_ERROR_REASONS.InvalidOrigin).toBe('invalid_origin')
    expect(GAS_TANK_ERROR_REASONS.InvalidPayload).toBe('invalid_payload')
  })

  it('classifies a decoded GasTank revert via the reasons map', async () => {
    const { classifyGasTankRevert } = await import('@/errors.js')
    const { gasTankAbi } = await import('@eth-optimism/viem/abis/experimental')
    const reverted = new ContractFunctionRevertedError({
      abi: gasTankAbi as Abi,
      data: encodeErrorResult({
        abi: gasTankAbi as Abi,
        errorName: 'InsufficientBalance',
      }),
      functionName: 'relayMessage',
    })
    expect(classifyGasTankRevert(reverted)).toEqual({
      errorName: 'InsufficientBalance',
      reason: 'insufficient_gas_tank_balance',
    })
  })

  it('classifies a nested L2ToL2CDM revert surfaced through the GasTank call', async () => {
    const { classifyGasTankRevert } = await import('@/errors.js')
    const { l2ToL2CrossDomainMessengerAbi } = await import(
      '@eth-optimism/viem/abis'
    )
    // GasTank.relayMessage calls MESSENGER.relayMessage internally, so a
    // MessageAlreadyRelayed revert bubbles up undecodable against gasTankAbi
    // alone (raw data only).
    const reverted = new ContractFunctionRevertedError({
      abi: [] as unknown as Abi, // viem could not decode against the call ABI
      data: encodeErrorResult({
        abi: l2ToL2CrossDomainMessengerAbi as Abi,
        errorName: 'MessageAlreadyRelayed',
      }),
      functionName: 'relayMessage',
    })
    expect(classifyGasTankRevert(reverted)).toEqual({
      errorName: 'MessageAlreadyRelayed',
      reason: 'already_relayed',
    })
  })
})
