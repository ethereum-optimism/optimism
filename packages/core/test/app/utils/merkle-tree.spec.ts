import '../../setup'

import MemDown from 'memdown'
import * as assert from 'assert'

import { BaseDB } from '../../../src/app/db'
import {
  BigNumber,
  keccak256,
  ONE,
  OptimizedSparseMerkleTree,
  THREE,
  TWO,
  ZERO,
} from '../../../src/app/utils'
import { TestUtils } from './test-utils'
import {
  HashFunction,
  MerkleTreeInclusionProof,
  SparseMerkleTree,
} from '../../../src/types/utils'
import { DB } from '../../../src/types'

const hashBuffer: Buffer = new Buffer(64)
const hashFunction: HashFunction = keccak256
const zeroHash: Buffer = hashFunction(OptimizedSparseMerkleTree.emptyBuffer)

const createAndVerifyEmptyTreeDepthWithDepth = async (
  db: DB,
  key: BigNumber,
  depth: number
): Promise<SparseMerkleTree> => {
  const tree: SparseMerkleTree = new OptimizedSparseMerkleTree(
    db,
    undefined,
    depth,
    hashFunction
  )

  let zeroHashParent: Buffer = zeroHash
  const siblings: Buffer[] = []
  for (let i = depth - 2; i >= 0; i--) {
    siblings.push(zeroHashParent)
    zeroHashParent = hashFunction(
      hashBuffer.fill(zeroHashParent, 0, 32).fill(zeroHashParent, 32)
    )
  }

  const inclusionProof: MerkleTreeInclusionProof = {
    key,
    value: OptimizedSparseMerkleTree.emptyBuffer,
    siblings: siblings.reverse(),
  }

  assert(
    await tree.verifyAndStore(inclusionProof),
    'Unable to verify inclusion proof on empty tree when it should be valid.'
  )
  return tree
}

const getRootHashOnlyHashingWithEmptySiblings = (
  leafValue: Buffer,
  key: BigNumber,
  treeHeight: number
): Buffer => {
  let zeroHashParent: Buffer = zeroHash
  let hash: Buffer = hashFunction(leafValue)

  for (let depth = treeHeight - 2; depth >= 0; depth--) {
    const left: boolean = key
      .shiftLeft(depth)
      .shiftRight(treeHeight - 2)
      .mod(TWO)
      .equals(ZERO)
    hash = left
      ? hashFunction(hashBuffer.fill(hash, 0, 32).fill(zeroHashParent, 32))
      : hashFunction(hashBuffer.fill(zeroHashParent, 0, 32).fill(hash, 32))

    zeroHashParent = hashFunction(
      hashBuffer.fill(zeroHashParent, 0, 32).fill(zeroHashParent, 32)
    )
  }

  return hash
}

