import {
  bytecodeToBuffer,
  EVMBytecode,
  Opcode,
  EVMOpcodeAndBytes,
  formatBytecode,
  getPCOfEVMBytecodeIndex,
} from '@eth-optimism/rollup-core'
import { bufferUtils, getLogger } from '@eth-optimism/core-utils'
import { getPUSHOpcode, getPUSHIntegerOp } from './helpers'
import {
  JumpReplacementResult,
  TranspilationError,
  TranspilationErrors,
} from '../../types/transpiler'
import { createError } from './util'
import { TranscodeEncoding } from 'buffer'
import { UnicodeNormalizationForm } from 'ethers/utils'

const log = getLogger('log-search-generator')

type LogSearchLeafNode = {
  key: number
  value: number
}

type LogSearchInternalNode = {
  largestAncestorKey: number
  keyToCompare: number
}

type LogSearchNode = LogSearchLeafNode | LogSearchInternalNode

type LogSearchTree = LogSearchNode[][]

const maxBytesOfContractSize = 2

const IS_PUSH_BINARY_SEARCH_NODE_LOCATION =
  'IS_PUSH_BINARY_SEARCH_NODE_LOCATION'
const IS_BINARY_SEARCH_NODE_JUMPDEST = 'IS_BINARY_SEARCH_NODE_JUMPDEST'

export const generateLogSearchTree = (
  keys: number[],
  values: number[]
): LogSearchTree => {
  let tree: LogSearchTree = [[]]
  // initialize tree's bottom level with key/value leaves
  tree[0] = keys.map((v, i) => {
    return {
      key: keys[i],
      value: values[i],
    }
  })

  let treeHeight = Math.ceil(Math.log2(keys.length))
  for (let depth = 1; depth <= treeHeight; depth++) {
    tree[depth] = []
    for (let i = 0; i < tree[depth - 1].length; i += 2) {
      let nodeToCreate: LogSearchInternalNode = {
        largestAncestorKey: undefined,
        keyToCompare: undefined,
      }
      const indexToCreate = i / 2
      const leftChild = tree[depth - 1][i]
      const rightChild = tree[depth - 1][i + 1]
      if (!rightChild) {
        tree[depth][indexToCreate] = tree[depth - 1].pop()
        continue
      }
      // if leaf node right child, its key is greatest ancestor
      if (isLeafNode(rightChild)) {
        nodeToCreate.largestAncestorKey = (rightChild as LogSearchLeafNode).key
      } else {
        nodeToCreate.largestAncestorKey = (rightChild as LogSearchInternalNode).largestAncestorKey
      }

      if (isLeafNode(leftChild)) {
        nodeToCreate.keyToCompare = (leftChild as LogSearchLeafNode).key
      } else {
        nodeToCreate.keyToCompare = (leftChild as LogSearchInternalNode).largestAncestorKey
      }

      tree[depth][indexToCreate] = nodeToCreate
    }
  }
  return tree.reverse() // reverse so that tree[0][0] is the root node
}

const isLeafNode = (node: LogSearchNode): boolean => {
  return !!(node as LogSearchLeafNode).value
}

export const getJumpIndexSearchBytecode = (
  jumpdestIndexesBefore: number[],
  jumpdestIndexesAfter: number[],
  indexOfThisBlock: number
): EVMBytecode => {
  const searchTree = generateLogSearchTree(
    jumpdestIndexesBefore,
    jumpdestIndexesAfter
  )
  const bytecode: EVMBytecode = [
    {
      opcode: Opcode.JUMPDEST,
      consumedBytes: undefined,
    },
    ...appendNodeToBytecode(searchTree, 0, 0, []), // should recursively fill out tree
  ]
  return fixTaggedNodePositions(bytecode, indexOfThisBlock)
}

