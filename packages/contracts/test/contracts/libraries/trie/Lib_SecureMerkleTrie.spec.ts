import { Contract } from 'ethers'

import { expect } from '../../../setup'
import { deploy, TrieTestGenerator } from '../../../helpers'

const NODE_COUNTS = [1, 2, 128]

describe('Lib_SecureMerkleTrie', () => {
  let Lib_SecureMerkleTrie: Contract
  before(async () => {
    Lib_SecureMerkleTrie = await deploy('TestLib_SecureMerkleTrie')
  })

  describe('verifyInclusionProof', () => {
    for (const nodeCount of NODE_COUNTS) {
      describe(`inside a trie with ${nodeCount} nodes`, () => {
        let generator: TrieTestGenerator
        before(async () => {
          generator = await TrieTestGenerator.fromRandom({
            seed: `seed.incluson.${nodeCount}`,
            nodeCount,
            secure: true,
          })
        })

        for (
          let i = 0;
          i < nodeCount;
          i += nodeCount / (nodeCount > 8 ? 8 : 1)
        ) {
          it(`should correctly prove inclusion for node #${i}`, async () => {
            const test = await generator.makeInclusionProofTest(i)

            expect(
              await Lib_SecureMerkleTrie.verifyInclusionProof(
                test.key,
                test.val,
                test.proof,
                test.root
              )
            ).to.equal(true)
          })
        }
      })
    }
  })

  describe('get', () => {
    for (const nodeCount of NODE_COUNTS) {
      describe(`inside a trie with ${nodeCount} nodes`, () => {
        let generator: TrieTestGenerator
        before(async () => {
          generator = await TrieTestGenerator.fromRandom({
            seed: `seed.get.${nodeCount}`,
            nodeCount,
            secure: true,
          })
        })

        for (
          let i = 0;
          i < nodeCount;
          i += nodeCount / (nodeCount > 8 ? 8 : 1)
        ) {
          it(`should correctly get the value of node #${i}`, async () => {
            const test = await generator.makeInclusionProofTest(i)

            expect(
              await Lib_SecureMerkleTrie.get(test.key, test.proof, test.root)
            ).to.deep.equal([true, test.val])
          })
        }
      })
    }
  })
})
