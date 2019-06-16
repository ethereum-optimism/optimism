import { should } from '../../../setup'

/* External Imports */
import BigNumber = require('bn.js')

/* Internal Imports */
import {
  MerkleIntervalTree,
} from '../../../../src/app/common/utils/merkle-interval-tree'

import { 
  MerkleIntervalTreeLeafNode,
  MerkleIntervalTreeInternalNode,
  MerkleIntervalTreeInclusionProof,
} from '../../../../src/interfaces/common/utils/merkle-interval-tree.interface'

/**
 * Converts a string to a hex buffer.
 * @param value String value to convert.
 * @returns the string as a hex buffer.
 */
const hexify = (value: string): Buffer => {
  return Buffer.from(value, 'hex')
}

describe('MerkleIntervalTree', () => {
  describe('construction', () => {
    it('should be created correctly with no leaves', () => {
      const tree = new MerkleIntervalTree()

      const root = tree.getRoot()
      const levels = tree.getLevels()

      should.not.exist(root)
      levels.should.deep.equal([[]])
    })

    it('should be created correctly with one leaf', () => {
      const tree = new MerkleIntervalTree([
        {
          start: new BigNumber(0),
          end: new BigNumber(100),
          data: hexify('1234'),
        },
      ])

      const root = tree.getRoot()

      root.should.deep.equal({
        index: new BigNumber(0),
        hash: hexify(
          '4d7654430bd809384e15ef3e842aad2449b6310b25c652a220af74716b37e0ae'
        ),
      })
    })

    it('should be created correctly with two leaves', () => {
      const tree = new MerkleIntervalTree([
        {
          start: new BigNumber(0),
          end: new BigNumber(100),
          data: hexify('1234'),
        },
        {
          start: new BigNumber(100),
          end: new BigNumber(200),
          data: hexify('5678'),
        },
      ])

      const root = tree.getRoot()

      root.should.deep.equal({
        index: new BigNumber(0),
        hash: hexify(
          'e1b53cab461af771ad8d060145d2e27a04ee7c2e671efe4feac748de8cef1fc5'
        ),
      })
    })
  })

  describe('checkInclusionProof', () => {
    it('should correctly verify a valid proof', () => {
      const leaf: MerkleIntervalTreeLeafNode = {
        start: new BigNumber(0),
        end: new BigNumber(100),
        data: hexify('1234'),
      }
      const inclusionProof: MerkleIntervalTreeInclusionProof = [
        {
          index: new BigNumber(100),
          hash: hexify(
            '05cc573cfe77fad641c92f62241633a64f5656275753ae9b8bf67b44f29a777b'
          ),
        },
      ]
      const rootHash = hexify(
        'e1b53cab461af771ad8d060145d2e27a04ee7c2e671efe4feac748de8cef1fc5'
      )

      const bounds = new MerkleIntervalTree().checkInclusionProof(
        leaf,
        0,
        inclusionProof,
        rootHash
      )

      bounds.should.deep.equal({
        start: new BigNumber(0),
        end: new BigNumber(100),
      })
    })

    it('should correctly reject a proof with the wrong root', () => {
      const leaf: MerkleIntervalTreeLeafNode = {
        start: new BigNumber(0),
        end: new BigNumber(100),
        data: hexify('1234'),
      }
      const inclusionProof: MerkleIntervalTreeInclusionProof = [
        {
          index: new BigNumber(100),
          hash: hexify(
            '05cc573cfe77fad641c92f62241633a64f5656275753ae9b8bf67b44f29a777b'
          ),
        },
      ]
      const rootHash = hexify(
        '0000000000000000000000000000000000000000000000000000000000000000'
      )

      should.Throw(() => {
        new MerkleIntervalTree().checkInclusionProof(
          leaf,
          0,
          inclusionProof,
          rootHash
        )
      }, 'Invalid Merkle Interval Tree proof -- invalid root hash.')
    })

    it('should correctly reject a proof with an invalid sibling hash', () => {
      const leaf: MerkleIntervalTreeLeafNode = {
        start: new BigNumber(0),
        end: new BigNumber(100),
        data: hexify('1234'),
      }
      const inclusionProof: MerkleIntervalTreeInclusionProof = [
        {
          index: new BigNumber(100),
          hash: hexify(
            '0000000000000000000000000000000000000000000000000000000000000000'
          ),
        },
      ]
      const rootHash = hexify(
        'e1b53cab461af771ad8d060145d2e27a04ee7c2e671efe4feac748de8cef1fc5'
      )

      should.Throw(() => {
        new MerkleIntervalTree().checkInclusionProof(
          leaf,
          0,
          inclusionProof,
          rootHash
        )
      }, 'Invalid Merkle Interval Tree proof -- invalid root hash.')
    })

    it('should correctly reject a proof with an overlapping sibling', () => {
      const leaf: MerkleIntervalTreeLeafNode = {
        start: new BigNumber(0),
        end: new BigNumber(100),
        data: hexify('1234'),
      }
      const inclusionProof: MerkleIntervalTreeInclusionProof = [
        {
          index: new BigNumber(50),
          hash: hexify(
            '85c599e3cc2588f5c561128ab27347805ad71c33ea8db75d18823e7117bb9d4b'
          ),
        },
      ]
      const rootHash = hexify(
        '224678c7f9b59e07eb036c1914798220b3d1b8d56beb18518f6698d9a0146b84'
      )

      should.Throw(() => {
        new MerkleIntervalTree().checkInclusionProof(
          leaf,
          0,
          inclusionProof,
          rootHash
        )
      }, 'Invalid Merkle Interval Tree proof -- potential intersection detected.')
    })

    it('should correctly reject a proof with a non-monotonic right sibling', () => {
      const leaf: MerkleIntervalTreeLeafNode = {
        start: new BigNumber(0),
        end: new BigNumber(100),
        data: hexify('1234'),
      }
      const inclusionProof: MerkleIntervalTreeInclusionProof = [
        {
          index: new BigNumber(300),
          hash: hexify(
            '12c3f0bbe76afc5f6aefd5b3584fa90bc9e49f945ebd6fe5f4b86205b44e2b71'
          ),
        },
        {
          index: new BigNumber(100),
          hash: hexify(
            '89e643ad387e04a149ba8c0d7b62ac42ff32c212183035b1a7a720c0ee24699e'
          ),
        },
      ]
      const rootHash = hexify(
        'd40fc6083f76dd701db66dfbc465a945d8148067181c4c6e007e8f02c90853e3'
      )

      should.Throw(() => {
        new MerkleIntervalTree().checkInclusionProof(
          leaf,
          0,
          inclusionProof,
          rootHash
        )
      }, 'Invalid Merkle Interval Tree proof -- potential intersection detected.')
    })
  })

  describe('getInclusionProof', () => {
    it('should return a valid proof for a node', () => {
      const tree = new MerkleIntervalTree([
        {
          start: new BigNumber(0),
          end: new BigNumber(100),
          data: hexify('1234'),
        },
        {
          start: new BigNumber(100),
          end: new BigNumber(200),
          data: hexify('5678'),
        },
      ])

      const inclusionProof = tree.getInclusionProof(0)

      inclusionProof.should.deep.equal([
        {
          index: new BigNumber(100),
          hash: hexify(
            '05cc573cfe77fad641c92f62241633a64f5656275753ae9b8bf67b44f29a777b'
          ),
        },
      ])
    })

    it('should throw an error when getting a proof for a non-existent node', () => {
      const tree = new MerkleIntervalTree([
        {
          start: new BigNumber(0),
          end: new BigNumber(100),
          data: hexify('1234'),
        },
        {
          start: new BigNumber(100),
          end: new BigNumber(200),
          data: hexify('5678'),
        },
      ])

      should.Throw(() => {
        tree.getInclusionProof(2)
      }, 'Leaf position is out of bounds.')
    })
  })
})
