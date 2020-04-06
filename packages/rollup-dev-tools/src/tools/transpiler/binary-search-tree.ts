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
  BinarySearchInternalNode,
  BinarySearchLeafNode,
  BinarySearchNode,
} from '../../types/transpiler'

const log = getLogger('binary-search-generator')

// The max number of bytes we expect a JUMPDEST's PC to be expressible in.  Setting to 3 allows 16 MB contracts--more than enough!
const maxBytesOfContractSize = 3

// reasonTagged label for opcode PUSHing the PC of a binary search tree node to the stack
const IS_PUSH_BINARY_SEARCH_NODE_LOCATION =
  'IS_PUSH_BINARY_SEARCH_NODE_LOCATION'
// reasonTagged label for JUMPDEST of a binary searchh tree node.
const IS_BINARY_SEARCH_NODE_JUMPDEST = 'IS_BINARY_SEARCH_NODE_JUMPDEST'

/**
 * Generates a binary search tree based on a set of keys to search against, and values associated with each key.
 *
 * @param keys The (lowest-to-highest) ordered array of keys to be searched for.
 * @param values The array of corresponding values to return.
 * @returns A flat array of nodes representing the tree, with the final node in the array being the root.
 */
export const generateLogSearchTreeNodes = (
  keys: number[],
  values: number[]
): BinarySearchNode[] => {
  // initialize our array of all tree nodes, populating first with all the leaves
  const allTreeNodes: BinarySearchNode[] = keys.map((v, i) => {
    return {
      key: keys[i],
      value: values[i],
      nodeId: i,
    }
  })

  // We process the tree in levels, with the parents of the current level becoming the next level
  // If the current level is length 1, it is the root and we have generated the full tree.
  let curLevel: BinarySearchNode[] = [...allTreeNodes]
  while (curLevel.length > 1) {
    const nextLevel: BinarySearchNode[] = []
    // Iterate over every other node in the level to compute parent nodes
    for (let i = 0; i < curLevel.length; i += 2) {
      const leftChild = curLevel[i]
      const rightChild = curLevel[i + 1]
      // If there is no right child, push the left child up to the next level so that the tree is a full binary tree.
      if (!rightChild) {
        nextLevel.push(leftChild)
        continue
      }

      // Now we calculate the parent for these children
      const parentNode: BinarySearchInternalNode = {
        nodeId: allTreeNodes.length, // nodeId is set as the position in the returned array
        leftChildNodeId: leftChild.nodeId,
        rightChildNodeId: rightChild.nodeId,
        largestAncestorKey: undefined,
        keyToCompare: undefined,
      }

      // The largest ancestor key is the right child's largest ancestor or its key if a leaf
      parentNode.largestAncestorKey = isLeafNode(rightChild)
        ? (rightChild as BinarySearchLeafNode).key
        : (rightChild as BinarySearchInternalNode).largestAncestorKey

      // The decision to go to the left or right sibling in a search is a comparison with the largest key in the left child's ancestors.
      parentNode.keyToCompare = isLeafNode(leftChild)
        ? (leftChild as BinarySearchLeafNode).key
        : (parentNode.keyToCompare = (leftChild as BinarySearchInternalNode).largestAncestorKey)

      // Add parent to the tree so it will be returned
      allTreeNodes.push(parentNode)
      // Add parent to the next level so its parent can be processed
      nextLevel.push(parentNode)
    }
    curLevel = nextLevel
  }
  return allTreeNodes
}

const isLeafNode = (node: BinarySearchNode): boolean => {
  return !!(node as BinarySearchLeafNode).value
}

/**
 * Generates the bytecode which will compute a binary search comparing jumpdestIndexesBefore to the
 * top stack element, and JUMPs to tohe corresponding element of jumpdestIndexesAfter.
 *
 * @param jumpdestIndexesBefore The (lowest-to-highest) ordered array of jump indexes expected as stack inputs.
 * @param values The array of corresponding PCs to JUMP to based on the stack input
 * @param indexOfThisBlock The offset of this block in the bytecode where the result will be inserted
 * @returns The EVM Bytecode performing the binary-search-and-JUMP operation.
 */
