import '../../setup'

/* External Imports */
import { BigNumber, TWO, ZERO } from '@pigi/core-utils'
import {
  MerkleTree,
  MerkleTreeInclusionProof,
  MerkleUpdate,
} from '@pigi/core-db'
import * as assert from 'assert'

/* Internal Imports */
import { MerkleInclusionQuantifier, QuantifierResult } from '../../../src'

class MockedMerkleTree implements MerkleTree {
  private rootHash: Buffer
  private readonly leaves: {} = {}

  public setRootHash(rootHash: Buffer): void {
    this.rootHash = rootHash
  }

  public setLeaf(key: BigNumber, leaf: Buffer): void {
    this.leaves[key.toString()] = leaf
  }

  public async getLeaf(leafKey: BigNumber, rootHash?: Buffer): Promise<Buffer> {
    if (!!rootHash && (!this.rootHash || !this.rootHash.equals(rootHash))) {
      return undefined
    }
    return leafKey.toString() in this.leaves
      ? this.leaves[leafKey.toString()]
      : undefined
  }

  public async getRootHash(): Promise<Buffer> {
    return this.rootHash
  }

  public async update(
    key: BigNumber,
    value: Buffer,
    purgeOldNodes: boolean = true
  ): Promise<boolean> {
    return undefined
  }

  public async getMerkleProof(
    leafKey: BigNumber,
    leafValue: Buffer
  ): Promise<MerkleTreeInclusionProof> {
    return undefined
  }

  public async batchUpdate(updates: MerkleUpdate[]): Promise<boolean> {
    return undefined
  }

  public getHeight(): number {
    return undefined
  }

  public purgeOldNodes(): Promise<void> {
    return
  }
}

describe('MerkleTreeQuantifier', () => {
  describe('getAllQuantified', () => {
    const key: BigNumber = ZERO
    let merkleTree: MockedMerkleTree

    beforeEach(() => {
      merkleTree = new MockedMerkleTree()
    })

    it('returns undefined on empty tree', async () => {
      const quantifier = new MerkleInclusionQuantifier(merkleTree)
      const result: QuantifierResult = await quantifier.getAllQuantified({
        key,
        root: Buffer.from('some root'),
      })

      assert(
        result.results.length === 1,
        'There should only be one result that is undefined.'
      )
      assert(
        result.results[0] === undefined,
        'There should only be one result that is undefined.'
      )
      assert(
        !result.allResultsQuantified,
        'All results should not be quantified if we are unable to find a leaf node in the tree.'
      )
    })

    it('returns undefined on populated tree with a root that does not match', async () => {
      const quantifier = new MerkleInclusionQuantifier(merkleTree)

      merkleTree.setLeaf(key, Buffer.from('VALUE'))

      const result: QuantifierResult = await quantifier.getAllQuantified({
        key,
        root: Buffer.from('some root'),
      })

      assert(
        result.results.length === 1,
        'There should only be one result that is undefined.'
      )
      assert(
        result.results[0] === undefined,
        'There should only be one result that is undefined.'
      )
      assert(
        !result.allResultsQuantified,
        'All results should not be quantified if we are unable to find a leaf node in the tree.'
      )
    })

    it('returns undefined on empty tree with a root that matches', async () => {
      const quantifier = new MerkleInclusionQuantifier(merkleTree)
      const root: Buffer = Buffer.from('some root')

      merkleTree.setRootHash(root)

      const result: QuantifierResult = await quantifier.getAllQuantified({
        key,
        root,
      })

      assert(
        result.results.length === 1,
        'There should only be one result that is undefined.'
      )
      assert(
        result.results[0] === undefined,
        'There should only be one result that is undefined.'
      )
      assert(
        !result.allResultsQuantified,
        'All results should not be quantified if we are unable to find a leaf node in the tree.'
      )
    })

    it('returns value on populated tree tree with a root that matches', async () => {
      const quantifier = new MerkleInclusionQuantifier(merkleTree)
      const root: Buffer = Buffer.from('some root')
      const value: Buffer = Buffer.from('VALUE')

      merkleTree.setRootHash(root)
      merkleTree.setLeaf(key, value)

      const result: QuantifierResult = await quantifier.getAllQuantified({
        key,
        root,
      })

      assert(
        result.results.length === 1,
        'There should only be one result that is the value.'
      )
      assert(
        result.results[0].equals(value),
        `Result value [${result.results[0].toString()}] found when expecting [${value.toString()}]`
      )
      assert(
        result.allResultsQuantified,
        'All results should be quantified if we are able to find the leaf node in the tree.'
      )
    })

    it('returns undefined on populated tree tree with a root that matches but mismatched key', async () => {
      const quantifier = new MerkleInclusionQuantifier(merkleTree)
      const root: Buffer = Buffer.from('some root')
      const value: Buffer = Buffer.from('VALUE')

      merkleTree.setRootHash(root)
      merkleTree.setLeaf(key, value)

      const result: QuantifierResult = await quantifier.getAllQuantified({
        key: TWO,
        root,
      })

      assert(
        result.results.length === 1,
        'There should only be one result that is undefined.'
      )
      assert(
        result.results[0] === undefined,
        'There should only be one result that is undefined.'
      )
      assert(
        !result.allResultsQuantified,
        'All results should not be quantified if we are unable to find a leaf node in the tree.'
      )
    })
  })
})
