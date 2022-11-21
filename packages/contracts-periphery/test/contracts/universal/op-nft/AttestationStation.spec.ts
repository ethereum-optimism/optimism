import hre from 'hardhat'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'
import { Contract, ethers } from 'ethers'

import { deploy } from '../../../helpers'
import { expect } from '../../../setup'

describe('AttestationStation', () => {
  let AttestationStation: Contract

  let signer1: SignerWithAddress
  let signer2: SignerWithAddress

  before('signer setup', async () => {
    ;[signer1, signer2] = await hre.ethers.getSigners()
  })

  beforeEach('deploy contracts', async () => {
    AttestationStation = await deploy('AttestationStation')
  })

  it('should attest some data', async () => {
    AttestationStation.connect(signer1).attest([
      {
        about: signer2.address,
        key: ethers.utils.keccak256(ethers.utils.toUtf8Bytes('test')),
        val: '0x1234',
      },
    ])
  })

  it('should emit an event on attestation', async () => {
    await expect(
      AttestationStation.connect(signer2).attest([
        {
          about: signer2.address,
          key: ethers.utils.keccak256(ethers.utils.toUtf8Bytes('test')),
          val: '0x1234',
        },
      ])
    )
      .to.emit(AttestationStation, 'AttestationCreated')
      .withArgs(
        signer2.address,
        signer2.address,
        ethers.utils.keccak256(ethers.utils.toUtf8Bytes('test')),
        '0x1234'
      )
  })
})
