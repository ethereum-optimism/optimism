import { expect } from '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { ContractFactory, Contract, Signer } from 'ethers'
import { NULL_ADDRESS } from '@eth-optimism/core-utils'

describe('MockCrossDomainMessenger', () => {
  let wallet: Signer
  before(async () => {
    ;[wallet] = await ethers.getSigners()
  })

  let MockCrossDomainMessengerFactory: ContractFactory
  let CrossDomainSimpleStorageFactory: ContractFactory
  before(async () => {
    MockCrossDomainMessengerFactory = await ethers.getContractFactory(
      'MockCrossDomainMessenger'
    )
    CrossDomainSimpleStorageFactory = await ethers.getContractFactory(
      'CrossDomainSimpleStorage'
    )
  })

  let L1MockCrossDomainMessenger: Contract
  let L2MockCrossDomainMessenger: Contract
  beforeEach(async () => {
    L1MockCrossDomainMessenger = await MockCrossDomainMessengerFactory.deploy()
    L2MockCrossDomainMessenger = await MockCrossDomainMessengerFactory.deploy()

    await L1MockCrossDomainMessenger.setTargetMessenger(
      L2MockCrossDomainMessenger.address
    )
    await L2MockCrossDomainMessenger.setTargetMessenger(
      L1MockCrossDomainMessenger.address
    )
  })

  let L2SimpleStorage: Contract
  beforeEach(async () => {
    L2SimpleStorage = await CrossDomainSimpleStorageFactory.deploy()

    await L2SimpleStorage.setMessenger(L2MockCrossDomainMessenger.address)
  })
  describe('relayMessage', () => {
    it('should successfully relay a message to the target receiver', async () => {
      const expectedStorageKey = ethers.utils.keccak256('0x1234')
      const expectedStorageValue = ethers.utils.keccak256('0x5678')

      const calldata = L2SimpleStorage.interface.encodeFunctionData(
        'crossDomainSetStorage',
        [expectedStorageKey, expectedStorageValue]
      )

      const expectedMessage = [await wallet.getAddress(), calldata]

      await L2MockCrossDomainMessenger.relayMessage(
        L2SimpleStorage.address,
        ...expectedMessage
      )

      const actualStorageValue = await L2SimpleStorage.getStorage(
        expectedStorageKey
      )
      expect(actualStorageValue).to.equal(expectedStorageValue)
      expect(await L2SimpleStorage.crossDomainMsgSender()).to.equal(
        await wallet.getAddress()
      )
    })
  })

  describe('sendMessage', () => {
    it('should successfully send a message to another messenger', async () => {
      const expectedStorageKey = ethers.utils.keccak256('0x1234')
      const expectedStorageValue = ethers.utils.keccak256('0x5678')

      const calldata = L2SimpleStorage.interface.encodeFunctionData(
        'crossDomainSetStorage',
        [expectedStorageKey, expectedStorageValue]
      )

      await L1MockCrossDomainMessenger.sendMessage(
        L2SimpleStorage.address,
        calldata,
        {
          from: await wallet.getAddress(),
        }
      )

      const currentBlock = await ethers.provider.getBlock('latest')
      const expectedMessage = [await wallet.getAddress(), calldata]

      const actualStorageValue = await L2SimpleStorage.getStorage(
        expectedStorageKey
      )
      expect(actualStorageValue).to.equal(expectedStorageValue)
      expect(await L2SimpleStorage.crossDomainMsgSender()).to.equal(
        await wallet.getAddress()
      )
    })

    it('should revert if its target messenger is not set', async () => {
      const expectedStorageKey = ethers.utils.keccak256('0x1234')
      const expectedStorageValue = ethers.utils.keccak256('0x5678')

      const calldata = L2SimpleStorage.interface.encodeFunctionData(
        'setStorage',
        [expectedStorageKey, expectedStorageValue]
      )

      await L1MockCrossDomainMessenger.setTargetMessenger(NULL_ADDRESS)

      await expect(
        L1MockCrossDomainMessenger.sendMessage(
          L2SimpleStorage.address,
          calldata,
          {
            from: await wallet.getAddress(),
          }
        )
      ).to.be.revertedWith(
        'Cannot send a message without setting the target messenger.'
      )
    })
  })
})
