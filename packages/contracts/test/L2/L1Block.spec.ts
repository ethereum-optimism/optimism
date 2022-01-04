import { expect } from 'chai'
import { ethers } from 'hardhat'
import { Signer } from 'ethers'

import { L1Block__factory, L1Block } from '../../typechain'

const DEPOSITOR_ACCOUNT = '0xdeaddeaddeaddeaddeaddeaddeaddeaddead0001'
const NON_ZERO_HASH = '0x' + 'ab'.repeat(32)

describe('L1Block contract', () => {
  let signer: Signer
  let signerAddress: string
  let l1Block: L1Block
  let depositor: Signer
  before(async () => {
    ;[signer] = await ethers.getSigners()
    signerAddress = await signer.getAddress()
    l1Block = await new L1Block__factory(signer).deploy()
    await l1Block.deployed()

    depositor = await ethers.getSigner(DEPOSITOR_ACCOUNT)
  })

  it('setL1BlockValues: Should revert if not called by L1 Attributes Depositor Account', async () => {
    await expect(
      l1Block.connect(signer).setL1BlockValues(1, 2, 3, NON_ZERO_HASH)
    ).to.be.revertedWith('OnlyDepositor()')
  })

  describe('Should return the correct block values for:', async () => {
    before(async () => {
      await ethers.provider.send('hardhat_impersonateAccount', [
        DEPOSITOR_ACCOUNT,
      ])
      await ethers.provider.send('hardhat_setBalance', [
        DEPOSITOR_ACCOUNT,
        '0xFFFFFFFFFFFF',
      ])
      await l1Block.connect(depositor).setL1BlockValues(1, 2, 3, NON_ZERO_HASH)
      await ethers.provider.send('hardhat_stopImpersonatingAccount', [
        DEPOSITOR_ACCOUNT,
      ])
      l1Block.connect(signer)
    })

    it('number', async () => {
      expect(await l1Block.number()).to.equal(1)
    })

    it('timestamp', async () => {
      expect(await l1Block.timestamp()).to.equal(2)
    })

    it('basefee', async () => {
      expect(await l1Block.basefee()).to.equal(3)
    })

    it('hash', async () => {
      expect(await l1Block.hash()).to.equal(NON_ZERO_HASH)
    })
  })
})
