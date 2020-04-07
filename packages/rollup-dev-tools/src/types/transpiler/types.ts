import { EVMBytecode } from '@eth-optimism/rollup-core'

export interface TranspilationError {
  index: number
  error: number
  message: string
}

export interface TranspilationResultBase {
  succeeded: boolean
}

export interface ErroredTranspilation extends TranspilationResultBase {
  succeeded: false
  errors: TranspilationError[]
}

export interface SuccessfulTranspilation extends TranspilationResultBase {
  succeeded: true
  bytecode: Buffer
}

export type TranspilationResult = ErroredTranspilation | SuccessfulTranspilation

export interface JumpReplacementResult {
  bytecode: EVMBytecode
  errors?: TranspilationError[]
}

export interface TaggedTranspilationResult {
  succeeded: boolean
  errors?: TranspilationError[]
  bytecodeWithTags?: EVMBytecode
}

export interface BinarySearchTreeNode {
  value: {
    jumpdestBefore: number
    jumpdestAfter: number
  }
  left: BinarySearchTreeNode
  right: BinarySearchTreeNode
}
