import { expect } from '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract } from 'ethers'

/* Internal Imports */
import { TrieTestGenerator, NON_NULL_BYTES32 } from '../../../helpers'
import { keccak256 } from 'ethers/lib/utils'

const makeDummyAccounts = (count: number): any[] => {
  return [...Array(count)].map((x, idx) => {
    return {
      address: '0xc0de' + `${idx.toString(16)}`.padStart(36, '0'),
      nonce: 0,
      balance: 0,
      codeHash: null,
      storage: [
        {
          key: keccak256('0x1234'),
          val: keccak256('0x5678'),
        },
      ],
    }
  })
}

const NODE_COUNTS = [1, 2, 128, 256, 512, 1024, 2048, 4096]

describe('Lib_EthMerkleTrie', () => {
  let Lib_EthMerkleTrie: Contract
  before(async () => {
    Lib_EthMerkleTrie = await (
      await ethers.getContractFactory('TestLib_EthMerkleTrie')
    ).deploy()
  })

  describe('proveAccountStorageSlotValue', () => {})

  describe('updateAccountStorageSlotValue', () => {})

  describe('proveAccountState', () => {
    for (const nodeCount of NODE_COUNTS) {
      describe(`inside a trie with ${nodeCount} nodes`, () => {
        let generator: TrieTestGenerator
        before(async () => {
          generator = await TrieTestGenerator.fromAccounts({
            accounts: makeDummyAccounts(nodeCount),
            secure: true,
          })
        })

        for (
          let i = 0;
          i < nodeCount;
          i += nodeCount / (nodeCount > 8 ? 8 : 1)
        ) {
          it(`should correctly prove inclusion for node #${i}`, async () => {
            const test = await generator.makeAccountProofTest(i)

            expect(
              await Lib_EthMerkleTrie.proveAccountState(
                test.address,
                test.account,
                test.accountTrieWitness,
                test.accountTrieRoot
              )
            ).to.equal(true)
          })
        }
      })
    }
  })

  describe.only('updateAccountState', () => {
    for (const nodeCount of NODE_COUNTS) {
      describe(`inside a trie with ${nodeCount} nodes`, () => {
        let generator: TrieTestGenerator
        before(async () => {
          generator = await TrieTestGenerator.fromAccounts({
            accounts: makeDummyAccounts(nodeCount),
            secure: true,
          })
        })

        for (
          let i = 0;
          i < nodeCount;
          i += nodeCount / (nodeCount > 8 ? 8 : 1)
        ) {
          it(`should correctly update node #${i}`, async () => {
            const test = await generator.makeAccountUpdateTest(i, {
              nonce: 1234,
              balance: 5678,
              codeHash: NON_NULL_BYTES32,
              storageRoot: NON_NULL_BYTES32,
            })

            expect(
              await Lib_EthMerkleTrie.updateAccountState(
                test.address,
                test.account,
                test.accountTrieWitness,
                test.accountTrieRoot
              )
            ).to.equal(test.newAccountTrieRoot)
          })
        }
      })
    }
  })
})
