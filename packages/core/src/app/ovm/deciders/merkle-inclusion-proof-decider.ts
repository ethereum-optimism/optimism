import { Decider, Decision } from '../../../types/ovm'
import { MerkleTreeInclusionProof } from '../../../types'
import { SparseMerkleTreeImpl } from '../../block-production'
import { newInMemoryDB } from '../../db'

export interface MerkleInclusionProofDeciderInput {
  merkleProof: MerkleTreeInclusionProof
}

/**
 * Decider that determines whether or not a Merkle Inclusion Proof is valid.
 */
export class MerkleInclusionProofDecider implements Decider {
  public async decide(
    input: MerkleInclusionProofDeciderInput,
    _noCache?: boolean
  ): Promise<Decision> {
    const tree = new SparseMerkleTreeImpl(
      newInMemoryDB(256),
      input.merkleProof.rootHash,
      input.merkleProof.siblings.length + 1
    )
    const outcome = await tree.verifyAndStore(input.merkleProof)

    return {
      outcome,
      justification: [
        {
          implication: {
            decider: this,
            input,
          },
          implicationWitness: undefined,
        },
      ],
    }
  }
}
