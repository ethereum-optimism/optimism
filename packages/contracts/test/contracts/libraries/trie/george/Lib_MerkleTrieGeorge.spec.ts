import { expect } from '../../../../setup'

/* External Imports */
import * as rlp from 'rlp'
import { ethers } from 'hardhat'
import { Contract } from 'ethers'
import { fromHexString, toHexString } from '@eth-optimism/core-utils'
import { Trie } from 'merkle-patricia-tree/dist/baseTrie'

/* Internal Imports */
import { TrieNode, TrieTestGenerator } from '../../../../helpers'
import * as officialTestJson from '../../../../data/json/libraries/trie/trietest.json'
import * as officialTestAnyOrderJson from '../../../../data/json/libraries/trie/trieanyorder.json'

const NODE_COUNTS = [1, 2, 32]
// The original also tests 128 but this makes the test timeout because we need to populate the trie.

const EMPTY_MERKLE_ROOT = ethers.utils.keccak256('0x80')

describe('Lib_MerkleTrieGeorge', () => {
  let Lib_MerkleTrieGeorge: Contract

  const populateTrie = async (nodes: TrieNode[]): Promise<string> => {
    let root = EMPTY_MERKLE_ROOT
    for (const node of nodes) {
      const tx = await Lib_MerkleTrieGeorge.update(node.key, node.val, root)
      const receipt = await tx.wait()
      // TODO very ugly & brittle - how to do this properly?
      const events = receipt.events?.filter((x) => {
        return x.event === 'GeorgeHash'
      })
      root = events[0].args[0]
    }
    return root
  }

  // IMPORTANT: because of how the storage is implemented (hash -> encoding/value), a single instance
  // is effectively able to hold arbitrarily many Merkle trees
  before(async () => {
    Lib_MerkleTrieGeorge = await (
      await ethers.getContractFactory('TestLib_MerkleTrieGeorge')
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

          const root = trie.root
          await trie.put(key, val)

          expect(
            await Lib_MerkleTrieGeorge.update(
              toHexString(key),
              toHexString(val),
              root
            )
          )
            .to.emit(Lib_MerkleTrieGeorge, 'GeorgeHash(bytes32)')
            .withArgs(toHexString(trie.root))
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

          const root = trie.root
          await trie.put(key, val)

          expect(
            await Lib_MerkleTrieGeorge.update(
              toHexString(key),
              toHexString(val),
              root
            )
          )
            .to.emit(Lib_MerkleTrieGeorge, 'GeorgeHash(bytes32)')
            .withArgs(toHexString(trie.root))
        }

        expect(toHexString(trie.root)).to.equal(expected)
      })
    }
  })

  // Don't test `verifyInclusionProof` and `inside a trie with one node` - since George's
  // implementation does not support passing proofs directly.

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
          await populateTrie(generator._nodes)
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
              await Lib_MerkleTrieGeorge.update(test.key, test.val, test.root)
            )
              .to.emit(Lib_MerkleTrieGeorge, 'GeorgeHash(bytes32)')
              .withArgs(test.newRoot)
          })
        }
      })
    }

    it('should return the single-node root hash if the trie was previously empty', async () => {
      const key = '0x1234'
      const val = '0x5678'
      const trie = new Trie()
      await trie.put(fromHexString(key), fromHexString(val))
      expect(await Lib_MerkleTrieGeorge.update(key, val, EMPTY_MERKLE_ROOT))
        .to.emit(Lib_MerkleTrieGeorge, 'GeorgeHash(bytes32)')
        .withArgs(toHexString(trie.root))
    })
  })

  describe('get', () => {
    for (const nodeCount of NODE_COUNTS) {
      describe(`inside a trie with ${nodeCount} nodes and keys/vals of size ${nodeCount} bytes`, async () => {
        let generator: TrieTestGenerator
        before(async () => {
          generator = await TrieTestGenerator.fromRandom({
            seed: `seed.get.${nodeCount}`,
            nodeCount,
            secure: false,
            keySize: nodeCount,
            valSize: nodeCount,
          })
          await populateTrie(generator._nodes)
        })

        for (
          let i = 0;
          i < nodeCount;
          i += nodeCount / (nodeCount > 8 ? 8 : 1)
        ) {
          it(`should correctly get the value of node #${i}`, async () => {
            const test = await generator.makeInclusionProofTest(i)
            expect(
              await Lib_MerkleTrieGeorge.get(test.key, test.root)
            ).to.deep.equal([true, test.val])
          })
          if (i > 3) {
            it(`should be false when calling get on an incorrect key`, async () => {
              const test = await generator.makeInclusionProofTest(i - 1)
              let newKey = test.key.slice(0, test.key.length - 8)
              newKey = newKey.concat('88888888')
              expect(
                await Lib_MerkleTrieGeorge.get(newKey, test.root)
              ).to.deep.equal([false, '0x'])
            })
          }
        }
      })
    }
  })
})
