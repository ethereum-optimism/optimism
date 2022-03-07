/* External Imports */
import { ethers } from 'hardhat'
import { Contract, Signer, ContractFactory } from 'ethers'

/* Internal Imports */
import { expect } from '../../../setup'

describe('ChugSplashDictator', () => {
  let signer: Signer
  let otherSigner: Signer
  let signerAddress: string

  let Factory__L1ChugSplashProxy: ContractFactory
  let Factory__ChugSplashDictator: ContractFactory
  before(async () => {
    ;[signer, otherSigner] = await ethers.getSigners()

    Factory__L1ChugSplashProxy = await ethers.getContractFactory(
      'L1ChugSplashProxy'
    )

    Factory__ChugSplashDictator = await ethers.getContractFactory(
      'ChugSplashDictator'
    )

    signerAddress = await signer.getAddress()
  })

  let L1ChugSplashProxy: Contract
  let ChugSplashDictator: Contract
  beforeEach(async () => {
    L1ChugSplashProxy = await Factory__L1ChugSplashProxy.connect(signer).deploy(
      signerAddress
    )

    ChugSplashDictator = await Factory__ChugSplashDictator.connect(
      signer
    ).deploy(
      L1ChugSplashProxy.address,
      signerAddress,
      ethers.utils.keccak256('0x1111'),
      ethers.utils.keccak256('0x1234'),
      ethers.utils.keccak256('0x5678'),
      ethers.utils.keccak256('0x1234'),
      ethers.utils.keccak256('0x1234')
    )

    await L1ChugSplashProxy.connect(signer).setOwner(ChugSplashDictator.address)
  })

  describe('doActions', () => {
    it('should revert when sent wrong code', async () => {
      await expect(ChugSplashDictator.doActions('0x2222')).to.be.revertedWith(
        'ChugSplashDictator: Incorrect code hash.'
      )
    })

    it('should set the proxy code, storage & owner', async () => {
      await expect(ChugSplashDictator.connect(signer).doActions('0x1111')).to
        .not.be.reverted
    })
  })

  describe('returnOwnership', () => {
    it('should transfer contractc ownership to finalOwner', async () => {
      await expect(ChugSplashDictator.connect(signer).returnOwnership()).to.not
        .be.reverted
    })

    it('should revert when called by non-owner', async () => {
      await expect(
        ChugSplashDictator.connect(otherSigner).returnOwnership()
      ).to.be.revertedWith('ChugSplashDictator: only callable by finalOwner')
    })
  })
})
