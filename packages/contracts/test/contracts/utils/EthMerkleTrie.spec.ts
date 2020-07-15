import { expect } from '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract, ContractFactory } from 'ethers'
import { keccak256 } from 'ethers/utils'

import {
  makeAccountStorageProofTest,
  makeAccountStorageUpdateTest,
} from '../../test-helpers'

const DUMMY_ACCOUNT_ADDRESSES = [
  '0x548855F6073c3430285c61Ed0ABf62F12084aA41',
  '0xD80e66Cbc34F06d24a0a4fDdD6f2aDB41ac1517D',
  '0x069889F3DC507DdA244d19b5f24caDCDd2a735c2',
  '0x808E5eCe9a8EA2cdce515764139Ee24bEF7098b4',
]

const NULL_BYTES32 = `0x${'00'.repeat(32)}`

describe('EthMerkleTrie', () => {
  let Trie: ContractFactory
  let trie: Contract
  before(async () => {
    Trie = await ethers.getContractFactory('EthMerkleTrie')
    trie = await Trie.deploy()
  })

  describe('proveAccountStorageSlotValue', () => {
    it('should verify proofs with a single account and a single storage slot', async () => {
      const test = await makeAccountStorageProofTest(
        {
          [DUMMY_ACCOUNT_ADDRESSES[0]]: {
            state: {
              nonce: 0,
              balance: 0,
              storageRoot: null,
              codeHash: null,
            },
            storage: [
              {
                key: keccak256('0x123'),
                val: keccak256('0x456'),
              },
            ],
          },
        },
        DUMMY_ACCOUNT_ADDRESSES[0],
        keccak256('0x123')
      )
      expect(
        await trie.proveAccountStorageSlotValue(
          test.address,
          test.key,
          test.val,
          test.stateTrieWitness,
          test.storageTrieWitness,
          test.stateTrieRoot
        )
      ).to.equal(true)
    })

    it('should verify proofs with a single account and multiple storage slots', async () => {
      const test = await makeAccountStorageProofTest(
        {
          [DUMMY_ACCOUNT_ADDRESSES[0]]: {
            state: {
              nonce: 0,
              balance: 0,
              storageRoot: null,
              codeHash: null,
            },
            storage: [
              {
                key: keccak256('0x123'),
                val: keccak256('0x456'),
              },
              {
                key: keccak256('0x123123'),
                val: keccak256('0x456456'),
              },
              {
                key: keccak256('0x123123123'),
                val: keccak256('0x456456456'),
              },
            ],
          },
        },
        DUMMY_ACCOUNT_ADDRESSES[0],
        keccak256('0x123')
      )
      expect(
        await trie.proveAccountStorageSlotValue(
          test.address,
          test.key,
          test.val,
          test.stateTrieWitness,
          test.storageTrieWitness,
          test.stateTrieRoot
        )
      ).to.equal(true)
    })

    it('should verify proofs with multiple accounts and multiple storage slots', async () => {
      const test = await makeAccountStorageProofTest(
        {
          [DUMMY_ACCOUNT_ADDRESSES[0]]: {
            state: {
              nonce: 0,
              balance: 0,
              storageRoot: null,
              codeHash: null,
            },
            storage: [
              {
                key: keccak256('0x123'),
                val: keccak256('0x456'),
              },
              {
                key: keccak256('0x123123'),
                val: keccak256('0x456456'),
              },
              {
                key: keccak256('0x123123123'),
                val: keccak256('0x456456456'),
              },
            ],
          },
          [DUMMY_ACCOUNT_ADDRESSES[1]]: {
            state: {
              nonce: 0,
              balance: 0,
              storageRoot: null,
              codeHash: null,
            },
            storage: [
              {
                key: keccak256('0x123'),
                val: keccak256('0x456'),
              },
              {
                key: keccak256('0x123123'),
                val: keccak256('0x456456'),
              },
              {
                key: keccak256('0x123123123'),
                val: keccak256('0x456456456'),
              },
            ],
          },
          [DUMMY_ACCOUNT_ADDRESSES[2]]: {
            state: {
              nonce: 0,
              balance: 0,
              storageRoot: null,
              codeHash: null,
            },
            storage: [
              {
                key: keccak256('0x123'),
                val: keccak256('0x456'),
              },
              {
                key: keccak256('0x123123'),
                val: keccak256('0x456456'),
              },
              {
                key: keccak256('0x123123123'),
                val: keccak256('0x456456456'),
              },
            ],
          },
        },
        DUMMY_ACCOUNT_ADDRESSES[0],
        keccak256('0x123')
      )
      expect(
        await trie.proveAccountStorageSlotValue(
          test.address,
          test.key,
          test.val,
          test.stateTrieWitness,
          test.storageTrieWitness,
          test.stateTrieRoot
        )
      ).to.equal(true)
    })
  })

  describe('updateAccountStorageSlotValue', () => {
    it('should update values with a single account and a single storage slot', async () => {
      const test = await makeAccountStorageUpdateTest(
        {
          [DUMMY_ACCOUNT_ADDRESSES[0]]: {
            state: {
              nonce: 0,
              balance: 0,
              storageRoot: null,
              codeHash: null,
            },
            storage: [
              {
                key: keccak256('0x123'),
                val: keccak256('0x456'),
              },
            ],
          },
        },
        DUMMY_ACCOUNT_ADDRESSES[0],
        keccak256('0x123'),
        keccak256('0x789')
      )
      expect(
        await trie.updateAccountStorageSlotValue(
          test.address,
          test.key,
          test.val,
          test.stateTrieWitness,
          test.storageTrieWitness,
          test.stateTrieRoot
        )
      ).to.equal(test.newStateTrieRoot)
    })

    it('should update values with a single account and multiple storage slots', async () => {
      const test = await makeAccountStorageUpdateTest(
        {
          [DUMMY_ACCOUNT_ADDRESSES[0]]: {
            state: {
              nonce: 0,
              balance: 0,
              storageRoot: null,
              codeHash: null,
            },
            storage: [
              {
                key: keccak256('0x123'),
                val: keccak256('0x456'),
              },
              {
                key: keccak256('0x123123'),
                val: keccak256('0x456456'),
              },
              {
                key: keccak256('0x123123123'),
                val: keccak256('0x456456456'),
              },
            ],
          },
        },
        DUMMY_ACCOUNT_ADDRESSES[0],
        keccak256('0x123'),
        keccak256('0x789')
      )
      expect(
        await trie.updateAccountStorageSlotValue(
          test.address,
          test.key,
          test.val,
          test.stateTrieWitness,
          test.storageTrieWitness,
          test.stateTrieRoot
        )
      ).to.equal(test.newStateTrieRoot)
    })

    it('should update values with multiple accounts and multiple storage slots', async () => {
      const test = await makeAccountStorageUpdateTest(
        {
          [DUMMY_ACCOUNT_ADDRESSES[0]]: {
            state: {
              nonce: 0,
              balance: 0,
              storageRoot: null,
              codeHash: null,
            },
            storage: [
              {
                key: keccak256('0x123'),
                val: keccak256('0x456'),
              },
              {
                key: keccak256('0x123123'),
                val: keccak256('0x456456'),
              },
              {
                key: keccak256('0x123123123'),
                val: keccak256('0x456456456'),
              },
            ],
          },
          [DUMMY_ACCOUNT_ADDRESSES[1]]: {
            state: {
              nonce: 0,
              balance: 0,
              storageRoot: null,
              codeHash: null,
            },
            storage: [
              {
                key: keccak256('0x123'),
                val: keccak256('0x456'),
              },
              {
                key: keccak256('0x123123'),
                val: keccak256('0x456456'),
              },
              {
                key: keccak256('0x123123123'),
                val: keccak256('0x456456456'),
              },
            ],
          },
          [DUMMY_ACCOUNT_ADDRESSES[2]]: {
            state: {
              nonce: 0,
              balance: 0,
              storageRoot: null,
              codeHash: null,
            },
            storage: [
              {
                key: keccak256('0x123'),
                val: keccak256('0x456'),
              },
              {
                key: keccak256('0x123123'),
                val: keccak256('0x456456'),
              },
              {
                key: keccak256('0x123123123'),
                val: keccak256('0x456456456'),
              },
            ],
          },
        },
        DUMMY_ACCOUNT_ADDRESSES[0],
        keccak256('0x123'),
        keccak256('0x789')
      )
      expect(
        await trie.updateAccountStorageSlotValue(
          test.address,
          test.key,
          test.val,
          test.stateTrieWitness,
          test.storageTrieWitness,
          test.stateTrieRoot
        )
      ).to.equal(test.newStateTrieRoot)
    })
  })

  describe('proveAccountState', () => {
    it('should prove a slot in a trie with a single account', async () => {
      const accountState = {
        nonce: 0,
        balance: 0,
        storageRoot: null,
        codeHash: null,
      }
      const test = await makeAccountStorageProofTest(
        {
          [DUMMY_ACCOUNT_ADDRESSES[0]]: {
            state: accountState,
            storage: [
              {
                key: keccak256('0x123'),
                val: keccak256('0x456'),
              },
            ],
          },
        },
        DUMMY_ACCOUNT_ADDRESSES[0],
        keccak256('0x123')
      )
      expect(
        await trie.proveAccountState(
          test.address,
          {
            nonce: accountState.nonce,
            balance: accountState.balance,
            storageRoot: accountState.storageRoot || NULL_BYTES32,
            codeHash: accountState.codeHash || NULL_BYTES32,
          },
          {
            checkNonce: true,
            checkBalance: true,
            checkStorageRoot: true,
            checkCodeHash: true,
          },
          test.stateTrieWitness,
          test.stateTrieRoot
        )
      ).to.equal(true)
    })

    it('should prove a slot in a trie with multiple accounts', async () => {
      const accountState = {
        nonce: 0,
        balance: 0,
        storageRoot: null,
        codeHash: null,
      }
      const test = await makeAccountStorageProofTest(
        {
          [DUMMY_ACCOUNT_ADDRESSES[0]]: {
            state: accountState,
            storage: [
              {
                key: keccak256('0x123'),
                val: keccak256('0x456'),
              },
              {
                key: keccak256('0x123123'),
                val: keccak256('0x456456'),
              },
              {
                key: keccak256('0x123123123'),
                val: keccak256('0x456456456'),
              },
            ],
          },
          [DUMMY_ACCOUNT_ADDRESSES[1]]: {
            state: {
              nonce: 0,
              balance: 0,
              storageRoot: null,
              codeHash: null,
            },
            storage: [
              {
                key: keccak256('0x123'),
                val: keccak256('0x456'),
              },
              {
                key: keccak256('0x123123'),
                val: keccak256('0x456456'),
              },
              {
                key: keccak256('0x123123123'),
                val: keccak256('0x456456456'),
              },
            ],
          },
          [DUMMY_ACCOUNT_ADDRESSES[2]]: {
            state: {
              nonce: 0,
              balance: 0,
              storageRoot: null,
              codeHash: null,
            },
            storage: [
              {
                key: keccak256('0x123'),
                val: keccak256('0x456'),
              },
              {
                key: keccak256('0x123123'),
                val: keccak256('0x456456'),
              },
              {
                key: keccak256('0x123123123'),
                val: keccak256('0x456456456'),
              },
            ],
          },
        },
        DUMMY_ACCOUNT_ADDRESSES[0],
        keccak256('0x123')
      )
      expect(
        await trie.proveAccountState(
          test.address,
          {
            nonce: accountState.nonce,
            balance: accountState.balance,
            storageRoot: accountState.storageRoot || NULL_BYTES32,
            codeHash: accountState.codeHash || NULL_BYTES32,
          },
          {
            checkNonce: true,
            checkBalance: true,
            checkStorageRoot: true,
            checkCodeHash: true,
          },
          test.stateTrieWitness,
          test.stateTrieRoot
        )
      ).to.equal(true)
    })
  })

  describe('updateAccountState', () => {
    it('should update a slot in a trie with a single account', async () => {
      const accountState = {
        nonce: 0,
        balance: 0,
        storageRoot: null,
        codeHash: null,
      }
      const newAccountState = {
        nonce: 123,
        balance: 456,
        storageRoot: keccak256('0x1234'),
        codeHash: keccak256('0x5678'),
      }
      const test = await makeAccountStorageUpdateTest(
        {
          [DUMMY_ACCOUNT_ADDRESSES[0]]: {
            state: accountState,
            storage: [
              {
                key: keccak256('0x123'),
                val: keccak256('0x456'),
              },
            ],
          },
        },
        DUMMY_ACCOUNT_ADDRESSES[0],
        '',
        '',
        newAccountState
      )
      expect(
        await trie.updateAccountState(
          test.address,
          {
            nonce: newAccountState.nonce,
            balance: newAccountState.balance,
            storageRoot: newAccountState.storageRoot || NULL_BYTES32,
            codeHash: newAccountState.codeHash || NULL_BYTES32,
          },
          {
            checkNonce: true,
            checkBalance: true,
            checkStorageRoot: true,
            checkCodeHash: true,
          },
          test.stateTrieWitness,
          test.stateTrieRoot
        )
      ).to.equal(test.newStateTrieRoot)
    })

    it('should update a slot in a trie with multiple accounts', async () => {
      const accountState = {
        nonce: 0,
        balance: 0,
        storageRoot: null,
        codeHash: null,
      }
      const newAccountState = {
        nonce: 123,
        balance: 456,
        storageRoot: keccak256('0x1234'),
        codeHash: keccak256('0x5678'),
      }
      const test = await makeAccountStorageUpdateTest(
        {
          [DUMMY_ACCOUNT_ADDRESSES[0]]: {
            state: accountState,
            storage: [
              {
                key: keccak256('0x123'),
                val: keccak256('0x456'),
              },
              {
                key: keccak256('0x123123'),
                val: keccak256('0x456456'),
              },
              {
                key: keccak256('0x123123123'),
                val: keccak256('0x456456456'),
              },
            ],
          },
          [DUMMY_ACCOUNT_ADDRESSES[1]]: {
            state: {
              nonce: 0,
              balance: 0,
              storageRoot: null,
              codeHash: null,
            },
            storage: [
              {
                key: keccak256('0x123'),
                val: keccak256('0x456'),
              },
              {
                key: keccak256('0x123123'),
                val: keccak256('0x456456'),
              },
              {
                key: keccak256('0x123123123'),
                val: keccak256('0x456456456'),
              },
            ],
          },
          [DUMMY_ACCOUNT_ADDRESSES[2]]: {
            state: {
              nonce: 0,
              balance: 0,
              storageRoot: null,
              codeHash: null,
            },
            storage: [
              {
                key: keccak256('0x123'),
                val: keccak256('0x456'),
              },
              {
                key: keccak256('0x123123'),
                val: keccak256('0x456456'),
              },
              {
                key: keccak256('0x123123123'),
                val: keccak256('0x456456456'),
              },
            ],
          },
        },
        DUMMY_ACCOUNT_ADDRESSES[0],
        '',
        '',
        newAccountState
      )
      expect(
        await trie.updateAccountState(
          test.address,
          {
            nonce: newAccountState.nonce,
            balance: newAccountState.balance,
            storageRoot: newAccountState.storageRoot || NULL_BYTES32,
            codeHash: newAccountState.codeHash || NULL_BYTES32,
          },
          {
            checkNonce: true,
            checkBalance: true,
            checkStorageRoot: true,
            checkCodeHash: true,
          },
          test.stateTrieWitness,
          test.stateTrieRoot
        )
      ).to.equal(test.newStateTrieRoot)
    })
  })
})
