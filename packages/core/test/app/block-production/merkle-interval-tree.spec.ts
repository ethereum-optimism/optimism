import { should } from '../../setup'

/* External Imports */
import debug from 'debug'
const log = debug('test:info:merkle-index-tree')
import BigNum = require('bn.js')

/* Internal Imports */
import {
    AbiStateUpdate, AbiStateObject, AbiRange,
    MerkleIntervalTree, MerkleIntervalTreeNode, MerkleStateIntervalTree, PlasmaBlock
  }  from '../../../src/app/'

describe.only('merkle-index-tree', () => {
  describe('MerkleIntervalTreeNode', () => {
    it('should concatenate index and hash after construction', async() => {
      const node = new MerkleIntervalTreeNode(Buffer.from([255]), Buffer.from([0])) 
      log('New merkle index tree node:', node)
      const expected = Buffer.concat([Buffer.from([255]), Buffer.from([0])])
      node.data.should.deep.equal(expected)
    })
  })
  describe('MerkleIntervalTree', () => {
    describe('parent', () => {
      it('should return the correct parent', async() => {
        const left = new MerkleIntervalTreeNode(Buffer.from([13]), Buffer.from([10])) 
        const right = new MerkleIntervalTreeNode(Buffer.from([31]), Buffer.from([15])) 
        const parent = MerkleIntervalTree.parent(left, right)
        // We calculated the hash by hand.
        parent.data.toString('hex').should.equal('69b053cd194c51ff15ac9db85fc581c4457a7160c78d878e7c5b84f4c1fbb9140a')
      })
      it('should throw if left & right nodes are out of order', async() => {
        const left = new MerkleIntervalTreeNode(Buffer.from([13]), Buffer.from([15])) 
        const right = new MerkleIntervalTreeNode(Buffer.from([31]), Buffer.from([10])) 
        const parentCall = () => MerkleIntervalTree.parent(left, right)
        parentCall.should.throw()
      })
    })
    it('should generate a generic tree', async() => {
      const leaves = []
      for (let i = 0; i < 4; i++) {
        leaves.push(new MerkleIntervalTreeNode(Buffer.from([Math.floor(Math.random()*100)]), Buffer.from([i])))
      }
      const IntervalTree = new MerkleIntervalTree(leaves)
      log(IntervalTree.levels)
      log(IntervalTree.root)
    })
    it('should generate and verify inclusion proofs for generic tree', async() => {
      const leaves = []
      for (let i = 0; i < 4; i++) {
        leaves.push(new MerkleIntervalTreeNode(Buffer.from([Math.floor(Math.random()*100)]), Buffer.from([i])))
      }
      const IntervalTree = new MerkleIntervalTree(leaves)
      const leafPosition = 3
      const inclusionProof = IntervalTree.getInclusionProof(leafPosition)
      MerkleIntervalTree.verify(
        leaves[leafPosition],
        leafPosition,
        inclusionProof,
        IntervalTree.root().hash
      )
    })
  })
  describe('MerkleStateIntervalTree', () => {
    it('should generate a tree without throwing', async() => {
      const stateUpdates = []
      for (let i = 0; i < 4; i++) {
        const stateObject = new AbiStateObject('0xbdAd2846585129Fc98538ce21cfcED21dDDE0a63', '0x123456')
        const range = new AbiRange( new BigNum(i*100), new BigNum((i+0.5)* 100) )
        const stateUpdate = new AbiStateUpdate(stateObject, range, new BigNum(1), '0xbdAd2846585129Fc98538ce21cfcED21dDDE0a63')
        stateUpdates.push(stateUpdate)
      }
      const merkleStateIntervalTree = new MerkleStateIntervalTree(stateUpdates)
      log('root', merkleStateIntervalTree.root())
    })
  })
  describe('PlasmaBlock', () => {
    it('should generate a tree without throwing', async() => {
      const stateUpdates = []
      for (let i = 0; i < 4; i++) {
        const stateObject = new AbiStateObject('0xbdAd2846585129Fc98538ce21cfcED21dDDE0a63', '0x123456')
        const range = new AbiRange( new BigNum(i*100), new BigNum((i+0.5)* 100) )
        const stateUpdate = new AbiStateUpdate(stateObject, range, new BigNum(1), '0xbdAd2846585129Fc98538ce21cfcED21dDDE0a63')
        stateUpdates.push(stateUpdate)
      }
      const blockContents = [
        {
          assetId: Buffer.from('1dAd2846585129Fc98538ce21cfcED21dDDE0a63', 'hex'),
          stateUpdates
        },
        {
          assetId: Buffer.from('bdAd2846585129Fc98538ce21cfcED21dDDE0a63', 'hex'),
          stateUpdates
        }
      ]
      const plasmaBlock = new PlasmaBlock(blockContents)
      log(plasmaBlock)
    })
    it('should generate and verify a StateUpdateInclusionProof', async() => {
      const stateUpdates = []
      for (let i = 0; i < 4; i++) {
        const stateObject = new AbiStateObject('0xbdAd2846585129Fc98538ce21cfcED21dDDE0a63', '0x123456')
        const range = new AbiRange( new BigNum(i*100), new BigNum((i+0.5)* 100) )
        const stateUpdate = new AbiStateUpdate(stateObject, range, new BigNum(1), '0xbdAd2846585129Fc98538ce21cfcED21dDDE0a63')
        stateUpdates.push(stateUpdate)
      }
      const blockContents = [
        {
          assetId: Buffer.from('1dAd2846585129Fc98538ce21cfcED21dDDE0a63', 'hex'),
          stateUpdates
        },
        {
          assetId: Buffer.from('bdAd2846585129Fc98538ce21cfcED21dDDE0a63', 'hex'),
          stateUpdates
        }
      ]
      const plasmaBlock = new PlasmaBlock(blockContents)
      const stateProof = (plasmaBlock.getStateUpdateInclusionProof(1, 1))
      PlasmaBlock.verifyStateUpdateInclusionProof(
        blockContents[1].stateUpdates[1],
        stateProof.stateTreeInclusionProof,
        1,
        stateProof.addressTreeInclusionProof,
        1,
        plasmaBlock.root().hash
      )
    })
  })
})