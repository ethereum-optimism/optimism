import {
  bytecodeToBuffer,
  EVMBytecode,
  Opcode,
  EVMOpcodeAndBytes,
  formatBytecode,
  getPCOfEVMBytecodeIndex,
  OpcodeTagReason,
} from '@eth-optimism/rollup-core'
import { bufferUtils, getLogger } from '@eth-optimism/core-utils'
import { getPUSHOpcode, getPUSHIntegerOp } from './helpers'
import { BinarySearchTreeNode } from '../../types/transpiler'

const log = getLogger('binary-search-tree-generator')

// The max number of bytes we expect a JUMPDEST's PC to be expressible in.  Setting to 3 allows 16 MB contracts--more than enough!
const pcMaxByteSize = 3

/**
 * Generates a JUMP-correctiing binary search tree block which is used to map pre-transpiled JUMPDESTs too post-transpiled JUMPDESTs.
 *
 * @param keys The jumpdest PCs before transpilation occurred
 * @param values The jumpdest PCs after transpilation occurred.
 * @param indexOfThisBlock the PC that this block of bytecode will be placed in.
 * @returns Bytecode block of the JUMPDEST-mapping functionality
 */

export const buildJumpBSTBytecode = (
  keys: number[],
  values: number[],
  indexOfThisBlock: number
): EVMBytecode => {
  if (keys.length !== values.length) {
    throw new Error(
      `Asked to build binary search tree, but given key array of length ${keys.length} and value array of length ${values.length}`
    )
  }
  const rootNode = getBSTRoot(keys, values)
  const BSTBytecodeWithIncorrectJUMPs: EVMBytecode = [
    // Bytecode to JUMP to when a successful match is found by a BST node
    ...getJumpdestMatchSuccessBytecode(),
    // Entry point for the actual BST matching logic
    {
      opcode: Opcode.JUMPDEST,
      consumedBytes: undefined,
    },
    // Bytecode for the "root subtree" recursively generates all BST logic as bytecode.
    ...getBytecodeForSubtreeRoot(rootNode, false),
  ]
  return fixJUMPsToNodes(BSTBytecodeWithIncorrectJUMPs, indexOfThisBlock)
}

/**
 * Generates a binary search tree based on a set of keys to search against, and values associated with each key.
 *
 * @param keys The (lowest-to-highest) ordered array of keys to be searched for.
 * @param values The array of corresponding values to return.
 * @returns A root BST node whose ancestors represent the full BST for the given k/v pairs.
 */
export const getBSTRoot = (keys: number[], values: number[]) => {
  // Associate nodes with k/v pairs and sort before building the tree
  const bottomNodes: BinarySearchTreeNode[] = keys.map(
    (key: number, index: number) => {
      return {
        value: {
          jumpdestBefore: keys[index],
          jumpdestAfter: values[index],
        },
        left: undefined,
        right: undefined,
      }
    }
  )
  const sortedBottomNodes = bottomNodes.sort(
    (node1: BinarySearchTreeNode, node2: BinarySearchTreeNode) => {
      return node1.value.jumpdestBefore - node2.value.jumpdestBefore
    }
  )
  return buildBST(sortedBottomNodes)
}

/**
 * Generates a binary search tree from a sorted list of values.
 * @param sortedValues The sorted BST nodes to be searched for
 * @returns The root node of the resulting BST.
 */
export const buildBST = (
  sortedAncestors: BinarySearchTreeNode[]
): BinarySearchTreeNode => {
  if (sortedAncestors.length === 0) {
    return undefined
  }

  const rootIndex = Math.floor(sortedAncestors.length / 2)
  const leftSubtreeElements: BinarySearchTreeNode[] = sortedAncestors.slice(
    0,
    rootIndex
  )
  const rightSubtreeElements: BinarySearchTreeNode[] = sortedAncestors.slice(
    rootIndex + 1
  )
  return {
    value: sortedAncestors[rootIndex].value,
    left: buildBST(leftSubtreeElements),
    right: buildBST(rightSubtreeElements),
  }
}

/**
 * Generates bytecode executing a binary search tree for a given node's subtree.
 * Recursively executes with left->right, depth-first approach.
 * @param node The BST nodeto
 * @returns The root node of the resulting BST.
 */
