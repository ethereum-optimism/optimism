/* Internal Imports */

/* External Imports */
import {
  RollupTransition,
  SwapTransition,
  TransferTransition,
  CreateAndTransferTransition,
  abiEncodeTransition,
} from '@pigi/wallet'
import {
  keccak256,
  hexStrToBuf,
  bufToHexString,
  BigNumber,
  BaseDB,
  SparseMerkleTreeImpl,
} from '@pigi/core'
import MemDown from 'memdown'

interface TransitionInclusionProof {
  blockNumber: number
  transitionIndex: number
  path: string
  siblings: string[]
}

interface IncludedTransition {
  transition: string
  inclusionProof: TransitionInclusionProof
}

/*
 * Helper class which provides all information requried for a particular
 * Rollup block. This includes all of the tranisitions in readable form
 * as well as the merkle tree which it generates.
 */
export class RollupBlock {
  public transitions: RollupTransition[]
  public encodedTransitions: string[]
  public blockNumber: number
  public tree: SparseMerkleTreeImpl

  constructor(transitions: RollupTransition[], blockNumber: number) {
    this.transitions = transitions
    this.encodedTransitions = transitions.map((transition) =>
      abiEncodeTransition(transition)
    )
    this.blockNumber = blockNumber
  }

  public async generateTree(): Promise<void> {
    // Create a tree!
    const treeHeight = Math.ceil(Math.log2(this.transitions.length)) + 1 // The height should actually not be plus 1
    this.tree = await SparseMerkleTreeImpl.create(
      new BaseDB(new MemDown('') as any, 256),
      undefined,
      treeHeight
    )
    for (let i = 0; i < this.encodedTransitions.length; i++) {
      await this.tree.update(
        new BigNumber(i, 10),
        hexStrToBuf(this.encodedTransitions[i])
      )
    }
  }

  public async getIncludedTransition(
    transitionIndex: number
  ): Promise<IncludedTransition> {
    const inclusionProof = await this.getInclusionProof(transitionIndex)
    return {
      transition: this.encodedTransitions[transitionIndex],
      inclusionProof,
    }
  }

  public async getInclusionProof(
    transitionIndex: number
  ): Promise<TransitionInclusionProof> {
    const blockInclusion = await this.tree.getMerkleProof(
      new BigNumber(transitionIndex),
      hexStrToBuf(this.encodedTransitions[transitionIndex])
    )
    const path = bufToHexString(blockInclusion.key.toBuffer('B', 32))
    const siblings = blockInclusion.siblings.map((sibBuf) =>
      bufToHexString(sibBuf)
    )
    return {
      blockNumber: this.blockNumber,
      transitionIndex,
      path,
      siblings,
    }
  }
}
