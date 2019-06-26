import { should } from '../../setup'

/* External Imports */
import debug from 'debug'
const log = debug('test:info:merkle-index-tree')
import BigNum = require('bn.js')

/* Internal Imports */
import {
  AbiStateUpdate,
  AbiStateObject,
  AbiRange,
  GenericMerkleIntervalTree,
  GenericMerkleIntervalTreeNode,
  MerkleStateIntervalTree,
  PlasmaBlock,
} from '../../../src/app/'

function generateNSequentialStateUpdates(numerOfUpdates: number): AbiStateUpdate[] {
  const stateUpdates: AbiStateUpdate[] = []
  for (let i = 0; i < numerOfUpdates; i++) {
    const stateObject = new AbiStateObject(
      '0xbdAd2846585129Fc98538ce21cfcED21dDDE0a63',
      '0x123456'
    )
    const range = new AbiRange(
      new BigNum(i * 100),
      new BigNum((i + 0.5) * 100)
    )
    const stateUpdate = new AbiStateUpdate(
      stateObject,
      range,
      new BigNum(1),
      '0xbdAd2846585129Fc98538ce21cfcED21dDDE0a63'
    )
    stateUpdates.push(stateUpdate)
  }
  return stateUpdates
}

describe.only('Interval Trees and Plasma Blocks', () => {
  describe('GenericMerkleIntervalTreeNode', () => {
    it('should concatenate index and hash after construction', async () => {
      const node = new GenericMerkleIntervalTreeNode(
        Buffer.from([255]),
        Buffer.from([0])
      )
      log('New merkle index tree node:', node)
      const expected = Buffer.concat([Buffer.from([255]), Buffer.from([0])])
      node.data.should.deep.equal(expected)
    })
  })
  describe('MerkleIntervalTree', () => {
    describe('parent', () => {
      it('should return the correct parent', async () => {
        const left = new GenericMerkleIntervalTreeNode(
          Buffer.from([13]),
          Buffer.from([10])
        )
        const right = new GenericMerkleIntervalTreeNode(
          Buffer.from([31]),
          Buffer.from([15])
        )
        const parent = GenericMerkleIntervalTree.parent(left, right)
        // We calculated the hash by hand.
        parent.data
          .toString('hex')
          .should.equal(
            '69b053cd194c51ff15ac9db85fc581c4457a7160c78d878e7c5b84f4c1fbb9140a'
          )
      })
      it('should throw if left & right nodes are out of order', async () => {
        const left = new GenericMerkleIntervalTreeNode(
          Buffer.from([13]),
          Buffer.from([15])
        )
        const right = new GenericMerkleIntervalTreeNode(
          Buffer.from([31]),
          Buffer.from([10])
        )
        const parentCall = () => GenericMerkleIntervalTree.parent(left, right)
        parentCall.should.throw()
      })
    })
    it('should generate a generic tree', async () => {
      const leaves = []
      for (let i = 0; i < 4; i++) {
        leaves.push(
          new GenericMerkleIntervalTreeNode(
            Buffer.from([Math.floor(Math.random() * 100)]),
            Buffer.from([i])
          )
        )
      }
      const IntervalTree = new GenericMerkleIntervalTree(leaves)
      log(IntervalTree.levels)
      log(IntervalTree.root)
    })
    it('should generate and verify inclusion proofs for generic tree', async () => {
      const leaves = []
      for (let i = 0; i < 4; i++) {
        leaves.push(
          new GenericMerkleIntervalTreeNode(
            Buffer.from([Math.floor(Math.random() * 100)]),
            Buffer.from([i])
          )
        )
      }
      const IntervalTree = new GenericMerkleIntervalTree(leaves)
      const leafPosition = 3
      const inclusionProof = IntervalTree.getInclusionProof(leafPosition)
      GenericMerkleIntervalTree.verify(
        leaves[leafPosition],
        leafPosition,
        inclusionProof,
        IntervalTree.root().hash
      )
    })
  })
  describe('MerkleStateIntervalTree', () => {
    it('should generate a tree without throwing', async () => {
      const stateUpdates = generateNSequentialStateUpdates(4)
      const merkleStateIntervalTree = new MerkleStateIntervalTree(stateUpdates)
      log('root', merkleStateIntervalTree.root())
    })
    it('verification should throw if state update range intersects branch bounds', async () => {
      // generate some valid tree contents
      const stateUpdates = generateNSequentialStateUpdates(5)
      // make an invalid range intersecting the second SU
      const faultyUpdateIndex = 0
      const updateToReplace = stateUpdates[faultyUpdateIndex]
      const conflictingRange = new AbiRange(
        updateToReplace.range.start, 
        stateUpdates[faultyUpdateIndex + 1].range.start.add(new BigNum(1))
      )
      // replace the valid SU range with the conflicting one
      const faultyUpdate = new AbiStateUpdate(
        updateToReplace.stateObject,
        conflictingRange,
        updateToReplace.plasmaBlockNumber,
        updateToReplace.depositAddress
      )
      stateUpdates[faultyUpdateIndex] = faultyUpdate
      // Generate inclusion proof
      const merkleStateIntervalTree = new MerkleStateIntervalTree(stateUpdates)
      const faultyInclusionProof = merkleStateIntervalTree.getInclusionProof(faultyUpdateIndex)
        
      should.throw(
        () =>  {
          MerkleStateIntervalTree.verifyExectedRoot(
            faultyUpdate,
            faultyUpdateIndex,
            faultyInclusionProof,
          )
        },
        Error,
        'State Update range.end exceeds the max for its inclusion proof.'
      )
    })
  })
  describe('PlasmaBlock', () => {
    it('should generate a plasma block without throwing', async () => {
      const stateUpdates = generateNSequentialStateUpdates(4)
      const blockContents = [
        {
          assetId: Buffer.from('1dAd2846585129Fc98538ce21cfcED21dDDE0a63', 'hex'),
          stateUpdates,
        },
        {
          assetId: Buffer.from('bdAd2846585129Fc98538ce21cfcED21dDDE0a63', 'hex'),
          stateUpdates,
        },
      ]
      const plasmaBlock = new PlasmaBlock(blockContents)
      log(plasmaBlock)
    })
    it('should generate and verify a StateUpdateInclusionProof', async () => {
      const stateUpdates = generateNSequentialStateUpdates(4)
      const blockContents = [
        {
          assetId: Buffer.from('1dAd2846585129Fc98538ce21cfcED21dDDE0a63', 'hex'),
          stateUpdates,
        },
        {
          assetId: Buffer.from('bdAd2846585129Fc98538ce21cfcED21dDDE0a63', 'hex'),
          stateUpdates,
        },
      ]
      const plasmaBlock = new PlasmaBlock(blockContents)
      const stateProof = plasmaBlock.getStateUpdateInclusionProof(1, 1)
      PlasmaBlock.verifyStateUpdateInclusionProof(
        blockContents[1].stateUpdates[1],
        stateProof.stateTreeInclusionProof,
        1,
        stateProof.assetTreeInclusionProof,
        1,
        plasmaBlock.root().hash
      )
    })
  })
})
