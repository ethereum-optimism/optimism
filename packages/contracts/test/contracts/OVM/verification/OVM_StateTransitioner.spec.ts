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
import { MockContract, smockit } from '@eth-optimism/smock'
import { keccak256 } from 'ethers/lib/utils'

describe.skip('OVM_StateTransitioner', () => {
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

  let Factory__OVM_StateTransitioner: ContractFactory
  before(async () => {
    Factory__OVM_StateTransitioner = await ethers.getContractFactory(
      'OVM_StateTransitioner'
    )
  })

  let OVM_StateTransitioner: Contract
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
              account.ethAddress,
              account.isFresh,
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
        Mock__OVM_StateManager.smocked.hasAccount.will.return.with(false)
      })

      describe('when provided an invalid slot inclusion proof', () => {
        let account: any
        let proof: string
        beforeEach(async () => {
          account = {
            nonce: 0,
            balance: 0,
            storageRoot: NULL_BYTES32,
            codeHash: NULL_BYTES32,
          }

          const generator = await TrieTestGenerator.fromAccounts({
            accounts: [
              {
                ...account,
                address: NON_ZERO_ADDRESS,
                storage: [
                  {
                    key: keccak256('0x1234'),
                    val: keccak256('0x5678'),
                  },
                ],
              },
            ],
            secure: true,
          })

          const test = await generator.makeAccountProofTest(NON_ZERO_ADDRESS)

          proof = test.accountTrieWitness

          OVM_StateTransitioner = await Factory__OVM_StateTransitioner.deploy(
            AddressManager.address,
            0,
            test.accountTrieRoot,
            NULL_BYTES32
          )
        })

        it('should revert', async () => {})
      })

      describe('when provided a valid slot inclusion proof', () => {})
    })
  })

  describe('applyTransaction', () => {
    // TODO
  })

  describe('commitContractState', () => {
    describe('when the account was not changed', () => {
      it('should revert', async () => {})
    })

    describe('when the account was changed', () => {
      describe('when the account has not been committed', () => {
        it('should commit the account and update the state', async () => {})
      })

      describe('when the account was already committed', () => {
        it('should revert', () => {})
      })
    })
  })

  describe('commitStorageSlot', () => {
    describe('when the slot was not changed', () => {
      it('should revert', async () => {})
    })

    describe('when the slot was changed', () => {
      describe('when the slot has not been committed', () => {
        it('should commit the slot and update the state', async () => {})
      })

      describe('when the slot was already committed', () => {
        it('should revert', () => {})
      })
    })
  })

  describe('completeTransition', () => {
    describe('when there are uncommitted accounts', () => {
      it('should revert', async () => {})
    })

    describe('when there are uncommitted storage slots', () => {
      it('should revert', async () => {})
    })

    describe('when all state changes are committed', () => {})
  })
})
