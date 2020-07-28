/* tslint:disable:no-empty */
import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger } from '@eth-optimism/core-utils'
import { Contract, ContractFactory, Signer } from 'ethers'

/* Internal Imports */
import { makeAddressResolver, AddressResolverMapping } from '../../test-helpers'

/* Logging */
const log = getLogger('partial-state-manager', true)

/* Begin tests */
describe('PartialStateManager', () => {
  let wallet: Signer
  before(async () => {
    ;[wallet] = await ethers.getSigners()
  })

  let resolver: AddressResolverMapping
  before(async () => {
    resolver = await makeAddressResolver(wallet)

    await resolver.addressResolver.setAddress(
      'ExecutionManager',
      await wallet.getAddress()
    )
  })

  let PartialStateManager: ContractFactory
  before(async () => {
    PartialStateManager = await ethers.getContractFactory('PartialStateManager')
  })

  let partialStateManager: Contract
  beforeEach(async () => {
    partialStateManager = await PartialStateManager.deploy(
      resolver.addressResolver.address,
      await wallet.getAddress()
    )
  })

  describe('initNewTransactionExecution()', async () => {
    it('should set the initial state', async () => {})

    it('should fail if not called by the state transitioner', async () => {})
  })

  describe('insertVerifiedStorage(...)', async () => {
    it('should mark a storage slot as verified', async () => {})

    it('should fail if not called by the state transitioner', async () => {})
  })

  describe('insertVerifiedContract(...)', async () => {
    it('should mark a contract as verified', async () => {})

    it('should fail if not called by the state transitioner', async () => {})
  })

  describe('peekUpdatedStorageSlot()', async () => {
    it('should return the last storage slot on the queue', async () => {})

    it('should fail if there are no storage slots to be updated', async () => {})
  })

  describe('popUpdatedStorageSlot()', async () => {
    it('should return the last storage slot on the queue and remove it', async () => {})

    it('should fail if there are no storage slots to be updated', async () => {})

    it('should fail if not called by the state transitioner', async () => {})
  })

  describe('peekUpdatedContract()', async () => {
    it('should return the last contract on the queue', async () => {})

    it('should fail if there are no contracts to be updated', async () => {})
  })

  describe('popUpdatedContract()', async () => {
    it('should return the last contract on the queue and remove it', async () => {})

    it('should fail if there are no contracts to be updated', async () => {})

    it('should fail if not called by the state transitioner', async () => {})
  })

  describe('getStorageView(...)', async () => {
    it('should return the value of a storage slot for a given address', async () => {})

    it('should return null bytes if the storage slot is not set', async () => {})
  })

  describe('getStorage(...)', async () => {
    it('should return the value of a storage slot when it exists', async () => {})

    it('should return null bytes and flag if the storage slot is not set', async () => {})

    it('should fail if not called by the execution manager', async () => {})
  })

  describe('setStorage(...)', async () => {
    it('should set the storage slot for a given address', async () => {})

    it('should not change the counter if the slot has already been touched', async () => {})

    it('should fail if not called by the execution manager', async () => {})
  })

  describe('getOvmContractNonceView(...)', async () => {
    it('should get the nonce for a given address', async () => {})

    it('should return zero if the address has not been set', async () => {})
  })

  describe('getOvmContractNonce(...)', async () => {
    it('should get the nonce for a given address', async () => {})

    it('should return zero and flag if the address has not been set', async () => {})

    it('should fail if not called by the execution manager', async () => {})
  })

  describe('setOvmContractNonce(...)', async () => {
    it('should set the nonce for an address', async () => {})

    it('should not change the counter if the address has already been touched', async () => {})

    it('should fail if not called by the execution manager', async () => {})
  })

  describe('incrementOvmContractNonce(...)', async () => {
    it('should increase the contract nonce by one', async () => {})

    it('should not change the counter if the address has already been touched', async () => {})

    it('should fail if not called by the execution manager', async () => {})
  })

  describe('associateCodeContract(...)', async () => {
    it('should set the code contract address for a given ovm contract', async () => {})

    it('should fail if not called by the execution manager', async () => {})
  })

  describe('associateCreatedContract(...)', async () => {
    it('should mark the contract as verified and set its nonce to zero', async () => {})

    it('should fail if not called by the execution manager', async () => {})
  })

  describe('getCodeContractAddressView(...)', async () => {
    it('should return the code contract for a given ovm contract', async () => {})

    it('should return null bytes if the contract is not associated', async () => {})
  })

  describe('getCodeContractAddressFromOvmAddress(...)', async () => {
    it('should return the code contract address when it exists', async () => {})

    it('should return null bytes and flag when the contract does not exist', async () => {})

    it('should fail if not called by the execution manager', async () => {})
  })

  describe('getCodeContractBytecode(...)', async () => {
    it('should get the bytecode of a contract at the given address', async () => {})

    it('should fail if the contract does not exist', async () => {})
  })

  describe('getCodeContractHash(...)', async () => {
    it('should get the hash of the bytecode at the given address', async () => {})

    it('should fail if the contract does not exist', async () => {})
  })
})