const fixTaggedNodePositions = (
  bytecode: EVMBytecode,
  indexOfThisBlock: number
): EVMBytecode => {
  for (let opcodeAndBytes of bytecode) {
    if (
      !!opcodeAndBytes.tag &&
      opcodeAndBytes.tag.reasonTagged == IS_PUSH_BINARY_SEARCH_NODE_LOCATION
    ) {
      const thisNodePosition = opcodeAndBytes.tag.metadata
      const rightChildNodeLevel = thisNodePosition.level + 1
      const rightChildNodeIndex = thisNodePosition.index * 2 + 1
      const rightChildJumpdestIndexInBytecodeBlock = bytecode.findIndex(
        (opcodeAndBytes: EVMOpcodeAndBytes) => {
          return (
            !!opcodeAndBytes.tag &&
            opcodeAndBytes.tag.reasonTagged == IS_BINARY_SEARCH_NODE_JUMPDEST &&
            opcodeAndBytes.tag.metadata.level == rightChildNodeLevel &&
            opcodeAndBytes.tag.metadata.index == rightChildNodeIndex
          )
        }
      )
      const rightChildJumpdestInFinalBuffer =
        indexOfThisBlock +
        getPCOfEVMBytecodeIndex(
          rightChildJumpdestIndexInBytecodeBlock,
          bytecode
        )
      opcodeAndBytes.consumedBytes = bufferUtils.numberToBuffer(
        rightChildJumpdestInFinalBuffer,
        maxBytesOfContractSize,
        maxBytesOfContractSize
      )
    }
  }
  return bytecode
}

const appendNodeToBytecode = (
  tree: LogSearchTree,
  level: number,
  index: number,
  bytecode: EVMBytecode
): EVMBytecode => {
  const thisNode: LogSearchNode = tree[level][index]
  log.info(
    `Processing node at level ${level} and index ${index} of tree into bytecode, its parameters are ${thisNode}.`
  )
  if (isLeafNode(thisNode)) {
    bytecode = [
      ...bytecode,
      ...generateLeafBytecode(thisNode as LogSearchLeafNode, level, index),
    ]
  } else {
    bytecode = [
      ...bytecode,
      ...generateComparisonBytecode(
        thisNode as LogSearchInternalNode,
        level,
        index
      ),
    ]
    const leftChildIndex = index * 2
    const rightChildIndex = leftChildIndex + 1
    const childrenLevel = level + 1
    bytecode = appendNodeToBytecode(
      tree,
      childrenLevel,
      leftChildIndex,
      bytecode
    )
    bytecode = appendNodeToBytecode(
      tree,
      childrenLevel,
      rightChildIndex,
      bytecode
    )
  }
  return bytecode
}

const generateComparisonBytecode = (
  node: LogSearchInternalNode,
  level: number,
  index: number
): EVMBytecode => {
  // if index is odd, add and tag JUMPDEST with "treeNodePosition" = [index, level]
  // For GT check working, add "destinationNodePosition" = index*2+1, level+1
  let bytecodeToReturn: EVMBytecode = []
  const willBeJUMPedTo: boolean = index % 2 == 0 ? false : true
  if (willBeJUMPedTo) {
    bytecodeToReturn.push(generateSearchNodeJumpdest(level, index))
  }

  bytecodeToReturn = [
    ...bytecodeToReturn,
    {
      opcode: Opcode.DUP1,
      consumedBytes: undefined,
    },
    getPUSHIntegerOp(node.keyToCompare),
    {
      opcode: Opcode.LT,
      consumedBytes: undefined,
    },
    {
      opcode: getPUSHOpcode(maxBytesOfContractSize),
      consumedBytes: Buffer.alloc(maxBytesOfContractSize),
      tag: {
        padPUSH: false,
        reasonTagged: IS_PUSH_BINARY_SEARCH_NODE_LOCATION,
        metadata: { level, index },
      },
    },
    {
      opcode: Opcode.SWAP1,
      consumedBytes: undefined,
    },
    {
      opcode: Opcode.JUMPI,
      consumedBytes: undefined,
    },
  ]

  return bytecodeToReturn
}

const generateLeafBytecode = (
  node: LogSearchLeafNode,
  level: number,
  index: number
): EVMBytecode => {
  // do the matching stuff
  let bytecodeToReturn: EVMBytecode = []
  const willBeJUMPedTo: boolean = index % 2 == 0 ? false : true
  if (willBeJUMPedTo) {
    bytecodeToReturn.push(generateSearchNodeJumpdest(level, index))
  }

  bytecodeToReturn = [
    ...bytecodeToReturn,
    {
      opcode: Opcode.POP,
      consumedBytes: undefined,
    },
    getPUSHIntegerOp(node.value),
    {
      opcode: Opcode.JUMP,
      consumedBytes: undefined,
    },
  ]
  return bytecodeToReturn
}

const generateSearchNodeJumpdest = (
  level: number,
  index: number
): EVMOpcodeAndBytes => {
  return {
    opcode: Opcode.JUMPDEST,
    consumedBytes: undefined,
    tag: {
      padPUSH: false,
      reasonTagged: IS_BINARY_SEARCH_NODE_JUMPDEST,
      metadata: { level, index },
    },
  }
}
