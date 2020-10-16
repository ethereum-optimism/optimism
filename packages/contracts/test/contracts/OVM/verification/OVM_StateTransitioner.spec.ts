/* tslint:disable:no-empty */
import { expect } from '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { BigNumber, Contract, ContractFactory } from 'ethers'

/* Internal Imports */
import {
  makeAddressManager,
  NON_NULL_BYTES32,
  NON_ZERO_ADDRESS,
  NULL_BYTES32,
  setProxyTarget,
  TrieTestGenerator,
  ZERO_ADDRESS,
} from '../../../helpers'
import {
  MockContract,
  smockit,
  ModifiableContract,
  smoddit,
  ModifiableContractFactory,
} from '@eth-optimism/smock'
import { keccak256 } from 'ethers/lib/utils'

describe('OVM_StateTransitioner', () => {
  let AddressManager: Contract
  before(async () => {
    AddressManager = await makeAddressManager()
  })

  let Mock__OVM_ExecutionManager: MockContract
  let Mock__OVM_StateManagerFactory: MockContract
  let Mock__OVM_StateManager: MockContract
  before(async () => {
    Mock__OVM_ExecutionManager = smockit(
      await ethers.getContractFactory('OVM_ExecutionManager')
    )
    Mock__OVM_StateManagerFactory = smockit(
      await ethers.getContractFactory('OVM_StateManagerFactory')
    )
    Mock__OVM_StateManager = smockit(
      await ethers.getContractFactory('OVM_StateManager')
    )

    await setProxyTarget(
      AddressManager,
      'OVM_ExecutionManager',
      Mock__OVM_ExecutionManager
    )

    await setProxyTarget(
      AddressManager,
      'OVM_StateManagerFactory',
      Mock__OVM_StateManagerFactory
    )

    Mock__OVM_StateManagerFactory.smocked.create.will.return.with(
      Mock__OVM_StateManager.address
    )

    Mock__OVM_StateManager.smocked.putAccount.will.return()
  })

  let Factory__OVM_StateTransitioner: ModifiableContractFactory
  before(async () => {
    Factory__OVM_StateTransitioner = await smoddit('OVM_StateTransitioner')
  })

  let OVM_StateTransitioner: ModifiableContract
  beforeEach(async () => {
    OVM_StateTransitioner = await Factory__OVM_StateTransitioner.deploy(
      AddressManager.address,
      0,
      NULL_BYTES32,
      NULL_BYTES32
    )
  })

  describe('proveContractState', () => {
    let ovmContractAddress = NON_ZERO_ADDRESS
    let ethContractAddress = ZERO_ADDRESS
    let account: any
    beforeEach(() => {
      Mock__OVM_StateManager.smocked.hasAccount.will.return.with(false)
      account = {
        nonce: 0,
        balance: 0,
        storageRoot: NULL_BYTES32,
        codeHash: NULL_BYTES32,
      }
    })

    describe('when provided an invalid code hash', () => {
      beforeEach(() => {
        account.codeHash = NON_NULL_BYTES32
      })

      it('should revert', async () => {
        await expect(
          OVM_StateTransitioner.proveContractState(
            ovmContractAddress,
            ethContractAddress,
            account,
            '0x'
          )
        ).to.be.revertedWith('Invalid code hash provided.')
      })
    })

    describe('when provided a valid code hash', () => {
      beforeEach(async () => {
        ethContractAddress = OVM_StateTransitioner.address
        account.codeHash = keccak256(
          await ethers.provider.getCode(OVM_StateTransitioner.address)
        )
      })

      describe('when provided an invalid account inclusion proof', () => {
        const proof = '0x'

        it('should revert', async () => {
          await expect(
            OVM_StateTransitioner.proveContractState(
              ovmContractAddress,
              ethContractAddress,
              account,
              proof
            )
          ).to.be.reverted
        })
      })

      describe('when provided a valid account inclusion proof', () => {
        let proof: string
        beforeEach(async () => {
          const generator = await TrieTestGenerator.fromAccounts({
            accounts: [
              {
                ...account,
                address: ovmContractAddress,
              },
            ],
            secure: true,
          })

          const test = await generator.makeAccountProofTest(ovmContractAddress)

          proof = test.accountTrieWitness

          OVM_StateTransitioner = await Factory__OVM_StateTransitioner.deploy(
            AddressManager.address,
            0,
            test.accountTrieRoot,
            NULL_BYTES32
          )
        })

        it('should put the account in the state manager', async () => {
          await expect(
            OVM_StateTransitioner.proveContractState(
              ovmContractAddress,
              ethContractAddress,
              account,
              proof
            )
          ).to.not.be.reverted

          expect(
            Mock__OVM_StateManager.smocked.putAccount.calls[0]
          ).to.deep.equal([
            NON_ZERO_ADDRESS,
            [
              BigNumber.from(account.nonce),
              BigNumber.from(account.balance),
              account.storageRoot,
              account.codeHash,
              ethContractAddress,
              false,
            ],
          ])
        })
      })
    })
  })

  describe('proveStorageSlot', () => {
    beforeEach(() => {
      Mock__OVM_StateManager.smocked.hasContractStorage.will.return.with(false)
    })

    describe('when the corresponding account is not proven', () => {
      beforeEach(() => {
        Mock__OVM_StateManager.smocked.hasAccount.will.return.with(false)
      })

      it('should revert', async () => {
        await expect(
          OVM_StateTransitioner.proveStorageSlot(
            NON_ZERO_ADDRESS,
            NON_NULL_BYTES32,
            NON_NULL_BYTES32,
            '0x'
          )
        ).to.be.revertedWith(
          'Contract must be verified before proving a storage slot.'
        )
      })
    })

    describe('when the corresponding account is proven', () => {
      beforeEach(() => {
        Mock__OVM_StateManager.smocked.hasAccount.will.return.with(true)
      })

      describe('when provided an invalid slot inclusion proof', () => {
        let key = keccak256('0x1234')
        let val = keccak256('0x5678')
        let proof = '0x'
        beforeEach(async () => {
          const generator = await TrieTestGenerator.fromNodes({
            nodes: [
              {
                key,
                val,
              },
            ],
            secure: true,
          })

          const test = await generator.makeInclusionProofTest(0)

          Mock__OVM_StateManager.smocked.getAccountStorageRoot.will.return.with(
            test.root
          )
        })

        it('should revert', async () => {
          await expect(
            OVM_StateTransitioner.proveStorageSlot(
              ZERO_ADDRESS,
              key,
              val,
              proof
            )
          ).to.be.reverted
        })
      })

      describe('when provided a valid slot inclusion proof', () => {
        let key = keccak256('0x1234')
        let val = keccak256('0x5678')
        let proof: string
        beforeEach(async () => {
          const generator = await TrieTestGenerator.fromNodes({
            nodes: [
              {
                key,
                val,
              },
            ],
            secure: true,
          })

          const test = await generator.makeInclusionProofTest(0)
          proof = test.proof

          Mock__OVM_StateManager.smocked.getAccountStorageRoot.will.return.with(
            test.root
          )
        })

        it('should insert the storage slot', async () => {
          await expect(
            OVM_StateTransitioner.proveStorageSlot(
              ZERO_ADDRESS,
              key,
              val,
              proof
            )
          ).to.not.be.reverted

          expect(
            Mock__OVM_StateManager.smocked.putContractStorage.calls[0]
          ).to.deep.equal([ZERO_ADDRESS, key, val])
        })
      })
    })
  })

  describe('applyTransaction', () => {
    // TODO
  })

  describe('commitContractState', () => {
    beforeEach(async () => {
      OVM_StateTransitioner.smodify.set({
        phase: 1,
      })
    })

    let ovmContractAddress = NON_ZERO_ADDRESS
    let account: any
    beforeEach(() => {
      account = {
        nonce: 0,
        balance: 0,
        storageRoot: NULL_BYTES32,
        codeHash: NULL_BYTES32,
        ethAddress: ZERO_ADDRESS,
        isFresh: false,
      }
      Mock__OVM_StateManager.smocked.hasAccount.will.return.with(false)
      Mock__OVM_StateManager.smocked.getAccount.will.return.with(account)
    })

    describe('when the account was not changed or has already been committed', () => {
      before(() => {
        Mock__OVM_StateManager.smocked.commitAccount.will.return.with(false)
      })

      it('should revert', async () => {
        await expect(
          OVM_StateTransitioner.commitContractState(ovmContractAddress, '0x')
        ).to.be.revertedWith(
          'Account was not changed or has already been committed.'
        )
      })
    })

    describe('when the account was changed or has not already been committed', () => {
      before(() => {
        Mock__OVM_StateManager.smocked.commitAccount.will.return.with(true)
      })

      describe('when given an valid update proof', () => {
        let proof: string
        let postStateRoot: string
        beforeEach(async () => {
          const generator = await TrieTestGenerator.fromAccounts({
            accounts: [
              {
                ...account,
                nonce: 10,
                address: ovmContractAddress,
              },
            ],
            secure: true,
          })

          const test = await generator.makeAccountUpdateTest(
            ovmContractAddress,
            account
          )

          proof = test.accountTrieWitness
          postStateRoot = test.newAccountTrieRoot

          OVM_StateTransitioner.smodify.put({
            postStateRoot: test.accountTrieRoot,
          })
        })

        it('should update the post state root', async () => {
          await expect(
            OVM_StateTransitioner.commitContractState(ovmContractAddress, proof)
          ).to.not.be.reverted

          expect(await OVM_StateTransitioner.getPostStateRoot()).to.equal(
            postStateRoot
          )
        })
      })
    })
  })

  describe('commitStorageSlot', () => {
    beforeEach(() => {
      OVM_StateTransitioner.smodify.set({
        phase: 1,
      })
    })

    let ovmContractAddress = NON_ZERO_ADDRESS
    let account: any
    let key = keccak256('0x1234')
    let val = keccak256('0x5678')
    let newVal = keccak256('0x4321')
    beforeEach(() => {
      account = {
        nonce: 0,
        balance: 0,
        storageRoot: NULL_BYTES32,
        codeHash: NULL_BYTES32,
      }

      Mock__OVM_StateManager.smocked.getAccount.will.return.with({
        ...account,
        ethAddress: ZERO_ADDRESS,
        isFresh: false,
      })

      Mock__OVM_StateManager.smocked.getContractStorage.will.return.with(val)
    })

    describe('when the slot was not changed or was already committed', () => {
      beforeEach(() => {
        Mock__OVM_StateManager.smocked.commitContractStorage.will.return.with(
          false
        )
      })

      it('should revert', async () => {
        await expect(
          OVM_StateTransitioner.commitStorageSlot(
            ovmContractAddress,
            key,
            '0x',
            '0x'
          )
        ).to.be.revertedWith(
          'Storage slot was not changed or has already been committed.'
        )
      })
    })

    describe('when the slot was changed or not already committed', () => {
      beforeEach(() => {
        Mock__OVM_StateManager.smocked.commitContractStorage.will.return.with(
          true
        )
      })

      describe('with a valid proof', () => {
        let accountTrieProof: string
        let storageTrieProof: string
        let postStateRoot: string
        beforeEach(async () => {
          const storageGenerator = await TrieTestGenerator.fromNodes({
            nodes: [
              {
                key,
                val,
              },
            ],
            secure: true,
          })

          const storageTest = await storageGenerator.makeNodeUpdateTest(
            key,
            newVal
          )

          const generator = await TrieTestGenerator.fromAccounts({
            accounts: [
              {
                ...account,
                storageRoot: storageTest.root,
                address: ovmContractAddress,
              },
            ],
            secure: true,
          })

          const test = await generator.makeAccountUpdateTest(
            ovmContractAddress,
            {
              ...account,
              storageRoot: storageTest.newRoot,
            }
          )

          Mock__OVM_StateManager.smocked.getAccount.will.return.with({
            ...account,
            storageRoot: storageTest.root,
            ethAddress: ZERO_ADDRESS,
            isFresh: false,
          })

          accountTrieProof = test.accountTrieWitness
          storageTrieProof = storageTest.proof

          postStateRoot = test.newAccountTrieRoot

          OVM_StateTransitioner.smodify.put({
            postStateRoot: test.accountTrieRoot,
          })
        })

        it('should commit the slot and update the state', async () => {
          await expect(
            OVM_StateTransitioner.commitStorageSlot(
              ovmContractAddress,
              key,
              accountTrieProof,
              storageTrieProof
            )
          ).to.not.be.reverted
        })
      })
    })
  })

  describe('completeTransition', () => {
    beforeEach(() => {
      OVM_StateTransitioner.smodify.set({
        phase: 1,
      })
    })

    describe('when there are uncommitted accounts', () => {
      beforeEach(() => {
        Mock__OVM_StateManager.smocked.getTotalUncommittedAccounts.will.return.with(
          1
        )
        Mock__OVM_StateManager.smocked.getTotalUncommittedContractStorage.will.return.with(
          0
        )
      })

      it('should revert', async () => {
        await expect(
          OVM_StateTransitioner.completeTransition()
        ).to.be.revertedWith(
          'All accounts must be committed before completing a transition.'
        )
      })
    })

    describe('when there are uncommitted storage slots', () => {
      beforeEach(() => {
        Mock__OVM_StateManager.smocked.getTotalUncommittedAccounts.will.return.with(
          0
        )
        Mock__OVM_StateManager.smocked.getTotalUncommittedContractStorage.will.return.with(
          1
        )
      })

      it('should revert', async () => {
        await expect(
          OVM_StateTransitioner.completeTransition()
        ).to.be.revertedWith(
          'All storage must be committed before completing a transition.'
        )
      })
    })

    describe('when all state changes are committed', () => {
      beforeEach(() => {
        Mock__OVM_StateManager.smocked.getTotalUncommittedAccounts.will.return.with(
          0
        )
        Mock__OVM_StateManager.smocked.getTotalUncommittedContractStorage.will.return.with(
          0
        )
      })

      it('should complete the transition', async () => {
        await expect(OVM_StateTransitioner.completeTransition()).to.not.be
          .reverted

        expect(await OVM_StateTransitioner.isComplete()).to.equal(true)
      })
    })
  })
})