const getBytecodeForSubtreeRoot = (
  node: BinarySearchTreeNode,
  isRightNode: boolean
): EVMBytecode => {
  if (!node) {
    return []
  }
  const bytecodeToReturn: EVMBytecode = []
  // Left->right, depth first makes it so that right nodes are always JUMPed to, and left nodes are continued to from the parent node with no JUMP.
  if (isRightNode) {
    bytecodeToReturn.push(generateBinarySearchTreeNodeJumpdest(node))
  }
  // Generate the match check for this node
  bytecodeToReturn.push(...generateNodeEqualityCheckBytecode(node))
  // If there are no children to continue to, there is definitely no match--STOP as this was an invalid JUMP according to pre-transpilation JUMPDESTs
  if (!node.left && !node.right) {
    bytecodeToReturn.push({
      opcode: Opcode.STOP,
      consumedBytes: undefined,
    })
    return bytecodeToReturn
  }
  // If this node has a right child, check whether the stack input is greater than this node value and JUMP there.  Otherwise we will continue to the left.
  if (node.right) {
    bytecodeToReturn.push(
      ...generateIfGreaterThenJumpToRightChildBytecode(node)
    )
  }
  // generate bytecode for the next subtree enforcing left->right depth first execution so that every left sibling is continued to, every right sibling JUMPed to
  bytecodeToReturn.push(
    ...getBytecodeForSubtreeRoot(node.left, false),
    ...getBytecodeForSubtreeRoot(node.right, true)
  )
  return bytecodeToReturn
}

/**
 * Generates a bytecode block that checks if the current stack element is >= this node's value, and jumping to the right child node if so.
 *
 * @param node The BST node being inequality checked
 * @returns The correctly tagged bytecode jumping to the right child of this node if needed.
 */
const generateIfGreaterThenJumpToRightChildBytecode = (
  node: BinarySearchTreeNode
): EVMBytecode => {
  return [
    {
      opcode: Opcode.DUP1,
      consumedBytes: undefined,
    },
    // PUSH the key to be compared to determine which node to proceed to
    getPUSHIntegerOp(node.value.jumpdestBefore),
    // Compare the keys
    {
      opcode: Opcode.LT,
      consumedBytes: undefined,
    },
    // PUSH a *placeholder* for the destination of thde right child to be JUMPed to if check passes--to be set later
    {
      opcode: getPUSHOpcode(pcMaxByteSize),
      consumedBytes: Buffer.alloc(pcMaxByteSize),
      tag: {
        padPUSH: false,
        reasonTagged: OpcodeTagReason.IS_PUSH_BINARY_SEARCH_NODE_LOCATION,
        metadata: { node },
      },
    },
    // JUMP if the LT check passed
    {
      opcode: Opcode.JUMPI,
      consumedBytes: undefined,
    },
  ]
}

/**
 * Generates a bytecode block that checks for equality of the stack element to the node, and jumping to a match success block if so.
 *
 * @param node The BST node being equality checked
 * @returns The correctly tagged bytecode jumping to the match success case for this node.
 */
const generateNodeEqualityCheckBytecode = (
  node: BinarySearchTreeNode
): EVMBytecode => {
  return [
    // DUP the value to match without deleting it forever
    {
      opcode: Opcode.DUP1,
      consumedBytes: undefined,
    },
    // Compare to JUMPDEST before
    getPUSHIntegerOp(node.value.jumpdestBefore),
    {
      opcode: Opcode.EQ,
      consumedBytes: undefined,
    },
    // If match, we will send the JUMPDEST after to the success block
    getPUSHIntegerOp(node.value.jumpdestAfter),
    {
      opcode: Opcode.SWAP1,
      consumedBytes: undefined,
    },
    // PUSH success block location (via a tag--will be filled out later)
    {
      opcode: getPUSHOpcode(pcMaxByteSize),
      consumedBytes: Buffer.alloc(pcMaxByteSize),
      tag: {
        padPUSH: false,
        reasonTagged: OpcodeTagReason.IS_PUSH_MATCH_SUCCESS_LOC,
        metadata: undefined,
      },
    },
    // JUMPI to success block if match
    {
      opcode: Opcode.JUMPI,
      consumedBytes: undefined,
    },
    // POP the JUMPDESTafter if not a match.
    {
      opcode: Opcode.POP,
      consumedBytes: undefined,
    },
  ]
}

