/* External Imports */
import { BigNumber } from '@eth-optimism/core-utils'
import { MerkleTree } from '@eth-optimism/core-db'

/* Internal Imports */
import { QuantifierResult, Quantifier } from '../../types'

interface MerkleInclusionQuantifierParameters {
  root: Buffer
  key: BigNumber
}

/*
 * The MerkleInclusionQuantifier that quantifies data at a specific key of a Merkle Tree,
 * if data exists at the key.
 */
export class MerkleInclusionQuantifier implements Quantifier {
  public constructor(private readonly tree: MerkleTree) {}

  /**
   * Returns the data at the provided key of the MerkleTree, if any data exists
   * at that location.
   *
   * @param input The MerkleInclusionQuantifierParameters object to quantify
   * @returns The QuantifierResult with the data in question
   */
  public async getAllQuantified(
    input: MerkleInclusionQuantifierParameters
  ): Promise<QuantifierResult> {
    const leafData: Buffer = await this.tree.getLeaf(input.key, input.root)
    return {
      results: leafData ? [leafData] : [undefined],
      allResultsQuantified: !!leafData,
    }
  }
}
