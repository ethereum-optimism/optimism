import { expect } from '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { ContractFactory, Contract, Signer } from 'ethers'

describe('MockCrossDomainMessageReceiver', () => {
  let wallet: Signer
  let wallet2: Signer
  before(async () => {
    ;[wallet, wallet2] = await ethers.getSigners()
  })

  let SimpleStorageMessageReceiverFactory: ContractFactory
  before(async () => {
    SimpleStorageMessageReceiverFactory = await ethers.getContractFactory('SimpleStorageMessageReceiver')
  })

  let L2SimpleStorageMessageReceiver: Contract
  beforeEach(async () => {
    L2SimpleStorageMessageReceiver = await SimpleStorageMessageReceiverFactory.deploy()
    
    await L2SimpleStorageMessageReceiver.setMessenger(await wallet.getAddress())
  })

  describe('receiveMessage', () => {
    it('should successfully receive a message from the messenger', async () => {
      const expectedStorageKey = ethers.utils.keccak256('0x1234')
      const expectedStorageValue = ethers.utils.keccak256('0x5678')

      const calldata = L2SimpleStorageMessageReceiver.interface.encodeFunctionData(
        'setStorage',
        [
          expectedStorageKey,
          expectedStorageValue
        ]
      )

      const expectedMessage = [
        await wallet.getAddress(),
        calldata,
        ethers.BigNumber.from(Date.now()),
        ethers.BigNumber.from(123)
      ]

      await L2SimpleStorageMessageReceiver.receiveMessage(
        ...expectedMessage,
        {
          from: await wallet.getAddress()
        }
      )

      const actualStorageValue = await L2SimpleStorageMessageReceiver.getStorage(expectedStorageKey)
      expect(actualStorageValue).to.equal(expectedStorageValue)

      const actualMessage = await L2SimpleStorageMessageReceiver.messages(0)
      expect(actualMessage).to.deep.equal(expectedMessage)
    })

    it('should fail if the message is not received from the messenger', async () => {
      const expectedStorageKey = ethers.utils.keccak256('0x1234')
      const expectedStorageValue = ethers.utils.keccak256('0x5678')

      const calldata = L2SimpleStorageMessageReceiver.interface.encodeFunctionData(
        'setStorage',
        [
          expectedStorageKey,
          expectedStorageValue
        ]
      )

      const expectedMessage = [
        await wallet.getAddress(),
        calldata,
        ethers.BigNumber.from(Date.now()),
        ethers.BigNumber.from(123)
      ]

      L2SimpleStorageMessageReceiver = L2SimpleStorageMessageReceiver.connect(wallet2)
      await expect(L2SimpleStorageMessageReceiver.receiveMessage(
        ...expectedMessage
      )).to.be.revertedWith('Only the CrossDomainMessenger can call this function.')
    })
  })
})