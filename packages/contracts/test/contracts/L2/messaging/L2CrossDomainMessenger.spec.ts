import { ethers } from 'hardhat'
import { Contract } from 'ethers'
import { applyL1ToL2Alias } from '@eth-optimism/core-utils'
import { smock, FakeContract, MockContract } from '@defi-wonderland/smock'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'

import { expect } from '../../../setup'
import { predeploys } from '../../../../src'
import {
  impersonate,
  deploy,
  NON_NULL_BYTES32,
  NON_ZERO_ADDRESS,
  encodeXDomainCalldata,
} from '../../../helpers'

describe('L2CrossDomainMessenger', () => {
  let signer: SignerWithAddress
  before(async () => {
    ;[signer] = await ethers.getSigners()
  })

  let Fake__TargetContract: FakeContract
  let Fake__L1CrossDomainMessenger: FakeContract
  let Fake__OVM_L2ToL1MessagePasser: FakeContract
  before(async () => {
    Fake__TargetContract = await smock.fake<Contract>('Helper_SimpleProxy')
    Fake__L1CrossDomainMessenger = await smock.fake<Contract>(
      'L1CrossDomainMessenger'
    )
    Fake__OVM_L2ToL1MessagePasser = await smock.fake<Contract>(
      'OVM_L2ToL1MessagePasser',
      { address: predeploys.OVM_L2ToL1MessagePasser }
    )
  })

  let impersonatedL1CrossDomainMessengerSender: SignerWithAddress
  before(async () => {
    impersonatedL1CrossDomainMessengerSender = await impersonate(
      applyL1ToL2Alias(Fake__L1CrossDomainMessenger.address),
      '0xFFFFFFFFFFFFFFFFF'
    )
  })

  let L2CrossDomainMessenger: Contract
  beforeEach(async () => {
    L2CrossDomainMessenger = await deploy('L2CrossDomainMessenger', {
      args: [Fake__L1CrossDomainMessenger.address],
    })
  })

  describe('xDomainMessageSender', () => {
    let Mock__L2CrossDomainMessenger: MockContract<Contract>
    before(async () => {
      Mock__L2CrossDomainMessenger = await (
        await smock.mock('L2CrossDomainMessenger')
      ).deploy(Fake__L1CrossDomainMessenger.address)
    })

    it('should return the xDomainMsgSender address', async () => {
      await Mock__L2CrossDomainMessenger.setVariable(
        'xDomainMsgSender',
        '0x0000000000000000000000000000000000000000'
      )

      expect(
        await Mock__L2CrossDomainMessenger.xDomainMessageSender()
      ).to.equal('0x0000000000000000000000000000000000000000')
    })
  })

  describe('sendMessage', () => {
    it('should be able to send a single message', async () => {
      await expect(
        L2CrossDomainMessenger.sendMessage(
          NON_ZERO_ADDRESS,
          NON_NULL_BYTES32,
          100_000
        )
      ).to.not.be.reverted

      expect(
        Fake__OVM_L2ToL1MessagePasser.passMessageToL1.getCall(0).args[0]
      ).to.deep.equal(
        encodeXDomainCalldata(
          NON_ZERO_ADDRESS,
          await signer.getAddress(),
          NON_NULL_BYTES32,
          0
        )
      )
    })

    it('should be able to send the same message twice', async () => {
      await L2CrossDomainMessenger.sendMessage(
        NON_ZERO_ADDRESS,
        NON_NULL_BYTES32,
        100_000
      )

      await expect(
        L2CrossDomainMessenger.sendMessage(
          NON_ZERO_ADDRESS,
          NON_NULL_BYTES32,
          100_000
        )
      ).to.not.be.reverted
    })
  })

  describe('relayMessage', () => {
    it('should revert if the L1 message sender is not the L1CrossDomainMessenger', async () => {
      await expect(
        L2CrossDomainMessenger.connect(signer).relayMessage(
          Fake__TargetContract.address,
          signer.address,
          Fake__TargetContract.interface.encodeFunctionData('setTarget', [
            NON_ZERO_ADDRESS,
          ]),
          0
        )
      ).to.be.revertedWith('Provided message could not be verified.')
    })

    it('should send a call to the target contract', async () => {
      await L2CrossDomainMessenger.connect(
        impersonatedL1CrossDomainMessengerSender
      ).relayMessage(
        Fake__TargetContract.address,
        signer.address,
        Fake__TargetContract.interface.encodeFunctionData('setTarget', [
          NON_ZERO_ADDRESS,
        ]),
        0
      )

      expect(Fake__TargetContract.setTarget.getCall(0).args[0]).to.deep.equal(
        NON_ZERO_ADDRESS
      )
    })

    it('the xDomainMessageSender is reset to the original value', async () => {
      await expect(
        L2CrossDomainMessenger.xDomainMessageSender()
      ).to.be.revertedWith('xDomainMessageSender is not set')

      await L2CrossDomainMessenger.connect(
        impersonatedL1CrossDomainMessengerSender
      ).relayMessage(
        Fake__TargetContract.address,
        signer.address,
        Fake__TargetContract.interface.encodeFunctionData('setTarget', [
          NON_ZERO_ADDRESS,
        ]),
        0
      )

      await expect(
        L2CrossDomainMessenger.xDomainMessageSender()
      ).to.be.revertedWith('xDomainMessageSender is not set')
    })

    it('should revert if trying to send the same message twice', async () => {
      await L2CrossDomainMessenger.connect(
        impersonatedL1CrossDomainMessengerSender
      ).relayMessage(
        Fake__TargetContract.address,
        signer.address,
        Fake__TargetContract.interface.encodeFunctionData('setTarget', [
          NON_ZERO_ADDRESS,
        ]),
        0
      )

      await expect(
        L2CrossDomainMessenger.connect(
          impersonatedL1CrossDomainMessengerSender
        ).relayMessage(
          Fake__TargetContract.address,
          signer.address,
          Fake__TargetContract.interface.encodeFunctionData('setTarget', [
            NON_ZERO_ADDRESS,
          ]),
          0
        )
      ).to.be.revertedWith('Provided message has already been received.')
    })

    it('should not make a call if the target is the L2 MessagePasser', async () => {
      const resProm = L2CrossDomainMessenger.connect(
        impersonatedL1CrossDomainMessengerSender
      ).relayMessage(
        predeploys.OVM_L2ToL1MessagePasser,
        signer.address,
        Fake__OVM_L2ToL1MessagePasser.interface.encodeFunctionData(
          'passMessageToL1(bytes)',
          [NON_NULL_BYTES32]
        ),
        0
      )

      // The call to relayMessage() should succeed.
      await expect(resProm).to.not.be.reverted

      // There should be no 'relayedMessage' event logged in the receipt.
      expect(
        (
          await Fake__OVM_L2ToL1MessagePasser.provider.getTransactionReceipt(
            (
              await resProm
            ).hash
          )
        ).logs
      ).to.deep.equal([])

      // The message should be registered as successful.
      expect(
        await L2CrossDomainMessenger.successfulMessages(
          ethers.utils.solidityKeccak256(
            ['bytes'],
            [
              encodeXDomainCalldata(
                predeploys.OVM_L2ToL1MessagePasser,
                signer.address,
                Fake__OVM_L2ToL1MessagePasser.interface.encodeFunctionData(
                  'passMessageToL1(bytes)',
                  [NON_NULL_BYTES32]
                ),
                0
              ),
            ]
          )
        )
      ).to.be.true
    })
  })
})
