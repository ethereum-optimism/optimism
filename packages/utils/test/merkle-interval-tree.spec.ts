import { should } from './setup'

/* External Imports */
import BigNumber = require('bn.js')

/* Internal Imports */
import {
  MerkleIntervalTree,
  MerkleIntervalTreeLeafNode,
  MerkleIntervalTreeInternalNode,
} from '../src/merkle-interval-tree'

describe('MerkleIntervalTree', () => {
  describe('construction', () => {
    it('should be created correctly with no leaves', () => {
      const tree = new MerkleIntervalTree()

      should.not.exist(tree.getRoot())
      tree.getLevels().should.deep.equal([[]])
    })

    it('should be created correctly with one leaf', () => {
      const leaves: MerkleIntervalTreeLeafNode[] = [
        {
          start: new BigNumber(0),
          end: new BigNumber(100),
          data: Buffer.from('0x123'),
        },
      ]
      const tree = new MerkleIntervalTree(leaves)

      tree
        .getRoot()
        .should.equal(
          '0x9b64c516a177f40236cfa25c63deb358d93081decbcac3a99dbdcfe856f1d20b'
        )
    })

    it('should be created correctly with two leaves', () => {
      const leaves: MerkleIntervalTreeLeafNode[] = [
        {
          start: new BigNumber(0),
          end: new BigNumber(100),
          data: Buffer.from('0x123'),
        },
        {
          start: new BigNumber(100),
          end: new BigNumber(200),
          data: Buffer.from('0x456'),
        },
      ]
      const tree = new MerkleIntervalTree(leaves)

      tree
        .getRoot()
        .should.equal(
          '0xec76338e61e80c68c487626bf8c793d88f189f553af34ca9d5683c2a1e81a9f5'
        )
    })
  })

  describe('checkInclusionProof', () => {
    it('should correctly verify a valid proof', () => {
      const tree = new MerkleIntervalTree()
      const leaf: MerkleIntervalTreeLeafNode = {
        start: new BigNumber(0),
        end: new BigNumber(100),
        data: Buffer.from('0x123'),
      }
      const inclusionProof: MerkleIntervalTreeInternalNode[] = [
        {
          index: new BigNumber(200),
          hash: Buffer.from(
            '0x2d8f2d36584051e513680eb7387c21fab7f2511d711694ada0674d669d89022d'
          ),
        },
      ]
      const rootHash =
        Buffer.from('0xec76338e61e80c68c487626bf8c793d88f189f553af34ca9d5683c2a1e81a9f5')

      should.not.Throw(() => {
        tree.checkInclusionProof(leaf, 0, inclusionProof, rootHash)
      })
    })

    it('should correctly reject a proof with the wrong root', () => {
      const tree = new MerkleIntervalTree()
      const leaf: MerkleIntervalTreeLeafNode = {
        start: new BigNumber(0),
        end: new BigNumber(100),
        data: Buffer.from('0x123'),
      }
      const inclusionProof: MerkleIntervalTreeInternalNode[] = [
        {
          index: new BigNumber(200),
          hash: Buffer.from(
            '0x2d8f2d36584051e513680eb7387c21fab7f2511d711694ada0674d669d89022d'
          ),
        },
      ]
      const rootHash = Buffer.from('0x000000')

      should.Throw(() => {
        tree.checkInclusionProof(leaf, 0, inclusionProof, rootHash)
      }, 'Invalid Merkle Sum Tree proof.')
    })

    it('should correctly reject a proof with the wrong siblings', () => {
      const tree = new MerkleIntervalTree()
      const leaf: MerkleIntervalTreeLeafNode = {
        start: new BigNumber(0),
        end: new BigNumber(100),
        data: Buffer.from('0x123'),
      }
      const inclusionProof: MerkleIntervalTreeInternalNode[] = [
        {
          index: new BigNumber(200),
          hash: Buffer.from('0x00000000'),
        },
      ]
      const rootHash =
        Buffer.from('0xec76338e61e80c68c487626bf8c793d88f189f553af34ca9d5683c2a1e81a9f5')

      should.Throw(() => {
        tree.checkInclusionProof(leaf, 0, inclusionProof, rootHash)
      }, 'Invalid Merkle Sum Tree proof.')
    })

    it('should correctly reject a proof with an invalid sibling', () => {
      const tree = new MerkleIntervalTree()
      const leaf: MerkleIntervalTreeLeafNode = {
        start: new BigNumber(0),
        end: new BigNumber(100),
        data: Buffer.from('0x123'),
      }
      const inclusionProof: MerkleIntervalTreeInternalNode[] = [
        {
          index: new BigNumber(50),
          hash: Buffer.from(
            '0x2d8f2d36584051e513680eb7387c21fab7f2511d711694ada0674d669d89022d'
          ),
        },
      ]
      const rootHash =
        Buffer.from('0xec76338e61e80c68c487626bf8c793d88f189f553af34ca9d5683c2a1e81a9f5')

      should.Throw(() => {
        tree.checkInclusionProof(leaf, 0, inclusionProof, rootHash)
      }, 'Invalid Merkle Sum Tree proof.')
    })
  })

  describe('getInclusionProof', () => {
    it('should return a valid proof for a node', () => {
      const leaves: MerkleIntervalTreeLeafNode[] = [
        {
          start: new BigNumber(0),
          end: new BigNumber(100),
          data: Buffer.from('0x123'),
        },
        {
          start: new BigNumber(0),
          end: new BigNumber(200),
          data: Buffer.from('0x456'),
        },
      ]
      const tree = new MerkleIntervalTree(leaves)
      const expected: MerkleIntervalTreeInternalNode[] = [
        {
          index: new BigNumber(200),
          hash: Buffer.from(
            '0x2d8f2d36584051e513680eb7387c21fab7f2511d711694ada0674d669d89022d'
          ),
        },
      ]

      const inclusionProof = tree.getInclusionProof(0)

      inclusionProof.should.deep.equal(expected)
    })
  })
})