export const getJumpIndexSearchBytecode = (
  jumpdestIndexesBefore: number[],
  jumpdestIndexesAfter: number[],
  indexOfThisBlock: number
): EVMBytecode => {
  // Generate a conceptual binary search tree for these jump indexes.
  const searchTreeNodes: BinarySearchNode[] = generateLogSearchTreeNodes(
    jumpdestIndexesBefore,
    jumpdestIndexesAfter
  )
  log.debug(
    `successfully generated conceptual log search tree, its flat structure is: \n${JSON.stringify(
      searchTreeNodes
    )}`
  )
  // The root node is always the final element rerturned by generateLogSearchTreeNodes()
  const rootNodeId = searchTreeNodes.length - 1

  const bytecode: EVMBytecode = [
    {
      opcode: Opcode.JUMPDEST,
      consumedBytes: undefined,
    },
    ...appendAncestorsToBytecode(searchTreeNodes, rootNodeId, true, []), // Recursively fills out the bytecode based on the tree.
  ]
  // Fix the PUSH (jumpdests) to be correct once the tree has been generated.
  const finalBytecode = fixJUMPsToNodes(
    bytecode,
    indexOfThisBlock,
    searchTreeNodes
  )
  log.debug(
    `Generated final bytecode for log search jump table : \n${formatBytecode(
      finalBytecode
    )}`
  )
  return finalBytecode
}

/**
 * Recursively appends to the given bytecode all ancestors for the given NodeId.
 *
 * @param treeNodes The binary search tree representing the nodes to process into bytecode
 * @param nodeId The nodeId to process all ancestors of.
 * @param isLeftSibling Whether the given nodeId is a left or right sibling (if a right sibling, it will be JUMPed to.)
 * @param bytecode The existing EVM bytecode to append to.
 * @returns The EVM Bytecode with all ancestors of the given nodeId having been added.
 */
const appendAncestorsToBytecode = (
  treeNodes: BinarySearchNode[],
  nodeId: number,
  isLeftSibling: boolean,
  bytecode: EVMBytecode
): EVMBytecode => {
  const thisNode: BinarySearchNode = treeNodes[nodeId]
  log.info(
    `Processing node with Id ${nodeId} of tree into bytecode, its parameters are ${JSON.stringify(
      thisNode
    )}.`
  )
  if (isLeafNode(thisNode)) {
    // If this is a leaf node, we can append the leaf and return since no ancestors.
    bytecode = [
      ...bytecode,
      ...generateLeafBytecode(
        thisNode as BinarySearchLeafNode,
        nodeId,
        isLeftSibling
      ),
    ]
  } else {
    // Otherwise, we process and append the left and right siblings, recursively using this function.
    // Append this node
    const thisInternalNode = thisNode as BinarySearchInternalNode
    bytecode = [
      ...bytecode,
      ...generateInternalNodeBytecode(thisInternalNode, nodeId, isLeftSibling),
    ]
    // Append left and right children (left first--right siblings are JUMPed to, left siblings not!)
    const leftChildId = thisInternalNode.leftChildNodeId
    const rightChildId = thisInternalNode.rightChildNodeId
    bytecode = appendAncestorsToBytecode(treeNodes, leftChildId, true, bytecode)
    bytecode = appendAncestorsToBytecode(
      treeNodes,
      rightChildId,
      false,
      bytecode
    )
  }
  return bytecode
}

/**
 * Fixes all PUSHes tagged by appendAncestorsToBytecode() to correspond to the correct nodes' JUMPDESTs
 *
 * @param bytecode The tagged bytecode of the jump search table with incorrect PUSHes for the jumnpdests
 * @param indexOfThisBlock The offset of this block in the bytecode where the result will be inserted
 * @param searchTreeNodes All of the binary nodes from which the bytecode was generated
 * @returns The EVM Bytecode with fixed PUSHes.
 */
