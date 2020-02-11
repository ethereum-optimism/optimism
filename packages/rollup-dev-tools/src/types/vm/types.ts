import { Address, EVMOpcode } from '@eth-optimism/rollup-core'

export interface ExecutionResult {
  result: Buffer
  error?: EvmError
}

export interface StepContext {
  address: Address
  pc: number
  opcode: EVMOpcode
  stack: Buffer[]
  stackDepth: number
  memory: Buffer
  memoryWordCount: number
}

export interface CallContext {
  input: {
    gas
    addr: Address
    value: Buffer
    argOffset: number
    argLength: number
    retOffset: number
    retLength: number
  }
  callData: Buffer
  stepContext: StepContext
}

export interface ExecutionResultComparison {
  resultsDiffer: boolean
  firstResult: ExecutionResult
  secondResult: ExecutionResult
}

export interface ExecutionComparison {
  executionDiffers: boolean
  firstContext: StepContext
  secondContext: StepContext
}

/* Right now duping ethereumjs-vm errors, but separated to isolate dependency */
export class EvmErrors {
  public static readonly OUT_OF_GAS_ERROR: EvmError = 'out of gas'
  public static readonly STACK_UNDERFLOW_ERROR: EvmError = 'stack underflow'
  public static readonly STACK_OVERFLOW_ERROR: EvmError = 'stack overflow'
  public static readonly INVALID_JUMP_ERROR: EvmError = 'invalid JUMP'
  public static readonly INVALID_OPCODE_ERROR: EvmError = 'invalid opcode'
  public static readonly OUT_OF_RANGE_ERROR: EvmError = 'value out of range'
  public static readonly REVERT_ERROR: EvmError = 'revert'
  public static readonly STATIC_STATE_CHANGE_ERROR: EvmError =
    'static state change'
  public static readonly INTERNAL_ERROR: EvmError = 'internal error'
  public static readonly CREATE_COLLISION_ERROR: EvmError = 'create collision'
  public static readonly STOP_ERROR: EvmError = 'stop'
  public static readonly REFUND_EXHAUSTED_ERROR: EvmError = 'refund exhausted'
}

export type EvmError = string
