import { expect } from '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { ContractFactory, Signer, Contract } from 'ethers'

const getXDomainCalldata = (
  messenger: Contract,
  target: string,
  sender: string,
  message: string,
  nonce: number
): string => {
  return messenger.interface.encodeFunctionData('relayMessage', [
    target,
    sender,
    message,
    nonce,
  ])
}

describe('L2CrossDomainMessenger', () => {
  let wallet: Signer
  let wallet2: Signer
  before(async () => {
    ;[wallet, wallet2] = await ethers.getSigners()
  })

  let MockL1MessageSenderFactory: ContractFactory
  let MockL2ToL1MessagePasserFactory: ContractFactory
  let L2CrossDomainMessengerFactory: ContractFactory
  let CrossDomainSimpleStorageFactory: ContractFactory
  before(async () => {
    MockL1MessageSenderFactory = await ethers.getContractFactory(
      'MockL1MessageSender'
    )
    MockL2ToL1MessagePasserFactory = await ethers.getContractFactory(
      'MockL2ToL1MessagePasser'
    )
    L2CrossDomainMessengerFactory = await ethers.getContractFactory(
      'L2CrossDomainMessenger'
    )
    CrossDomainSimpleStorageFactory = await ethers.getContractFactory(
      'CrossDomainSimpleStorage'
    )
  })

  let MockL1MessageSender: Contract
  let MockL2ToL1MessagePasser: Contract
  let CrossDomainSimpleStorage: Contract
  let L2CrossDomainMessenger: Contract
  beforeEach(async () => {
    MockL1MessageSender = await MockL1MessageSenderFactory.deploy()
    MockL2ToL1MessagePasser = await MockL2ToL1MessagePasserFactory.deploy()
    CrossDomainSimpleStorage = await CrossDomainSimpleStorageFactory.deploy()
    L2CrossDomainMessenger = await L2CrossDomainMessengerFactory.deploy(
      MockL1MessageSender.address,
      MockL2ToL1MessagePasser.address
    )

    await CrossDomainSimpleStorage.setMessenger(L2CrossDomainMessenger.address)
    await L2CrossDomainMessenger.setTargetMessengerAddress(
      await wallet.getAddress()
    )
  })

  describe('relayMessage()', () => {
    it('should relay a message to a target contract', async () => {
      const expectedKey = ethers.utils.keccak256('0x1234')
      const expectedVal = ethers.utils.keccak256('0x5678')

      const calldata = CrossDomainSimpleStorage.interface.encodeFunctionData(
        'crossDomainSetStorage',
        [expectedKey, expectedVal]
      )

      await MockL1MessageSender.setL1MessageSender(await wallet.getAddress())

      await L2CrossDomainMessenger.relayMessage(
        CrossDomainSimpleStorage.address,
        await wallet.getAddress(),
        calldata,
        0
      )

      const actualVal = await CrossDomainSimpleStorage.getStorage(expectedKey)
      expect(actualVal).to.equal(expectedVal)
    })

    it('should fail if attempting to relay a message twice', async () => {
      const expectedKey = ethers.utils.keccak256('0x1234')
      const expectedVal = ethers.utils.keccak256('0x5678')

      const calldata = CrossDomainSimpleStorage.interface.encodeFunctionData(
        'crossDomainSetStorage',
        [expectedKey, expectedVal]
      )

      await MockL1MessageSender.setL1MessageSender(await wallet.getAddress())

      await L2CrossDomainMessenger.relayMessage(
        CrossDomainSimpleStorage.address,
        await wallet.getAddress(),
        calldata,
        0
      )

      await expect(
        L2CrossDomainMessenger.relayMessage(
          CrossDomainSimpleStorage.address,
          await wallet.getAddress(),
          calldata,
          0
        )
      ).to.be.rejectedWith('Provided message has already been received.')
    })

    it('should fail if the sender is not the L1 messenger', async () => {
      const expectedKey = ethers.utils.keccak256('0x1234')
      const expectedVal = ethers.utils.keccak256('0x5678')

      const calldata = CrossDomainSimpleStorage.interface.encodeFunctionData(
        'crossDomainSetStorage',
        [expectedKey, expectedVal]
      )

      await MockL1MessageSender.setL1MessageSender(await wallet2.getAddress())

      await expect(
        L2CrossDomainMessenger.relayMessage(
          CrossDomainSimpleStorage.address,
          await wallet.getAddress(),
          calldata,
          0
        )
      ).to.be.rejectedWith('Provided message could not be verified.')
    })
  })

  describe('sendMessage()', () => {
    it('should send a message to the L2ToL1MessagePasser', async () => {
      const expectedKey = ethers.utils.keccak256('0x1234')
      const expectedVal = ethers.utils.keccak256('0x5678')

      const calldata = CrossDomainSimpleStorage.interface.encodeFunctionData(
        'crossDomainSetStorage',
        [expectedKey, expectedVal]
      )

      const messageNonce = await L2CrossDomainMessenger.messageNonce()

      await L2CrossDomainMessenger.sendMessage(
        CrossDomainSimpleStorage.address,
        calldata
      )

      const xDomainCalldata = getXDomainCalldata(
        L2CrossDomainMessenger,
        CrossDomainSimpleStorage.address,
        await wallet.getAddress(),
        calldata,
        messageNonce
      )

      const messageHash = ethers.utils.keccak256(xDomainCalldata)
      const messageStored = await MockL2ToL1MessagePasser.storedMessages(
        messageHash
      )
      expect(messageStored).to.equal(true)

      const messageSent = await L2CrossDomainMessenger.sentMessages(messageHash)
      expect(messageSent).to.equal(true)

      const newMessageNonce = await L2CrossDomainMessenger.messageNonce()
      expect(newMessageNonce.toNumber()).to.equal(1)
    })
  })
})