const fixJUMPsToNodes = (
  bytecode: EVMBytecode,
  indexOfThisBlock: number,
  searchTreeNodes: BinarySearchNode[]
): EVMBytecode => {
  for (const opcodeAndBytes of bytecode) {
    // Find all the PUSHes which we need to append the right sibling's JUMPDEST location to.
    if (
      !!opcodeAndBytes.tag &&
      opcodeAndBytes.tag.reasonTagged === IS_PUSH_BINARY_SEARCH_NODE_LOCATION
    ) {
      // Get this node and the node it should JUMP to from the input tree
      const thisNodeId = opcodeAndBytes.tag.metadata.nodeId
      const rightChildNodeId = (searchTreeNodes[
        thisNodeId
      ] as BinarySearchInternalNode).rightChildNodeId
      // Find the index of the right child's JUMPDEST in the bytecode.
      const rightChildJumpdestIndexInBytecodeBlock = bytecode.findIndex(
        (toCheck: EVMOpcodeAndBytes) => {
          return (
            !!toCheck.tag &&
            toCheck.tag.reasonTagged === IS_BINARY_SEARCH_NODE_JUMPDEST &&
            toCheck.tag.metadata.nodeId === rightChildNodeId
          )
        }
      )
      // Calculate the PC of the found JUMPDEST, offsetting by the index of this block.
      const rightChildJumpdestPC =
        indexOfThisBlock +
        getPCOfEVMBytecodeIndex(
          rightChildJumpdestIndexInBytecodeBlock,
          bytecode
        )
      // Set the consumed bytes to be this PC
      opcodeAndBytes.consumedBytes = bufferUtils.numberToBuffer(
        rightChildJumpdestPC,
        maxBytesOfContractSize,
        maxBytesOfContractSize
      )
    }
  }
  return bytecode
}

/**
 * Generates bytecode for an internal binary search node, which either JUMPs to its right child
 * if the input is greater than its valueToCompare, or continues to its left child which is immediately below.
 *
 * @param node The internal node to process
 * @param nodeId The nodeId of this node (used for tagging for later correction)
 * @param isLeftSibling Whether this is a left sibling which will be continued to, or a right sibling which will be JUMPed to.
 * @returns The tagged block of EVM Bytecode for this internal node.
 */
const generateInternalNodeBytecode = (
  node: BinarySearchInternalNode,
  nodeId: number,
  isLeftSibling: boolean
): EVMBytecode => {
  let bytecodeToReturn: EVMBytecode = []
  const willBeJUMPedTo: boolean = !isLeftSibling
  if (willBeJUMPedTo) {
    bytecodeToReturn.push(generateSearchNodeJumpdest(nodeId))
  }

  bytecodeToReturn = [
    ...bytecodeToReturn,
    // DUP the input to compare
    {
      opcode: Opcode.DUP1,
      consumedBytes: undefined,
    },
    // PUSH the key to be compared to determine which node to proceed to
    getPUSHIntegerOp(node.keyToCompare),
    // Compare the keys
    {
      opcode: Opcode.LT,
      consumedBytes: undefined,
    },
    // PUSH a *placeholder* for the destination of thde right child to be JUMPed to if check passes--to be set later
    {
      opcode: getPUSHOpcode(maxBytesOfContractSize),
      consumedBytes: Buffer.alloc(maxBytesOfContractSize),
      tag: {
        padPUSH: false,
        reasonTagged: IS_PUSH_BINARY_SEARCH_NODE_LOCATION,
        metadata: { nodeId },
      },
    },
    // JUMP if the LT check passed
    {
      opcode: Opcode.JUMPI,
      consumedBytes: undefined,
    },
  ]

  return bytecodeToReturn
}

/**
 * Generates bytecode for binary search leaf node -- if we're here, the internal nodes have fully sorted the input
 *
 * @param node The leaf node to process
 * @param nodeId The nodeId of this node (used for tagging for later correction)
 * @param isLeftSibling Whether this is a left sibling which will be continued to, or a right sibling which will be JUMPed to.
 * @returns The tagged block of EVM Bytecode for this internal node.
 */
const generateLeafBytecode = (
  node: BinarySearchLeafNode,
  nodeId: number,
  isLeftSibling: boolean
): EVMBytecode => {
  let bytecodeToReturn: EVMBytecode = []
  const willBeJUMPedTo: boolean = !isLeftSibling
  if (willBeJUMPedTo) {
    bytecodeToReturn.push(generateSearchNodeJumpdest(nodeId))
  }

  bytecodeToReturn = [
    ...bytecodeToReturn,
    // We have found a match for the input so it's no longer needed--POP it.
    {
      opcode: Opcode.POP,
      consumedBytes: undefined,
    },
    // JUMP to the jumpdestIndexAfter.
    getPUSHIntegerOp(node.value),
    {
      opcode: Opcode.JUMP,
      consumedBytes: undefined,
    },
  ]
  return bytecodeToReturn
}

/**
 * Generates a tagged JUMPDEST for a binary search node
 *
 * @param nodeId The nodeId of this node (used for tagging for later correction)
 * @returns The correctly tagged JUMPDEST.
 */
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
