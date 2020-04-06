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

// Conceptual data structures representing a leaf node in a binary search tree.

export interface BinarySearchLeafNode {
  key: number // The number which the binary search should be searching against.
  value: number // The number which should be returned if the binary search matches the input to this key.
  nodeId: number // Identifier for this node in the binary search tree, used for indexing
}

export interface BinarySearchInternalNode {
  largestAncestorKey: number // The largest .key value of any leaf-node ancestor of this node.
  keyToCompare: number // The key which this node should compare against in the search, to progress either to the right or left child.
  nodeId: number // Identifier for this node in the binary search tree, used for indexing
  leftChildNodeId: number // Identifier of this node's left child.
  rightChildNodeId: number // Identifier of this node's right child.
}

export type BinarySearchNode = BinarySearchLeafNode | BinarySearchInternalNode
