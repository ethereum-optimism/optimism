import '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { getLogger, TestUtils, NULL_ADDRESS } from '@eth-optimism/core-utils'
import { Contract, ContractFactory, Signer } from 'ethers'

/* Internal Imports */
import { makeAddressResolver, AddressResolverMapping } from '../../test-helpers'

/* Logging */
const log = getLogger('partial-state-manager', true)

/* Begin tests */
describe('PartialStateManager', () => {
  const DUMMY_CONTRACTS = [
    '0x' + '01'.repeat(20),
    '0x' + '02'.repeat(20),
    '0x' + '03'.repeat(20),
  ]

  const DUMMY_CODE_CONTRACTS = [
    '0x' + '10'.repeat(20),
    '0x' + '20'.repeat(20),
    '0x' + '30'.repeat(20),
  ]

  const DUMMY_SLOTS = [
    '0x' + '04'.repeat(32),
    '0x' + '05'.repeat(32),
    '0x' + '06'.repeat(32),
  ]

  const DUMMY_VALUES = [
    '0x' + '07'.repeat(32),
    '0x' + '08'.repeat(32),
    '0x' + '09'.repeat(32),
  ]

  const NULL_BYTES32 = '0x' + '00'.repeat(32)
  const RLP_NULL_HASH =
    '0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470'

  let wallet: Signer
  let randomWallet: Signer
  before(async () => {
    ;[wallet, randomWallet] = await ethers.getSigners()
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
    it('should set the initial state', async () => {
      await partialStateManager.initNewTransactionExecution()

      const existsInvalidStateAccessFlag = await partialStateManager.existsInvalidStateAccessFlag()
      const updatedStorageSlotCounter = await partialStateManager.updatedStorageSlotCounter()
      const updatedContractsCounter = await partialStateManager.updatedContractsCounter()

      existsInvalidStateAccessFlag.should.equal(false)
      updatedStorageSlotCounter.should.equal(0)
      updatedContractsCounter.should.equal(0)
    })

    it('should fail if not called by the state transitioner', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await partialStateManager.initNewTransactionExecution({
          from: await randomWallet.getAddress(),
        })
      })
    })
  })

  describe('insertVerifiedStorage(...)', async () => {
    it('should mark a storage slot as verified', async () => {
      await partialStateManager.insertVerifiedStorage(
        DUMMY_CONTRACTS[0],
        DUMMY_SLOTS[0],
        DUMMY_VALUES[0]
      )

      const isVerifiedStorage = await partialStateManager.isVerifiedStorage(
        DUMMY_CONTRACTS[0],
        DUMMY_SLOTS[0]
      )
      const ovmContractStorage = await partialStateManager.ovmContractStorage(
        DUMMY_CONTRACTS[0],
        DUMMY_SLOTS[0]
      )

      isVerifiedStorage.should.equal(true)
      ovmContractStorage.should.equal(DUMMY_VALUES[0])
    })

    it('should fail if not called by the state transitioner', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await partialStateManager.insertVerifiedStorage(
          DUMMY_CONTRACTS[0],
          DUMMY_SLOTS[0],
          DUMMY_VALUES[0],
          {
            from: await randomWallet.getAddress(),
          }
        )
      })
    })
  })

  describe('insertVerifiedContract(...)', async () => {
    it('should mark a contract as verified', async () => {
      const nonce = 1234
      await partialStateManager.insertVerifiedContract(
        DUMMY_CONTRACTS[0],
        DUMMY_CODE_CONTRACTS[0],
        nonce
      )

      const isVerifiedContract = await partialStateManager.isVerifiedContract(
        DUMMY_CONTRACTS[0]
      )
      const ovmContractNonce = await partialStateManager.ovmContractNonces(
        DUMMY_CONTRACTS[0]
      )
      const codeContractAddress = await partialStateManager.ovmAddressToCodeContractAddress(
        DUMMY_CONTRACTS[0]
      )

      isVerifiedContract.should.equal(true)
      ovmContractNonce.should.equal(nonce)
      codeContractAddress.should.equal(DUMMY_CODE_CONTRACTS[0])
    })

    it('should fail if not called by the state transitioner', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        const nonce = 1234
        await partialStateManager.insertVerifiedContract(
          DUMMY_CONTRACTS[0],
          DUMMY_CODE_CONTRACTS[0],
          nonce,
          {
            from: await randomWallet.getAddress(),
          }
        )
      })
    })
  })

  describe('setStorage(...)', async () => {
    it('should set the storage slot for a given address', async () => {
      await partialStateManager.setStorage(
        DUMMY_CONTRACTS[0],
        DUMMY_SLOTS[0],
        DUMMY_VALUES[0]
      )

      const slotContract = await partialStateManager.updatedStorageSlotContract(
        0
      )
      const slotKey = await partialStateManager.updatedStorageSlotKey(0)
      const slotTouched = await partialStateManager.storageSlotTouched(
        DUMMY_CONTRACTS[0],
        DUMMY_SLOTS[0]
      )
      const slotValue = await partialStateManager.ovmContractStorage(
        DUMMY_CONTRACTS[0],
        DUMMY_SLOTS[0]
      )
      const counter = await partialStateManager.updatedStorageSlotCounter()

      slotContract.should.equal(DUMMY_CONTRACTS[0] + '00'.repeat(12))
      slotKey.should.equal(DUMMY_SLOTS[0])
      slotTouched.should.equal(true)
      slotValue.should.equal(DUMMY_VALUES[0])
      counter.should.equal(1)
    })

    it('should not change the counter if the slot has already been touched', async () => {
      await partialStateManager.setStorage(
        DUMMY_CONTRACTS[0],
        DUMMY_SLOTS[0],
        DUMMY_VALUES[0]
      )

      await partialStateManager.setStorage(
        DUMMY_CONTRACTS[0],
        DUMMY_SLOTS[0],
        DUMMY_VALUES[1]
      )

      const slotContract = await partialStateManager.updatedStorageSlotContract(
        0
      )
      const slotKey = await partialStateManager.updatedStorageSlotKey(0)
      const slotTouched = await partialStateManager.storageSlotTouched(
        DUMMY_CONTRACTS[0],
        DUMMY_SLOTS[0]
      )
      const slotValue = await partialStateManager.ovmContractStorage(
        DUMMY_CONTRACTS[0],
        DUMMY_SLOTS[0]
      )
      const counter = await partialStateManager.updatedStorageSlotCounter()

      slotContract.should.equal(DUMMY_CONTRACTS[0] + '00'.repeat(12))
      slotKey.should.equal(DUMMY_SLOTS[0])
      slotTouched.should.equal(true)
      slotValue.should.equal(DUMMY_VALUES[1])
      counter.should.equal(1)
    })

    it('should fail if not called by the execution manager', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await partialStateManager.setStorage(
          DUMMY_CONTRACTS[0],
          DUMMY_SLOTS[0],
          DUMMY_VALUES[0],
          {
            from: await randomWallet.getAddress(),
          }
        )
      })
    })
  })

  describe('getStorageView(...)', async () => {
    it('should return the value of a storage slot for a given address', async () => {
      await partialStateManager.setStorage(
        DUMMY_CONTRACTS[0],
        DUMMY_SLOTS[0],
        DUMMY_VALUES[0]
      )

      const value = await partialStateManager.getStorageView(
        DUMMY_CONTRACTS[0],
        DUMMY_SLOTS[0]
      )

      value.should.equal(DUMMY_VALUES[0])
    })

    it('should return null bytes if the storage slot is not set', async () => {
      const value = await partialStateManager.getStorageView(
        DUMMY_CONTRACTS[0],
        DUMMY_SLOTS[0]
      )

      value.should.equal(NULL_BYTES32)
    })
  })

  describe('getStorage(...)', async () => {
    it('should return the value of a storage slot when it exists', async () => {
      await partialStateManager.insertVerifiedStorage(
        DUMMY_CONTRACTS[0],
        DUMMY_SLOTS[0],
        DUMMY_VALUES[0]
      )

      await partialStateManager.getStorage(DUMMY_CONTRACTS[0], DUMMY_SLOTS[0])
      const flagged = await partialStateManager.existsInvalidStateAccessFlag()

      flagged.should.equal(false)
    })

    it('should return null bytes and flag if the storage slot is not set', async () => {
      await partialStateManager.getStorage(DUMMY_CONTRACTS[0], DUMMY_SLOTS[0])
      const flagged = await partialStateManager.existsInvalidStateAccessFlag()

      flagged.should.equal(true)
    })

    it('should fail if not called by the execution manager', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await partialStateManager.getStorage(
          DUMMY_CONTRACTS[0],
          DUMMY_SLOTS[0],
          {
            from: await randomWallet.getAddress(),
          }
        )
      })
    })
  })

  describe('setOvmContractNonce(...)', async () => {
    it('should set the nonce for an address', async () => {
      const nonce = 1234
      await partialStateManager.setOvmContractNonce(DUMMY_CONTRACTS[0], nonce)

      const contract = await partialStateManager.updatedContracts(0)
      const counter = await partialStateManager.updatedContractsCounter()
      const touched = await partialStateManager.contractTouched(
        DUMMY_CONTRACTS[0]
      )
      const updatedNonce = await partialStateManager.ovmContractNonces(
        DUMMY_CONTRACTS[0]
      )

      contract.should.equal(DUMMY_CONTRACTS[0])
      counter.should.equal(1)
      touched.should.equal(true)
      updatedNonce.should.equal(nonce)
    })

    it('should not change the counter if the address has already been touched', async () => {
      const nonce = 1234
      await partialStateManager.setOvmContractNonce(DUMMY_CONTRACTS[0], nonce)

      const newNonce = 5678
      await partialStateManager.setOvmContractNonce(
        DUMMY_CONTRACTS[0],
        newNonce
      )

      const contract = await partialStateManager.updatedContracts(0)
      const counter = await partialStateManager.updatedContractsCounter()
      const touched = await partialStateManager.contractTouched(
        DUMMY_CONTRACTS[0]
      )
      const updatedNonce = await partialStateManager.ovmContractNonces(
        DUMMY_CONTRACTS[0]
      )

      contract.should.equal(DUMMY_CONTRACTS[0])
      counter.should.equal(1)
      touched.should.equal(true)
      updatedNonce.should.equal(newNonce)
    })

    it('should fail if not called by the execution manager', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        const nonce = 1234
        await partialStateManager.setOvmContractNonce(
          DUMMY_CONTRACTS[0],
          nonce,
          {
            from: await randomWallet.getAddress(),
          }
        )
      })
    })
  })

  describe('getOvmContractNonceView(...)', async () => {
    it('should get the nonce for a given address', async () => {
      const nonce = 1234
      await partialStateManager.setOvmContractNonce(DUMMY_CONTRACTS[0], nonce)

      const result = await partialStateManager.getOvmContractNonceView(
        DUMMY_CONTRACTS[0]
      )

      result.should.equal(nonce)
    })

    it('should return zero if the address has not been set', async () => {
      const result = await partialStateManager.getOvmContractNonceView(
        DUMMY_CONTRACTS[0]
      )

      result.should.equal(0)
    })
  })

  describe('getOvmContractNonce(...)', async () => {
    it('should get the nonce for a given address', async () => {
      const nonce = 1234
      await partialStateManager.insertVerifiedContract(
        DUMMY_CONTRACTS[0],
        DUMMY_CODE_CONTRACTS[0],
        nonce
      )

      await partialStateManager.getOvmContractNonce(DUMMY_CONTRACTS[0])
      const flagged = await partialStateManager.existsInvalidStateAccessFlag()

      flagged.should.equal(false)
    })

    it('should return zero and flag if the address has not been set', async () => {
      await partialStateManager.getOvmContractNonce(DUMMY_CONTRACTS[0])
      const flagged = await partialStateManager.existsInvalidStateAccessFlag()

      flagged.should.equal(true)
    })

    it('should fail if not called by the execution manager', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await partialStateManager.getOvmContractNonce(DUMMY_CONTRACTS[0], {
          from: await randomWallet.getAddress(),
        })
      })
    })
  })

  describe('incrementOvmContractNonce(...)', async () => {
    it('should increase the contract nonce by one', async () => {
      const nonce = 1234
      await partialStateManager.setOvmContractNonce(DUMMY_CONTRACTS[0], nonce)
      await partialStateManager.incrementOvmContractNonce(DUMMY_CONTRACTS[0])

      const contract = await partialStateManager.updatedContracts(0)
      const counter = await partialStateManager.updatedContractsCounter()
      const touched = await partialStateManager.contractTouched(
        DUMMY_CONTRACTS[0]
      )
      const result = await partialStateManager.ovmContractNonces(
        DUMMY_CONTRACTS[0]
      )

      contract.should.equal(DUMMY_CONTRACTS[0])
      counter.should.equal(1)
      touched.should.equal(true)
      result.should.equal(nonce + 1)
    })

    it('should not change the counter if the address has already been touched', async () => {
      const nonce = 1234
      await partialStateManager.setOvmContractNonce(DUMMY_CONTRACTS[0], nonce)
      await partialStateManager.incrementOvmContractNonce(DUMMY_CONTRACTS[0])
      await partialStateManager.incrementOvmContractNonce(DUMMY_CONTRACTS[0])

      const contract = await partialStateManager.updatedContracts(0)
      const counter = await partialStateManager.updatedContractsCounter()
      const touched = await partialStateManager.contractTouched(
        DUMMY_CONTRACTS[0]
      )
      const result = await partialStateManager.ovmContractNonces(
        DUMMY_CONTRACTS[0]
      )

      contract.should.equal(DUMMY_CONTRACTS[0])
      counter.should.equal(1)
      touched.should.equal(true)
      result.should.equal(nonce + 2)
    })

    it('should flag if not verified', async () => {
      await partialStateManager.incrementOvmContractNonce(DUMMY_CONTRACTS[0])

      const flagged = await partialStateManager.existsInvalidStateAccessFlag()

      flagged.should.equal(true)
    })

    it('should fail if not called by the execution manager', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await partialStateManager.incrementOvmContractNonce(
          DUMMY_CONTRACTS[0],
          {
            from: await randomWallet.getAddress(),
          }
        )
      })
    })
  })

  describe('associateCodeContract(...)', async () => {
    it('should set the code contract address for a given ovm contract', async () => {
      await partialStateManager.associateCodeContract(
        DUMMY_CONTRACTS[0],
        DUMMY_CODE_CONTRACTS[0]
      )

      const associated = await partialStateManager.ovmAddressToCodeContractAddress(
        DUMMY_CONTRACTS[0]
      )

      associated.should.equal(DUMMY_CODE_CONTRACTS[0])
    })

    it('should fail if not called by the execution manager', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await partialStateManager.associateCodeContract(
          DUMMY_CONTRACTS[0],
          DUMMY_CODE_CONTRACTS[0],
          {
            from: await randomWallet.getAddress(),
          }
        )
      })
    })
  })

  describe('registerCreatedContract(...)', async () => {
    it('should mark the contract as verified and set its nonce to zero', async () => {
      await partialStateManager.registerCreatedContract(DUMMY_CONTRACTS[0])

      const isVerifiedContract = await partialStateManager.isVerifiedContract(
        DUMMY_CONTRACTS[0]
      )
      const nonce = await partialStateManager.getOvmContractNonceView(
        DUMMY_CONTRACTS[0]
      )

      isVerifiedContract.should.equal(true)
      nonce.should.equal(0)
    })

    it('should fail if not called by the execution manager', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await partialStateManager.registerCreatedContract(DUMMY_CONTRACTS[0], {
          from: await randomWallet.getAddress(),
        })
      })
    })
  })

  describe('peekUpdatedStorageSlot()', async () => {
    it('should return the last storage slot on the queue', async () => {
      await partialStateManager.setStorage(
        DUMMY_CONTRACTS[0],
        DUMMY_SLOTS[0],
        DUMMY_VALUES[0]
      )

      const [
        updatedContract,
        updatedSlot,
        updatedValue,
      ] = await partialStateManager.peekUpdatedStorageSlot()

      updatedContract.should.equal(DUMMY_CONTRACTS[0])
      updatedSlot.should.equal(DUMMY_SLOTS[0])
      updatedValue.should.equal(DUMMY_VALUES[0])
    })

    it('should fail if there are no storage slots to be updated', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await partialStateManager.peekUpdatedStorageSlot()
      })
    })
  })

  describe('popUpdatedStorageSlot()', async () => {
    it('should return the last storage slot on the queue and remove it', async () => {
      await partialStateManager.setStorage(
        DUMMY_CONTRACTS[0],
        DUMMY_SLOTS[0],
        DUMMY_VALUES[0]
      )

      await partialStateManager.popUpdatedStorageSlot()
      const counter = await partialStateManager.updatedStorageSlotCounter()

      counter.should.equal(0)
    })

    it('should fail if there are no storage slots to be updated', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await partialStateManager.popUpdatedStorageSlot()
      })
    })

    it('should fail if not called by the state transitioner', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await partialStateManager.popUpdatedStorageSlot({
          from: await randomWallet.getAddress(),
        })
      })
    })
  })

  describe('peekUpdatedContract()', async () => {
    it('should return the last contract on the queue', async () => {
      await partialStateManager.registerCreatedContract(DUMMY_CONTRACTS[0])
      await partialStateManager.associateCodeContract(
        DUMMY_CONTRACTS[0],
        DUMMY_CODE_CONTRACTS[0]
      )

      const [
        updatedContract,
        updatedNonce,
        updatedCodeHash,
      ] = await partialStateManager.peekUpdatedContract()

      updatedContract.should.equal(DUMMY_CONTRACTS[0])
      updatedNonce.should.equal(0)
      updatedCodeHash.should.equal(NULL_BYTES32)
    })

    it('should fail if there are no contracts to be updated', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await partialStateManager.peekUpdatedContract()
      })
    })
  })

  describe('popUpdatedContract()', async () => {
    it('should return the last contract on the queue and remove it', async () => {
      await partialStateManager.registerCreatedContract(DUMMY_CONTRACTS[0])
      await partialStateManager.associateCodeContract(
        DUMMY_CONTRACTS[0],
        DUMMY_CODE_CONTRACTS[0]
      )

      await partialStateManager.popUpdatedContract()
      const counter = await partialStateManager.updatedContractsCounter()

      counter.should.equal(0)
    })

    it('should fail if there are no contracts to be updated', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await partialStateManager.popUpdatedContract()
      })
    })

    it('should fail if not called by the state transitioner', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await partialStateManager.popUpdatedContract({
          from: await randomWallet.getAddress(),
        })
      })
    })
  })

  describe('getCodeContractAddressView(...)', async () => {
    it('should return the code contract for a given ovm contract', async () => {
      await partialStateManager.associateCodeContract(
        DUMMY_CONTRACTS[0],
        DUMMY_CODE_CONTRACTS[0]
      )

      const codeAddress = await partialStateManager.getCodeContractAddressView(
        DUMMY_CONTRACTS[0]
      )

      codeAddress.should.equal(DUMMY_CODE_CONTRACTS[0])
    })

    it('should return null bytes if the contract is not associated', async () => {
      const codeAddress = await partialStateManager.getCodeContractAddressView(
        DUMMY_CONTRACTS[0]
      )

      codeAddress.should.equal(NULL_ADDRESS)
    })
  })

  describe('getCodeContractAddressFromOvmAddress(...)', async () => {
    it('should return the code contract address when it exists', async () => {
      await partialStateManager.insertVerifiedContract(
        DUMMY_CONTRACTS[0],
        DUMMY_CODE_CONTRACTS[0],
        0
      )

      await partialStateManager.getCodeContractAddressFromOvmAddress(
        DUMMY_CONTRACTS[0]
      )
      const flagged = await partialStateManager.existsInvalidStateAccessFlag()

      flagged.should.equal(false)
    })

    it('should return null bytes and flag when the contract does not exist', async () => {
      await partialStateManager.getCodeContractAddressFromOvmAddress(
        DUMMY_CONTRACTS[0]
      )
      const flagged = await partialStateManager.existsInvalidStateAccessFlag()

      flagged.should.equal(true)
    })

    it('should fail if not called by the execution manager', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await partialStateManager.getCodeContractAddressFromOvmAddress(
          DUMMY_CONTRACTS[0],
          {
            from: await randomWallet.getAddress(),
          }
        )
      })
    })
  })

  describe('getCodeContractBytecode(...)', async () => {
    it('should get the bytecode of a contract at the given address', async () => {
      const bytecode = await partialStateManager.getCodeContractBytecode(
        DUMMY_CONTRACTS[0]
      )

      bytecode.should.equal('0x')
    })
  })

  describe('getCodeContractHash(...)', async () => {
    it('should get the hash of the bytecode at the given address', async () => {
      const codehash = await partialStateManager.getCodeContractHash(
        DUMMY_CONTRACTS[0]
      )

      codehash.should.equal(RLP_NULL_HASH)
    })
  })
})
