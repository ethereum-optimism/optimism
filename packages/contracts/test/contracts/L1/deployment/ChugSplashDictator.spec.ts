import { ethers } from 'hardhat'
import { Contract } from 'ethers'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'

import { expect } from '../../../setup'
import { deploy } from '../../../helpers'

describe('ChugSplashDictator', () => {
  let signer1: SignerWithAddress
  let signer2: SignerWithAddress
  before(async () => {
    ;[signer1, signer2] = await ethers.getSigners()
  })

  let L1ChugSplashProxy: Contract
  let ChugSplashDictator: Contract
  beforeEach(async () => {
    L1ChugSplashProxy = await deploy('L1ChugSplashProxy', {
      signer: signer1,
      args: [signer1.address],
    })

    ChugSplashDictator = await deploy('ChugSplashDictator', {
      signer: signer1,
      args: [
        L1ChugSplashProxy.address,
        signer1.address,
        ethers.utils.keccak256('0x1111'),
        ethers.utils.keccak256('0x1234'),
        ethers.utils.keccak256('0x5678'),
        ethers.utils.keccak256('0x1234'),
        ethers.utils.keccak256('0x1234'),
      ],
    })

    await L1ChugSplashProxy.setOwner(ChugSplashDictator.address)
  })

  describe('doActions', () => {
    it('should revert when sent wrong code', async () => {
      await expect(ChugSplashDictator.doActions('0x2222')).to.be.revertedWith(
        'ChugSplashDictator: Incorrect code hash.'
      )
    })

    it('should set the proxy code, storage & owner', async () => {
      await expect(ChugSplashDictator.doActions('0x1111')).to.not.be.reverted
    })
  })

  describe('returnOwnership', () => {
    it('should transfer contractc ownership to finalOwner', async () => {
      await expect(ChugSplashDictator.returnOwnership()).to.not.be.reverted
    })

    it('should revert when called by non-owner', async () => {
      await expect(
        ChugSplashDictator.connect(signer2).returnOwnership()
      ).to.be.revertedWith('ChugSplashDictator: only callable by finalOwner')
    })
  })
})
