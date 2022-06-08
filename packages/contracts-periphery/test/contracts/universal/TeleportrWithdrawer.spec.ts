import hre from 'hardhat'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'
import { Contract } from 'ethers'
import { toRpcHexString } from '@eth-optimism/core-utils'

import { expect } from '../../setup'
import { deploy } from '../../helpers'

describe('TeleportrWithdrawer', () => {
  let signer1: SignerWithAddress
  let signer2: SignerWithAddress
  before('signer setup', async () => {
    ;[signer1, signer2] = await hre.ethers.getSigners()
  })

  let SimpleStorage: Contract
  let MockTeleportr: Contract
  let TeleportrWithdrawer: Contract
  beforeEach('deploy contracts', async () => {
    SimpleStorage = await deploy('SimpleStorage')
    MockTeleportr = await deploy('MockTeleportr')
    TeleportrWithdrawer = await deploy('TeleportrWithdrawer', {
      signer: signer1,
      args: [signer1.address],
    })
  })

  describe('setRecipient', () => {
    describe('when called by authorized address', () => {
      it('should set the recipient', async () => {
        await TeleportrWithdrawer.setRecipient(signer1.address)
        expect(await TeleportrWithdrawer.recipient()).to.equal(signer1.address)
      })
    })

    describe('when called by not authorized address', () => {
      it('should revert', async () => {
        await expect(
          TeleportrWithdrawer.connect(signer2).setRecipient(signer2.address)
        ).to.be.revertedWith('UNAUTHORIZED')
      })
    })
  })

  describe('setTeleportr', () => {
    describe('when called by authorized address', () => {
      it('should set the recipient', async () => {
        await TeleportrWithdrawer.setTeleportr(MockTeleportr.address)
        expect(await TeleportrWithdrawer.teleportr()).to.equal(
          MockTeleportr.address
        )
      })
    })

    describe('when called by not authorized address', () => {
      it('should revert', async () => {
        await expect(
          TeleportrWithdrawer.connect(signer2).setTeleportr(signer2.address)
        ).to.be.revertedWith('UNAUTHORIZED')
      })
    })
  })

  describe('setData', () => {
    const data = `0x${'ff'.repeat(64)}`

    describe('when called by authorized address', () => {
      it('should set the data', async () => {
        await TeleportrWithdrawer.setData(data)
        expect(await TeleportrWithdrawer.data()).to.equal(data)
      })
    })

    describe('when called by not authorized address', () => {
      it('should revert', async () => {
        await expect(
          TeleportrWithdrawer.connect(signer2).setData(data)
        ).to.be.revertedWith('UNAUTHORIZED')
      })
    })
  })

  describe('withdrawTeleportrBalance', () => {
    const recipient = `0x${'11'.repeat(20)}`
    const amount = hre.ethers.constants.WeiPerEther
    beforeEach(async () => {
      await hre.ethers.provider.send('hardhat_setBalance', [
        MockTeleportr.address,
        toRpcHexString(amount),
      ])
      await TeleportrWithdrawer.setRecipient(recipient)
      await TeleportrWithdrawer.setTeleportr(MockTeleportr.address)
    })

    describe('when target is an EOA', () => {
      it('should withdraw the balance', async () => {
        await TeleportrWithdrawer.withdrawFromTeleportr()
        expect(await hre.ethers.provider.getBalance(recipient)).to.equal(amount)
      })
    })

    describe('when target is a contract', () => {
      it('should withdraw the balance and trigger code', async () => {
        const key = `0x${'dd'.repeat(32)}`
        const val = `0x${'ee'.repeat(32)}`
        await TeleportrWithdrawer.setRecipient(SimpleStorage.address)
        await TeleportrWithdrawer.setData(
          SimpleStorage.interface.encodeFunctionData('set', [key, val])
        )

        await TeleportrWithdrawer.withdrawFromTeleportr()

        expect(
          await hre.ethers.provider.getBalance(SimpleStorage.address)
        ).to.equal(amount)
        expect(await SimpleStorage.get(key)).to.equal(val)
      })
    })
  })
})
