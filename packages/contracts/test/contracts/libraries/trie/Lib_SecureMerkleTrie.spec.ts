/* External Imports */
import { ethers } from 'hardhat'
import { Contract } from 'ethers'

/* Internal Imports */
import { expect } from '../../../setup'
import { TrieTestGenerator } from '../../../helpers'

const NODE_COUNTS = [1, 2, 128]

describe('Lib_SecureMerkleTrie', () => {
  let Lib_SecureMerkleTrie: Contract
  before(async () => {
    Lib_SecureMerkleTrie = await (
      await ethers.getContractFactory('TestLib_SecureMerkleTrie')
    ).deploy()
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

  describe('update', () => {
    for (const nodeCount of NODE_COUNTS) {
      describe(`inside a trie with ${nodeCount} nodes`, () => {
        let generator: TrieTestGenerator
        before(async () => {
          generator = await TrieTestGenerator.fromRandom({
            seed: `seed.update.${nodeCount}`,
            nodeCount,
            secure: true,
          })
        })

        for (
          let i = 0;
          i < nodeCount;
          i += nodeCount / (nodeCount > 8 ? 8 : 1)
        ) {
          it(`should correctly update node #${i}`, async () => {
            const test = await generator.makeNodeUpdateTest(
              i,
              '0x1234123412341234'
            )

            expect(
              await Lib_SecureMerkleTrie.update(
                test.key,
                test.val,
                test.proof,
                test.root
              )
            ).to.equal(test.newRoot)
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

  describe('getSingleNodeRootHash', () => {
    let generator: TrieTestGenerator
    before(async () => {
      generator = await TrieTestGenerator.fromRandom({
        seed: `seed.get.${1}`,
        nodeCount: 1,
        secure: true,
      })
    })

    it(`should get the root hash of a trie with a single node`, async () => {
      const test = await generator.makeInclusionProofTest(0)
      expect(
        await Lib_SecureMerkleTrie.getSingleNodeRootHash(test.key, test.val)
      ).to.equal(test.root)
    })
  })
})
