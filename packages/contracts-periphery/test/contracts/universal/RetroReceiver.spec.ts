import hre from 'hardhat'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'
import { Contract } from 'ethers'

import { expect } from '../../setup'
import { deploy } from '../../helpers'

describe('RetroReceiver', () => {
  const DEFAULT_TOKEN_ID = 0
  const DEFAULT_AMOUNT = hre.ethers.constants.WeiPerEther
  const DEFAULT_RECIPIENT = '0x' + '11'.repeat(20)

  let signer1: SignerWithAddress
  let signer2: SignerWithAddress
  before('signer setup', async () => {
    ;[signer1, signer2] = await hre.ethers.getSigners()
  })

  let TestERC20: Contract
  let TestERC721: Contract
  let RetroReceiver: Contract
  beforeEach('deploy contracts', async () => {
    TestERC20 = await deploy('TestERC20', { signer: signer1 })
    TestERC721 = await deploy('TestERC721', { signer: signer1 })
    RetroReceiver = await deploy('RetroReceiver', {
      signer: signer1,
      args: [signer1.address],
    })
  })

  beforeEach('balance setup', async () => {
    await TestERC20.mint(signer1.address, hre.ethers.constants.MaxUint256)
    await TestERC721.mint(signer1.address, DEFAULT_TOKEN_ID)
    await hre.ethers.provider.send('hardhat_setBalance', [
      DEFAULT_RECIPIENT,
      '0x0',
    ])
  })

  describe('constructor', () => {
    it('should set the owner', async () => {
      expect(await RetroReceiver.owner()).to.equal(signer1.address)
    })
  })

  describe('receive', () => {
    it('should be able to receive ETH', async () => {
      await expect(
        signer1.sendTransaction({
          to: RetroReceiver.address,
          value: DEFAULT_AMOUNT,
        })
      ).to.not.be.reverted

      expect(
        await hre.ethers.provider.getBalance(RetroReceiver.address)
      ).to.equal(DEFAULT_AMOUNT)
    })
  })

  describe('withdrawETH(address)', () => {
    describe('when called by the owner', () => {
      it('should withdraw all ETH in the contract', async () => {
        await signer1.sendTransaction({
          to: RetroReceiver.address,
          value: DEFAULT_AMOUNT,
        })

        await expect(RetroReceiver['withdrawETH(address)'](DEFAULT_RECIPIENT))
          .to.emit(RetroReceiver, 'WithdrewETH')
          .withArgs(signer1.address, DEFAULT_RECIPIENT, DEFAULT_AMOUNT)

        expect(
          await hre.ethers.provider.getBalance(RetroReceiver.address)
        ).to.equal(0)

        expect(
          await hre.ethers.provider.getBalance(DEFAULT_RECIPIENT)
        ).to.equal(DEFAULT_AMOUNT)
      })
    })

    describe('when called by not the owner', () => {
      it('should revert', async () => {
        await expect(
          RetroReceiver.connect(signer2)['withdrawETH(address)'](
            signer2.address
          )
        ).to.be.revertedWith('UNAUTHORIZED')
      })
    })
  })

  describe('withdrawETH(address,uint256)', () => {
    describe('when called by the owner', () => {
      it('should withdraw the given amount of ETH', async () => {
        await signer1.sendTransaction({
          to: RetroReceiver.address,
          value: DEFAULT_AMOUNT.mul(2),
        })

        await expect(
          RetroReceiver['withdrawETH(address,uint256)'](
            DEFAULT_RECIPIENT,
            DEFAULT_AMOUNT
          )
        )
          .to.emit(RetroReceiver, 'WithdrewETH')
          .withArgs(signer1.address, DEFAULT_RECIPIENT, DEFAULT_AMOUNT)

        expect(
          await hre.ethers.provider.getBalance(RetroReceiver.address)
        ).to.equal(DEFAULT_AMOUNT)

        expect(
          await hre.ethers.provider.getBalance(DEFAULT_RECIPIENT)
        ).to.equal(DEFAULT_AMOUNT)
      })
    })

    describe('when called by not the owner', () => {
      it('should revert', async () => {
        await expect(
          RetroReceiver.connect(signer2)['withdrawETH(address,uint256)'](
            DEFAULT_RECIPIENT,
            DEFAULT_AMOUNT
          )
        ).to.be.revertedWith('UNAUTHORIZED')
      })
    })
  })

  describe('withdrawERC20(address,address)', () => {
    describe('when called by the owner', () => {
      it('should withdraw all ERC20 balance held by the contract', async () => {
        await TestERC20.transfer(RetroReceiver.address, DEFAULT_AMOUNT)

        await expect(
          RetroReceiver['withdrawERC20(address,address)'](
            TestERC20.address,
            DEFAULT_RECIPIENT
          )
        )
          .to.emit(RetroReceiver, 'WithdrewERC20')
          .withArgs(
            signer1.address,
            DEFAULT_RECIPIENT,
            TestERC20.address,
            DEFAULT_AMOUNT
          )

        expect(await TestERC20.balanceOf(DEFAULT_RECIPIENT)).to.equal(
          DEFAULT_AMOUNT
        )
      })
    })

    describe('when called by not the owner', () => {
      it('should revert', async () => {
        await expect(
          RetroReceiver.connect(signer2)['withdrawERC20(address,address)'](
            TestERC20.address,
            DEFAULT_RECIPIENT
          )
        ).to.be.revertedWith('UNAUTHORIZED')
      })
    })
  })

  describe('withdrawERC20(address,address,uint256)', () => {
    describe('when called by the owner', () => {
      it('should withdraw the given ERC20 amount', async () => {
        await TestERC20.transfer(RetroReceiver.address, DEFAULT_AMOUNT.mul(2))

        await expect(
          RetroReceiver['withdrawERC20(address,address,uint256)'](
            TestERC20.address,
            DEFAULT_RECIPIENT,
            DEFAULT_AMOUNT
          )
        )
          .to.emit(RetroReceiver, 'WithdrewERC20')
          .withArgs(
            signer1.address,
            DEFAULT_RECIPIENT,
            TestERC20.address,
            DEFAULT_AMOUNT
          )

        expect(await TestERC20.balanceOf(DEFAULT_RECIPIENT)).to.equal(
          DEFAULT_AMOUNT
        )
      })
    })

    describe('when called by not the owner', () => {
      it('should revert', async () => {
        await expect(
          RetroReceiver.connect(signer2)[
            'withdrawERC20(address,address,uint256)'
          ](TestERC20.address, DEFAULT_RECIPIENT, DEFAULT_AMOUNT)
        ).to.be.revertedWith('UNAUTHORIZED')
      })
    })
  })

  describe('withdrawERC721', () => {
    describe('when called by the owner', () => {
      it('should withdraw the token', async () => {
        await TestERC721.transferFrom(
          signer1.address,
          RetroReceiver.address,
          DEFAULT_TOKEN_ID
        )

        await expect(
          RetroReceiver.withdrawERC721(
            TestERC721.address,
            DEFAULT_RECIPIENT,
            DEFAULT_TOKEN_ID
          )
        )
          .to.emit(RetroReceiver, 'WithdrewERC721')
          .withArgs(
            signer1.address,
            DEFAULT_RECIPIENT,
            TestERC721.address,
            DEFAULT_TOKEN_ID
          )

        expect(await TestERC721.ownerOf(DEFAULT_TOKEN_ID)).to.equal(
          DEFAULT_RECIPIENT
        )
      })
    })

    describe('when called by not the owner', () => {
      it('should revert', async () => {
        await expect(
          RetroReceiver.connect(signer2).withdrawERC721(
            TestERC721.address,
            DEFAULT_RECIPIENT,
            DEFAULT_TOKEN_ID
          )
        ).to.be.revertedWith('UNAUTHORIZED')
      })
    })
  })
})
