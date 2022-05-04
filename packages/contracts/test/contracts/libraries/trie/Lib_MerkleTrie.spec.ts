import * as rlp from 'rlp'
import { ethers } from 'hardhat'
import { Contract } from 'ethers'
import { toHexString } from '@eth-optimism/core-utils'

import { expect } from '../../../setup'
import { deploy, TrieTestGenerator } from '../../../helpers'

const NODE_COUNTS = [1, 2, 32, 128]

describe('Lib_MerkleTrie', () => {
  let Lib_MerkleTrie: Contract
  before(async () => {
    Lib_MerkleTrie = await deploy('TestLib_MerkleTrie')
  })

  // Eth-foundation tests: https://github.com/ethereum/tests/tree/develop/TrieTests
  describe('verifyInclusionProof', () => {
    for (const nodeCount of NODE_COUNTS) {
      describe(`inside a trie with ${nodeCount} nodes and keys/vals of size ${nodeCount} bytes`, () => {
        let generator: TrieTestGenerator
        before(async () => {
          generator = await TrieTestGenerator.fromRandom({
            seed: `seed.incluson.${nodeCount}`,
            nodeCount,
            secure: false,
            keySize: nodeCount,
            valSize: nodeCount,
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
              await Lib_MerkleTrie.verifyInclusionProof(
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
      describe(`inside a trie with ${nodeCount} nodes and keys/vals of size ${nodeCount} bytes`, () => {
        let generator: TrieTestGenerator
        before(async () => {
          generator = await TrieTestGenerator.fromRandom({
            seed: `seed.get.${nodeCount}`,
            nodeCount,
            secure: false,
            keySize: nodeCount,
            valSize: nodeCount,
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
              await Lib_MerkleTrie.get(test.key, test.proof, test.root)
            ).to.deep.equal([true, test.val])
          })
          if (i > 3) {
            it(`should revert when the proof node does not pass the root check`, async () => {
              const test = await generator.makeInclusionProofTest(i - 1)
              const test2 = await generator.makeInclusionProofTest(i - 2)
              await expect(
                Lib_MerkleTrie.get(test2.key, test.proof, test.root)
              ).to.be.revertedWith('Invalid large internal hash')
            })
            it(`should revert when the first proof element is not the root node`, async () => {
              const test = await generator.makeInclusionProofTest(0)
              const decodedProof = rlp.decode(test.proof)
              decodedProof[0].write('abcd', 8) // change the 1st element (root) of the proof
              const badProof = rlp.encode(decodedProof as rlp.Input)
              await expect(
                Lib_MerkleTrie.get(test.key, badProof, test.root)
              ).to.be.revertedWith('Invalid root hash')
            })
            it(`should be false when calling get on an incorrect key`, async () => {
              const test = await generator.makeInclusionProofTest(i - 1)
              let newKey = test.key.slice(0, test.key.length - 8)
              newKey = newKey.concat('88888888')
              expect(
                await Lib_MerkleTrie.get(newKey, test.proof, test.root)
              ).to.deep.equal([false, '0x'])
            })
          }
        }
      })
    }
  })

  describe(`inside a trie with one node`, () => {
    let generator: TrieTestGenerator
    const nodeCount = 1
    before(async () => {
      generator = await TrieTestGenerator.fromRandom({
        seed: `seed.get.${nodeCount}`,
        nodeCount,
        secure: false,
      })
    })

    it(`should revert on an incorrect proof node prefix`, async () => {
      const test = await generator.makeInclusionProofTest(0)
      const decodedProof = rlp.decode(test.proof)
      decodedProof[0].write('a', 3) // change the prefix
      test.root = ethers.utils.keccak256(toHexString(decodedProof[0]))
      const badProof = rlp.encode(decodedProof as rlp.Input)
      await expect(
        Lib_MerkleTrie.get(test.key, badProof, test.root)
      ).to.be.revertedWith('Received a node with an unknown prefix')
    })
  })
})
