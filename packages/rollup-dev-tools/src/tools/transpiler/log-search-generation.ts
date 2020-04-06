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

interface LogSearchLeafNode {
  key: number
  value: number
  nodeId: number
}

interface LogSearchInternalNode {
  largestAncestorKey: number
  keyToCompare: number
  nodeId: number
  leftChildNodeId: number
  rightChildNodeId: number
}

type LogSearchNode = LogSearchLeafNode | LogSearchInternalNode

type LogSearchTree = LogSearchNode[][]

const maxBytesOfContractSize = 2

const IS_PUSH_BINARY_SEARCH_NODE_LOCATION =
  'IS_PUSH_BINARY_SEARCH_NODE_LOCATION'
const IS_BINARY_SEARCH_NODE_JUMPDEST = 'IS_BINARY_SEARCH_NODE_JUMPDEST'

export const generateLogSearchTreeNodes = (
  keys: number[],
  values: number[]
): LogSearchNode[] => {
  const allTreeNodes: LogSearchNode[] = keys.map((v, i) => {
    return {
      key: keys[i],
      value: values[i],
      nodeId: i,
    }
  })

  let curLevel: LogSearchNode[] = [...allTreeNodes]
  while (curLevel.length > 1) {
    // console.log(`processing level: ${JSON.stringify(curLevel)}`)
    const nextLevel: LogSearchNode[] = []
    for (let i = 0; i < curLevel.length; i += 2) {
      const leftChild = curLevel[i]
      const rightChild = curLevel[i + 1]
      if (!rightChild) {
        nextLevel.push(leftChild)
        continue
      }

      const newNode: LogSearchInternalNode = {
        nodeId: allTreeNodes.length,
        leftChildNodeId: leftChild.nodeId,
        rightChildNodeId: rightChild.nodeId,
        largestAncestorKey: undefined,
        keyToCompare: undefined,
      }

      newNode.largestAncestorKey = isLeafNode(rightChild)
        ? (rightChild as LogSearchLeafNode).key
        : (rightChild as LogSearchInternalNode).largestAncestorKey

      newNode.keyToCompare = isLeafNode(leftChild)
        ? (leftChild as LogSearchLeafNode).key
        : (newNode.keyToCompare = (leftChild as LogSearchInternalNode).largestAncestorKey)

      allTreeNodes.push(newNode)
      nextLevel.push(newNode)
    }
    curLevel = nextLevel
  }
  return allTreeNodes
}

const isLeafNode = (node: LogSearchNode): boolean => {
  return !!(node as LogSearchLeafNode).value
}

export const getJumpIndexSearchBytecode = (
  jumpdestIndexesBefore: number[],
  jumpdestIndexesAfter: number[],
  indexOfThisBlock: number
): EVMBytecode => {
  const searchTreeNodes: LogSearchNode[] = generateLogSearchTreeNodes(
    jumpdestIndexesBefore,
    jumpdestIndexesAfter
  )
  log.debug(
    `successfully generated conceptual log search tree, its flat structure is: \n${JSON.stringify(
      searchTreeNodes
    )}`
  )
  const rootNodeId = searchTreeNodes.length - 1 // root node is the final one
  const bytecode: EVMBytecode = [
    {
      opcode: Opcode.JUMPDEST,
      consumedBytes: undefined,
    },
    ...appendNodeToBytecode(searchTreeNodes, rootNodeId, true, []), // should recursively fill out tree
  ]
  const finalBytecode = fixTaggedNodePositions(
    bytecode,
    indexOfThisBlock,
    searchTreeNodes
  )
  // log.debug(`generated final bytecode for log searcher : \n${formatBytecode(finalBytecode)}`)
  return finalBytecode
}

const fixTaggedNodePositions = (
  bytecode: EVMBytecode,
  indexOfThisBlock: number,
  searchTreeNodes: LogSearchNode[]
): EVMBytecode => {
  for (const opcodeAndBytes of bytecode) {
    if (
      !!opcodeAndBytes.tag &&
      opcodeAndBytes.tag.reasonTagged === IS_PUSH_BINARY_SEARCH_NODE_LOCATION
    ) {
      const thisNodeId = opcodeAndBytes.tag.metadata.nodeId
      const rightChildNodeId = (searchTreeNodes[
        thisNodeId
      ] as LogSearchInternalNode).rightChildNodeId
      const rightChildJumpdestIndexInBytecodeBlock = bytecode.findIndex(
        (toCheck: EVMOpcodeAndBytes) => {
          return (
            !!toCheck.tag &&
            toCheck.tag.reasonTagged === IS_BINARY_SEARCH_NODE_JUMPDEST &&
            toCheck.tag.metadata.nodeId === rightChildNodeId
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
  treeNodes: LogSearchNode[],
  nodeId: number,
  isLeftSibling: boolean,
  bytecode: EVMBytecode
): EVMBytecode => {
  const thisNode: LogSearchNode = treeNodes[nodeId]
  log.info(
    `Processing node with Id ${nodeId} of tree into bytecode, its parameters are ${JSON.stringify(
      thisNode
    )}.`
  )
  if (isLeafNode(thisNode)) {
    bytecode = [
      ...bytecode,
      ...generateLeafBytecode(
        thisNode as LogSearchLeafNode,
        nodeId,
        isLeftSibling
      ),
    ]
  } else {
    const thisInternalNode = thisNode as LogSearchInternalNode
    bytecode = [
      ...bytecode,
      ...generateComparisonBytecode(thisInternalNode, nodeId, isLeftSibling),
    ]
    const leftChildId = thisInternalNode.leftChildNodeId
    const rightChildId = thisInternalNode.rightChildNodeId
    bytecode = appendNodeToBytecode(treeNodes, leftChildId, true, bytecode)
    bytecode = appendNodeToBytecode(treeNodes, rightChildId, false, bytecode)
  }
  return bytecode
}

const generateComparisonBytecode = (
  node: LogSearchInternalNode,
  nodeId: number,
  isLeftSibling: boolean
): EVMBytecode => {
  // if index is odd, add and tag JUMPDEST with "treeNodePosition" = [index, level]
  // For GT check working, add "destinationNodePosition" = index*2+1, level+1
  let bytecodeToReturn: EVMBytecode = []
  const willBeJUMPedTo: boolean = !isLeftSibling
  if (willBeJUMPedTo) {
    bytecodeToReturn.push(generateSearchNodeJumpdest(nodeId))
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
        metadata: { nodeId },
      },
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
  nodeId: number,
  isLeftSibling: boolean
): EVMBytecode => {
  // do the matching stuff
  let bytecodeToReturn: EVMBytecode = []
  const willBeJUMPedTo: boolean = !isLeftSibling
  if (willBeJUMPedTo) {
    bytecodeToReturn.push(generateSearchNodeJumpdest(nodeId))
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

const generateSearchNodeJumpdest = (nodeId: number): EVMOpcodeAndBytes => {
  return {
    opcode: Opcode.JUMPDEST,
    consumedBytes: undefined,
    tag: {
      padPUSH: false,
      reasonTagged: IS_BINARY_SEARCH_NODE_JUMPDEST,
      metadata: { nodeId },
    },
  }
}
