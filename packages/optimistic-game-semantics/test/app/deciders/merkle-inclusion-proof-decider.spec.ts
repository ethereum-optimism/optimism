import '../../setup'

/* External Imports */
import { BigNumber, objectsEqual, ONE } from '@eth-optimism/core-utils'
import {
  MerkleTreeInclusionProof,
  newInMemoryDB,
  SparseMerkleTreeImpl,
} from '@eth-optimism/core-db'
import * as assert from 'assert'

/* Internal Imports */
import {
  Decider,
  Decision,
  ImplicationProofItem,
  MerkleInclusionProofDecider,
} from '../../../src'

describe('MerkleTreeInclusionProofDecider', () => {
  describe('decide', () => {
    const decider: Decider = new MerkleInclusionProofDecider()
    const leafValue: Buffer = Buffer.from('Leaf value')

    it('should return true when inclusion proof is valid', async () => {
      const merkleTree: SparseMerkleTreeImpl = await SparseMerkleTreeImpl.create(
        newInMemoryDB(256)
      )
      assert(
        await merkleTree.update(ONE, leafValue),
        'Merkle tree update should have succeeded'
      )

      const merkleProof: MerkleTreeInclusionProof = await merkleTree.getMerkleProof(
        ONE,
        leafValue
      )

      const decision: Decision = await decider.decide({ merkleProof })

      decision.outcome.should.equal(true)
      decision.justification.length.should.equal(1)

      const justification: ImplicationProofItem = decision.justification[0]
      justification.implication.decider.should.equal(decider)
      assert(
        objectsEqual(
          justification.implication.input['merkleProof'],
          merkleProof
        )
      )
    })

    it('should return true when inclusion proof is valid for 32-height tree', async () => {
      const merkleTree: SparseMerkleTreeImpl = await SparseMerkleTreeImpl.create(
        newInMemoryDB(256),
        undefined,
        32
      )

      assert(
        await merkleTree.update(ONE, leafValue),
        'Merkle tree update should have succeeded'
      )

      const merkleProof: MerkleTreeInclusionProof = await merkleTree.getMerkleProof(
        ONE,
        leafValue
      )

      const decision: Decision = await decider.decide({ merkleProof })

      decision.outcome.should.equal(true)
      decision.justification.length.should.equal(1)

      const justification: ImplicationProofItem = decision.justification[0]
      justification.implication.decider.should.equal(decider)
      assert(
        objectsEqual(
          justification.implication.input['merkleProof'],
          merkleProof
        )
      )
    })

    it('should return true when inclusion proof is valid for different key', async () => {
      const key: BigNumber = new BigNumber(10)
      const merkleTree: SparseMerkleTreeImpl = await SparseMerkleTreeImpl.create(
        newInMemoryDB(256)
      )
      assert(
        await merkleTree.update(key, leafValue),
        'Merkle tree update should have succeeded'
      )

      const merkleProof: MerkleTreeInclusionProof = await merkleTree.getMerkleProof(
        key,
        leafValue
      )

      const decision: Decision = await decider.decide({ merkleProof })

      decision.outcome.should.equal(true)
      decision.justification.length.should.equal(1)

      const justification: ImplicationProofItem = decision.justification[0]
      justification.implication.decider.should.equal(decider)
      assert(
        objectsEqual(
          justification.implication.input['merkleProof'],
          merkleProof
        )
      )
    })

    it('should return false when inclusion proof is invalid', async () => {
      const key: BigNumber = new BigNumber(10)
      const merkleTree: SparseMerkleTreeImpl = await SparseMerkleTreeImpl.create(
        newInMemoryDB(256)
      )
      assert(
        await merkleTree.update(key, leafValue),
        'Merkle tree update should have succeeded'
      )

      const merkleProof: MerkleTreeInclusionProof = await merkleTree.getMerkleProof(
        key,
        leafValue
      )
      merkleProof.siblings[0] = Buffer.from('not the right hash')

      const decision: Decision = await decider.decide({ merkleProof })

      decision.outcome.should.equal(false)
      decision.justification.length.should.equal(1)

      const justification: ImplicationProofItem = decision.justification[0]
      justification.implication.decider.should.equal(decider)
      assert(
        objectsEqual(
          justification.implication.input['merkleProof'],
          merkleProof
        )
      )
    })
  })
})
