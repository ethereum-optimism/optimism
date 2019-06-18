import { should } from '../../../setup'

/* External Imports */
import BigNum = require('bn.js')

/* Internal Imports */
import { MerkleSumTree, MerkleTreeNode } from '../../../../src/app/common/utils'

describe('MerkleSumTree', () => {
  describe('construction', () => {
    it('should be created correctly with no leaves', () => {
      const tree = new MerkleSumTree()

      should.not.exist(tree.root)
      tree.leaves.should.deep.equal([])
      tree.levels.should.deep.equal([[]])
    })

    it('should be created correctly with one leaf', () => {
      const leaves: MerkleTreeNode[] = [
        {
          end: new BigNum(100),
          data: '0x123',
        },
      ]
      const tree = new MerkleSumTree({
        leaves,
      })

      tree.root.should.equal(
        '0x9b64c516a177f40236cfa25c63deb358d93081decbcac3a99dbdcfe856f1d20b'
      )
    })

    it('should be created correctly with two leaves', () => {
      const leaves: MerkleTreeNode[] = [
        {
          end: new BigNum(100),
          data: '0x123',
        },
        {
          end: new BigNum(200),
          data: '0x456',
        },
      ]
      const tree = new MerkleSumTree({
        leaves,
      })

      tree.root.should.equal(
        '0xec76338e61e80c68c487626bf8c793d88f189f553af34ca9d5683c2a1e81a9f5'
      )
    })
  })

  describe('verify', () => {
    it('should correctly verify a valid proof', () => {
      const tree = new MerkleSumTree()
      const leaf: MerkleTreeNode = {
        end: new BigNum(100),
        data: '0x123',
      }
      const inclusionProof: MerkleTreeNode[] = [
        {
          end: new BigNum(200),
          data:
            '0x2d8f2d36584051e513680eb7387c21fab7f2511d711694ada0674d669d89022d',
        },
      ]
      const root =
        '0xec76338e61e80c68c487626bf8c793d88f189f553af34ca9d5683c2a1e81a9f5'

      should.not.Throw(() => {
        tree.verify(leaf, 0, inclusionProof, root)
      })
    })

    it('should correctly reject a proof with the wrong root', () => {
      const tree = new MerkleSumTree()
      const leaf: MerkleTreeNode = {
        end: new BigNum(100),
        data: '0x123',
      }
      const inclusionProof: MerkleTreeNode[] = [
        {
          end: new BigNum(200),
          data:
            '0x2d8f2d36584051e513680eb7387c21fab7f2511d711694ada0674d669d89022d',
        },
      ]
      const root = '0x000000'

      should.Throw(() => {
        tree.verify(leaf, 0, inclusionProof, root)
      }, 'Invalid Merkle Sum Tree proof.')
    })

    it('should correctly reject a proof with the wrong siblings', () => {
      const tree = new MerkleSumTree()
      const leaf: MerkleTreeNode = {
        end: new BigNum(100),
        data: '0x123',
      }
      const inclusionProof: MerkleTreeNode[] = [
        {
          end: new BigNum(200),
          data: '0x00000000',
        },
      ]
      const root =
        '0xec76338e61e80c68c487626bf8c793d88f189f553af34ca9d5683c2a1e81a9f5'

      should.Throw(() => {
        tree.verify(leaf, 0, inclusionProof, root)
      }, 'Invalid Merkle Sum Tree proof.')
    })

    it('should correctly reject a proof with an invalid sibling', () => {
      const tree = new MerkleSumTree()
      const leaf: MerkleTreeNode = {
        end: new BigNum(100),
        data: '0x123',
      }
      const inclusionProof: MerkleTreeNode[] = [
        {
          end: new BigNum(50),
          data:
            '0x2d8f2d36584051e513680eb7387c21fab7f2511d711694ada0674d669d89022d',
        },
      ]
      const root =
        '0xec76338e61e80c68c487626bf8c793d88f189f553af34ca9d5683c2a1e81a9f5'

      should.Throw(() => {
        tree.verify(leaf, 0, inclusionProof, root)
      }, 'Invalid Merkle Sum Tree proof.')
    })
  })

  describe('getInclusionProof', () => {
    it('should return a valid proof for a node', () => {
      const leaves: MerkleTreeNode[] = [
        {
          end: new BigNum(100),
          data: '0x123',
        },
        {
          end: new BigNum(200),
          data: '0x456',
        },
      ]
      const tree = new MerkleSumTree({
        leaves,
      })
      const expected: MerkleTreeNode[] = [
        {
          end: new BigNum(200),
          data:
            '0x2d8f2d36584051e513680eb7387c21fab7f2511d711694ada0674d669d89022d',
        },
      ]

      const inclusionProof = tree.getInclusionProof(0)

      inclusionProof.should.deep.equal(expected)
    })
  })
})
