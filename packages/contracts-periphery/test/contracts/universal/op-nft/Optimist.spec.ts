import hre from 'hardhat'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'
import { Contract, ethers } from 'ethers'

import { deploy } from '../../../helpers'
import { expect } from '../../../setup'

describe('Optimist', () => {
  let Optimist: Contract
  let AttestationStation: Contract

  let signer1: SignerWithAddress
  let signer2: SignerWithAddress
  let signer3: SignerWithAddress

  before('signer setup', async () => {
    ;[signer1, signer2, signer3] = await hre.ethers.getSigners()
  })

  beforeEach('deploy contracts', async () => {
    Optimist = await deploy('Optimist')
    AttestationStation = await deploy('AttestationStation')
    AttestationStation.connect(signer1).attest([
      {
        about: Optimist.address,
        key: ethers.utils.keccak256(ethers.utils.toUtf8Bytes('opnft.optimistNftBaseURI')),
        val: ethers.utils.toUtf8Bytes('https://optimism.io/optimist/'),
      },
    ])
  })

  // init
  it('should initialize', async () => {
    await Optimist.initialize(
      'Optimist',
      'OPT',
      signer1.address,
      AttestationStation.address
    )
    expect(await Optimist.name()).to.equal('Optimist')
    expect(await Optimist.symbol()).to.equal('OPT')
    expect(await Optimist.owner()).to.equal(signer1.address)
    expect(await Optimist.attestationStation()).to.equal(AttestationStation.address)
  })

  // mint
  it('should mint', async () => {
    await Optimist.initialize(
      'Optimist',
      'OPT',
      signer1.address,
      AttestationStation.address
    )
    await Optimist.connect(signer2).mint(signer2.address)
    expect(await Optimist.balanceOf(signer2.address)).to.equal(1)
    // prevent minting more than 1
    await expect(
      Optimist.connect(signer2).mint(signer2.address)
    ).to.be.revertedWith('Optimist::mint: ALREADY_MINTED')
  })

  // expect revert on transfer
  it('should not transfer', async () => {
    await Optimist.initialize(
      'Optimist',
      'OPT',
      signer1.address,
      AttestationStation.address
    )
    await Optimist.connect(signer2).mint(signer2.address)
    await expect(
      Optimist.connect(signer2).transferFrom(
        signer2.address,
        signer3.address,
        signer2.address.padStart(24, '0')
      )
    ).to.be.revertedWith('Optimist::_beforeTokenTransfer: SOUL_BOUND')
  })

  // baseURI and tokenURI should exist and be correct
  it('should have the correct baseURI with a tokenURI', async () => {
    await Optimist.initialize(
      'Optimist',
      'OPT',
      signer1.address,
      AttestationStation.address
    )
    await Optimist.connect(signer2).mint(signer2.address)
    expect(
      await Optimist
        .connect(signer2)
        .tokenURI(signer2.address.padStart(24, '0a')))
      .contains('https://optimism.io/optimist/')
  })

})
