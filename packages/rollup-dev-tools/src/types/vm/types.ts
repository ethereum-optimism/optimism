import { EVMOpcode } from '@pigi/rollup-core'

export interface StepContext {
  pc: number
  opcode: EVMOpcode
  stack: Buffer[]
  stackDepth: number
  memory: Buffer[]
  memoryWordCount: number
}

export interface ExecutionResultComparison {
  resultsDiffer: boolean
  firstResult: Buffer
  secondResult: Buffer
  firstError?: EvmError
  secondError?: EvmError
}

/* Right now duping ethereumjs-vm errors, but separated to isolate dependency */
export const OUT_OF_GAS_ERROR = 'out of gas'
export const STACK_UNDERFLOW_ERROR = 'stack underflow'
export const STACK_OVERFLOW_ERROR = 'stack overflow'
export const INVALID_JUMP_ERROR = 'invalid JUMP'
export const INVALID_OPCODE_ERROR = 'invalid opcode'
export const OUT_OF_RANGE_ERROR = 'value out of range'
export const REVERT_ERROR = 'revert'
export const STATIC_STATE_CHANGE_ERROR = 'static state change'
export const INTERNAL_ERROR = 'internal error'
export const CREATE_COLLISION_ERROR = 'create collision'
export const STOP_ERROR = 'stop'
export const REFUND_EXHAUSTED_ERROR = 'refund exhausted'

export type EvmError = string
