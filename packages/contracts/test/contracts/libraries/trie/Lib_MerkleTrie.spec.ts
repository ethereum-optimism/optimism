/* External Imports */
import * as rlp from 'rlp'
import { ethers } from 'hardhat'
import { Contract } from 'ethers'
import { fromHexString, toHexString } from '@eth-optimism/core-utils'
import { Trie } from 'merkle-patricia-tree/dist/baseTrie'

/* Internal Imports */
import { expect } from '../../../setup'
import { TrieTestGenerator } from '../../../helpers'
import * as officialTestJson from '../../../data/json/libraries/trie/trietest.json'
import * as officialTestAnyOrderJson from '../../../data/json/libraries/trie/trieanyorder.json'

const NODE_COUNTS = [1, 2, 32, 128]

describe('Lib_MerkleTrie', () => {
  let Lib_MerkleTrie: Contract
  before(async () => {
    Lib_MerkleTrie = await (
      await ethers.getContractFactory('TestLib_MerkleTrie')
    ).deploy()
  })

  // Eth-foundation tests: https://github.com/ethereum/tests/tree/develop/TrieTests
  describe('official tests', () => {
    for (const testName of Object.keys(officialTestJson.tests)) {
      it(`should perform official test: ${testName}`, async () => {
        const trie = new Trie()
        const inputs = officialTestJson.tests[testName].in
        const expected = officialTestJson.tests[testName].root

        for (const input of inputs) {
          let key: Buffer
          if (input[0].startsWith('0x')) {
            key = fromHexString(input[0])
          } else {
            key = fromHexString(
              ethers.utils.hexlify(ethers.utils.toUtf8Bytes(input[0]))
            )
          }

          let val: Buffer
          if (input[1] === null) {
            throw new Error('deletions not supported, check your tests')
          } else if (input[1].startsWith('0x')) {
            val = fromHexString(input[1])
          } else {
            val = fromHexString(
              ethers.utils.hexlify(ethers.utils.toUtf8Bytes(input[1]))
            )
          }

          const proof = await Trie.createProof(trie, key)
          const root = trie.root
          await trie.put(key, val)

          const out = await Lib_MerkleTrie.update(
            toHexString(key),
            toHexString(val),
            toHexString(rlp.encode(proof)),
            root
          )

          expect(out).to.equal(toHexString(trie.root))
        }

        expect(toHexString(trie.root)).to.equal(expected)
      })
    }
  })

  describe('official tests - trie any order', () => {
    for (const testName of Object.keys(officialTestAnyOrderJson.tests)) {
      it(`should perform official test: ${testName}`, async () => {
        const trie = new Trie()
        const inputs = officialTestAnyOrderJson.tests[testName].in
        const expected = officialTestAnyOrderJson.tests[testName].root

        for (const input of Object.keys(inputs)) {
          let key: Buffer
          if (input.startsWith('0x')) {
            key = fromHexString(input)
          } else {
            key = fromHexString(
              ethers.utils.hexlify(ethers.utils.toUtf8Bytes(input))
            )
          }

          let val: Buffer
          if (inputs[input] === null) {
            throw new Error('deletions not supported, check your tests')
          } else if (inputs[input].startsWith('0x')) {
            val = fromHexString(inputs[input])
          } else {
            val = fromHexString(
              ethers.utils.hexlify(ethers.utils.toUtf8Bytes(inputs[input]))
            )
          }

          const proof = await Trie.createProof(trie, key)
          const root = trie.root
          await trie.put(key, val)

          const out = await Lib_MerkleTrie.update(
            toHexString(key),
            toHexString(val),
            toHexString(rlp.encode(proof)),
            root
          )

          expect(out).to.equal(toHexString(trie.root))
        }

        expect(toHexString(trie.root)).to.equal(expected)
      })
    }
  })

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

  describe('update', () => {
    for (const nodeCount of NODE_COUNTS) {
      describe(`inside a trie with ${nodeCount} nodes and keys/vals of size ${nodeCount} bytes`, () => {
        let generator: TrieTestGenerator
        before(async () => {
          generator = await TrieTestGenerator.fromRandom({
            seed: `seed.update.${nodeCount}`,
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
          it(`should correctly update node #${i}`, async () => {
            const test = await generator.makeNodeUpdateTest(
              i,
              '0x1234123412341234'
            )

            expect(
              await Lib_MerkleTrie.update(
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

    it('should return the single-node root hash if the trie was previously empty', async () => {
      const key = '0x1234'
      const val = '0x5678'

      const trie = new Trie()
      await trie.put(fromHexString(key), fromHexString(val))

      expect(
        await Lib_MerkleTrie.update(
          key,
          val,
          '0x', // Doesn't require a proof
          ethers.utils.keccak256('0x80') // Empty Merkle trie root hash
        )
      ).to.equal(toHexString(trie.root))
    })
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