/**
 * Generates a tagged JUMPDEST for a binary search node
 *
 * @param node The BST node being jumped to (used for tagging for later correction)
 * @returns The correctly tagged JUMPDEST.
 */
const generateBinarySearchTreeNodeJumpdest = (
  node: BinarySearchTreeNode
): EVMOpcodeAndBytes => {
  return {
    opcode: Opcode.JUMPDEST,
    consumedBytes: undefined,
    tag: {
      padPUSH: false,
      reasonTagged: OpcodeTagReason.IS_BINARY_SEARCH_NODE_JUMPDEST,
      metadata: { node },
    },
  }
}

/**
 * Gets the success jumpdest for the footer switch statement. This will be jumped to when the
 * switch statement finds a match. It is responsible for getting rid of extra stack arguments
 * that the footer switch statement adds.
 *
 * @returns The success bytecode.
 */

export const getJumpdestMatchSuccessBytecode = (): EVMBytecode => {
  return [
    // This JUMPDEST is hit on successful switch match
    { opcode: Opcode.JUMPDEST, consumedBytes: undefined },
    // Swaps the duped pre-transpilation JUMPDEST with the post-transpilation JUMPDEST
    { opcode: Opcode.SWAP1, consumedBytes: undefined },
    // Pops the pre-transpilation JUMPDEST
    { opcode: Opcode.POP, consumedBytes: undefined },
    // Jumps to the post-transpilation JUMPDEST
    { opcode: Opcode.JUMP, consumedBytes: undefined },
  ]
}

/**
 * Fixes all PUSHes tagged by appendAncestorsToBytecode() to correspond to the correct nodes' JUMPDESTs
 *
 * @param bytecode The tagged bytecode of the jump search table with incorrect PUSHes for the jumnpdests
 * @param indexOfThisBlock The offset of this block in the bytecode where the result will be inserted
 * @returns The EVM Bytecode with fixed PUSHes.
 */
const fixJUMPsToNodes = (
  bytecode: EVMBytecode,
  indexOfThisBlock: number
): EVMBytecode => {
  for (const pushMatchSuccessOp of bytecode.filter(
    (x) =>
      !!x.tag &&
      x.tag.reasonTagged === OpcodeTagReason.IS_PUSH_MATCH_SUCCESS_LOC
  )) {
    pushMatchSuccessOp.consumedBytes = bufferUtils.numberToBuffer(
      indexOfThisBlock,
      pcMaxByteSize,
      pcMaxByteSize
    )
  }
  for (const pushBSTNodeOp of bytecode.filter(
    (x) =>
      !!x.tag &&
      x.tag.reasonTagged === OpcodeTagReason.IS_PUSH_BINARY_SEARCH_NODE_LOCATION
  )) {
    const rightChild: BinarySearchTreeNode =
      pushBSTNodeOp.tag.metadata.node.right
    // Find the index of the right child's JUMPDEST in the bytecode, for each node.
    const rightChildJumpdestIndexInBytecodeBlock = bytecode.findIndex(
      (toCheck: EVMOpcodeAndBytes) => {
        return (
          !!toCheck.tag &&
          toCheck.tag.reasonTagged ===
            OpcodeTagReason.IS_BINARY_SEARCH_NODE_JUMPDEST &&
          toCheck.tag.metadata.node === rightChild
        )
      }
    )
    // Calculate the PC of the found JUMPDEST, offsetting by the index of this block.
    const rightChildJumpdestPC =
      indexOfThisBlock +
      getPCOfEVMBytecodeIndex(rightChildJumpdestIndexInBytecodeBlock, bytecode)
    // Set the consumed bytes to be this PC
    pushBSTNodeOp.consumedBytes = bufferUtils.numberToBuffer(
      rightChildJumpdestPC,
      pcMaxByteSize,
      pcMaxByteSize
    )
  }
  return bytecode
}