describe('OptimizedSparseMerkleTree', () => {
  let db: BaseDB
  let memdown: any
  beforeEach(() => {
    memdown = new MemDown('') as any
    db = new BaseDB(memdown, 256)
  })

  afterEach(async () => {
    await Promise.all([db.close()])
    memdown = undefined
  })

  describe('Constructor', () => {
    it('should construct without error', async () => {
      new OptimizedSparseMerkleTree(db)
    })

    it('accepts a non-empty root hash', async () => {
      new OptimizedSparseMerkleTree(db, new Buffer(32).fill('root', 0))
    })

    it('throws if root is not 32 bytes', async () => {
      TestUtils.assertThrows(() => {
        new OptimizedSparseMerkleTree(db, new Buffer(31).fill('root', 0))
      }, assert.AssertionError)
    })

    it('throws if height is < 0', async () => {
      TestUtils.assertThrows(() => {
        new OptimizedSparseMerkleTree(db, undefined, -1)
      }, assert.AssertionError)
    })
  })

  describe('verifyAndStore', () => {
    it('verifies empty root', async () => {
      await createAndVerifyEmptyTreeDepthWithDepth(db, ZERO, 2)
    })

    it('verifies 3-level empty root', async () => {
      await createAndVerifyEmptyTreeDepthWithDepth(db, ZERO, 3)
    })
    it('verifies 4-level empty root', async () => {
      await createAndVerifyEmptyTreeDepthWithDepth(db, ZERO, 4)
    })

    it('verifies empty root with key of 1', async () => {
      await createAndVerifyEmptyTreeDepthWithDepth(db, ONE, 2)
    })

    it('verifies 3-level empty root with key of 1', async () => {
      await createAndVerifyEmptyTreeDepthWithDepth(db, ONE, 3)
    })
    it('verifies 4-level empty root with key of 1', async () => {
      await createAndVerifyEmptyTreeDepthWithDepth(db, ONE, 4)
    })

    it('fails on invalid proof for empty root', async () => {
      const tree: SparseMerkleTree = new OptimizedSparseMerkleTree(
        db,
        undefined,
        2,
        hashFunction
      )

      const inclusionProof: MerkleTreeInclusionProof = {
        key: ZERO,
        value: Buffer.from('this will fail.'),
        siblings: [hashFunction(OptimizedSparseMerkleTree.emptyBuffer)],
      }

      assert(
        !(await tree.verifyAndStore(inclusionProof)),
        'Should have failed on invalid proof for empty root but did not'
      )
    })

    it('verifies non-empty root', async () => {
      const value: Buffer = new Buffer('non-empty')
      const root: Buffer = hashFunction(
        hashBuffer
          .fill(hashFunction(value), 0, 32)
          .fill(hashFunction(OptimizedSparseMerkleTree.emptyBuffer), 32)
      )

      const tree: SparseMerkleTree = new OptimizedSparseMerkleTree(
        db,
        root,
        2,
        hashFunction
      )

      const inclusionProof: MerkleTreeInclusionProof = {
        key: ZERO,
        value,
        siblings: [hashFunction(OptimizedSparseMerkleTree.emptyBuffer)],
      }

      assert(
        await tree.verifyAndStore(inclusionProof),
        'Should have verified non-empty root but did not.'
      )
    })

    it('verifies non-empty root with key of 1', async () => {
      const value: Buffer = new Buffer('non-empty')
      const root: Buffer = hashFunction(
        hashBuffer
          .fill(hashFunction(OptimizedSparseMerkleTree.emptyBuffer), 0, 32)
          .fill(hashFunction(value), 32)
      )

      const tree: SparseMerkleTree = new OptimizedSparseMerkleTree(
        db,
        root,
        2,
        hashFunction
      )

      const inclusionProof: MerkleTreeInclusionProof = {
        key: ONE,
        value,
        siblings: [hashFunction(OptimizedSparseMerkleTree.emptyBuffer)],
      }

      assert(
        await tree.verifyAndStore(inclusionProof),
        'Should have verified non-empty root but did not.'
      )
    })

    it('fails verifying invalid non-empty root', async () => {
      const value: Buffer = new Buffer('non-empty')
      const root: Buffer = hashFunction(
        hashBuffer
          .fill(hashFunction(value), 0, 32)
          .fill(hashFunction(OptimizedSparseMerkleTree.emptyBuffer), 32)
      )

      const tree: SparseMerkleTree = new OptimizedSparseMerkleTree(
        db,
        root,
        2,
        hashFunction
      )

      const inclusionProof: MerkleTreeInclusionProof = {
        key: ZERO,
        value: Buffer.from('not the right value'),
        siblings: [hashFunction(OptimizedSparseMerkleTree.emptyBuffer)],
      }

      assert(
        !(await tree.verifyAndStore(inclusionProof)),
        'Did not fail when verifying an invalid non-zero root.'
      )
    })
  })

  describe('update', () => {
    it('updates empty tree', async () => {
      const tree: SparseMerkleTree = await createAndVerifyEmptyTreeDepthWithDepth(
        db,
        ZERO,
        3
      )

      const value: Buffer = Buffer.from('much better value')
      assert(await tree.update(ZERO, value))

      const root: Buffer = getRootHashOnlyHashingWithEmptySiblings(
        value,
        ZERO,
        3
      )
      assert(
        root.equals(await tree.getRootHash()),
        'Root hashes do not match after update'
      )
    })

    it('updates empty tree at key 1', async () => {
      const tree: SparseMerkleTree = await createAndVerifyEmptyTreeDepthWithDepth(
        db,
        ONE,
        3
      )

      const value: Buffer = Buffer.from('much better value')
      assert(await tree.update(ONE, value))

      const root: Buffer = getRootHashOnlyHashingWithEmptySiblings(
        value,
        ONE,
        3
      )
      assert(
        root.equals(await tree.getRootHash()),
        'Root hashes do not match after update'
      )
    })

    it('updates empty tree at key 2', async () => {
      const tree: SparseMerkleTree = await createAndVerifyEmptyTreeDepthWithDepth(
        db,
        TWO,
        3
      )

      const value: Buffer = Buffer.from('much better value')
      assert(await tree.update(TWO, value))

      const root: Buffer = getRootHashOnlyHashingWithEmptySiblings(
        value,
        TWO,
        3
      )
      assert(
        root.equals(await tree.getRootHash()),
        'Root hashes do not match after update'
      )
    })

    it('updates empty tree at key 3', async () => {
      const tree: SparseMerkleTree = await createAndVerifyEmptyTreeDepthWithDepth(
        db,
        THREE,
        3
      )

      const value: Buffer = Buffer.from('much better value')
      assert(await tree.update(THREE, value))

      const root: Buffer = getRootHashOnlyHashingWithEmptySiblings(
        value,
        THREE,
        3
      )
      assert(
        root.equals(await tree.getRootHash()),
        'Root hashes do not match after update'
      )
    })

    it('updates empty tree at key 0 and 1', async () => {
      /*
              zh                    C                  F
             /  \                 /  \              /    \
           zh    zh     ->      B    zh     ->     E      zh
          /  \  /  \           /  \  /  \        /  \    /  \
        zh  zh  zh  zh        A  zh  zh  zh     A    D  zh  zh
      */

      const tree: SparseMerkleTree = await createAndVerifyEmptyTreeDepthWithDepth(
        db,
        ZERO,
        3
      )

      const value: Buffer = Buffer.from('much better value')
      const valueHash: Buffer = hashFunction(value)
      assert(await tree.update(ZERO, value))

      const root: Buffer = getRootHashOnlyHashingWithEmptySiblings(
        value,
        ZERO,
        3
      )
      assert(
        root.equals(await tree.getRootHash()),
        'Root hashes do not match after update'
      )

      // VERIFY AND UPDATE ONE

      // first sibling is other value, next is zero hash because parent's sibling tree is empty
      const siblings: Buffer[] = [valueHash]
      const zeroHashParent: Buffer = hashFunction(
        hashBuffer.fill(zeroHash, 0, 32).fill(zeroHash, 32)
      )
      siblings.push(zeroHashParent)

      const inclusionProof: MerkleTreeInclusionProof = {
        key: ONE,
        value: OptimizedSparseMerkleTree.emptyBuffer,
        siblings: siblings.reverse(),
      }

      assert(await tree.verifyAndStore(inclusionProof))

      const secondValue: Buffer = Buffer.from('much better value 2')
      const secondValueHash: Buffer = hashFunction(secondValue)

      assert(await tree.update(ONE, secondValue))

      let parentHash: Buffer = hashFunction(
        hashBuffer.fill(valueHash, 0, 32).fill(secondValueHash, 32)
      )
      parentHash = hashFunction(
        hashBuffer.fill(parentHash, 0, 32).fill(zeroHashParent, 32)
      )

      assert(
        parentHash.equals(await tree.getRootHash()),
        'Root hashes do not match after update'
      )
    })

    it('updates empty tree at key 0 and 2', async () => {
      /*
              zh                    C                  F
             /  \                 /  \              /    \
           zh    zh     ->      B    zh     ->     B      E
          /  \  /  \           /  \  /  \        /  \    /  \
        zh  zh  zh  zh        A  zh  zh  zh     A    zh  D  zh
      */

      const tree: SparseMerkleTree = await createAndVerifyEmptyTreeDepthWithDepth(
        db,
        ZERO,
        3
      )

      const value: Buffer = Buffer.from('much better value')
      const valueHash: Buffer = hashFunction(value)
      assert(await tree.update(ZERO, value))

      const root: Buffer = getRootHashOnlyHashingWithEmptySiblings(
        value,
        ZERO,
        3
      )
      assert(
        root.equals(await tree.getRootHash()),
        'Root hashes do not match after update'
      )

      // VERIFY AND UPDATE TWO

      const leftSubtreeSibling: Buffer = hashFunction(
        hashBuffer.fill(valueHash, 0, 32).fill(zeroHash, 32)
      )
      const siblings: Buffer[] = [zeroHash, leftSubtreeSibling]

      const inclusionProof: MerkleTreeInclusionProof = {
        key: TWO,
        value: OptimizedSparseMerkleTree.emptyBuffer,
        siblings: siblings.reverse(),
      }

      assert(await tree.verifyAndStore(inclusionProof))

      const secondValue: Buffer = Buffer.from('much better value 2')
      const secondValueHash: Buffer = hashFunction(secondValue)

      assert(await tree.update(TWO, secondValue))

      let parentHash: Buffer = hashFunction(
        hashBuffer.fill(secondValueHash, 0, 32).fill(zeroHash, 32)
      )
      parentHash = hashFunction(
        hashBuffer.fill(leftSubtreeSibling, 0, 32).fill(parentHash, 32)
      )

      assert(
        parentHash.equals(await tree.getRootHash()),
        'Root hashes do not match after update'
      )
    })
  })
})
