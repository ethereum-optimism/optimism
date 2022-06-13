import { BigNumber, Contract, Wallet } from 'ethers'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'
import { expect } from 'chai'
import { ethers } from 'hardhat'
import * as ethSigUtil from 'eth-sig-util'
import { fromRpcSig } from 'ethereumjs-util'

import {
  MAX_UINT256,
  buildDataPermit,
  buildDataDelegation,
} from './helpers/eip712'
import {
  SECONDS_IN_365_DAYS,
  getBlockTimestamp,
  fastForwardDays,
} from './helpers/time-travel'

describe('Governance Token Testing', () => {
  let network: { chainId: number }
  let governanceToken: Contract
  let mintManager: Contract
  let minter: Wallet
  let optimismMultisig: Wallet
  let userA: Wallet
  let userB: Wallet
  let hardhatSigner1: SignerWithAddress
  let hardhatSigner2: SignerWithAddress
  let hardhatSigner3: SignerWithAddress
  let initialSupply: BigNumber

  before(async () => {
    network = await ethers.provider.getNetwork()
    ;[hardhatSigner1, hardhatSigner2, hardhatSigner3] =
      await ethers.getSigners()

    minter = ethers.Wallet.createRandom().connect(ethers.provider)
    optimismMultisig = ethers.Wallet.createRandom().connect(ethers.provider)
    userA = ethers.Wallet.createRandom().connect(ethers.provider)
    userB = ethers.Wallet.createRandom().connect(ethers.provider)

    await hardhatSigner1.sendTransaction({
      to: minter.address,
      value: ethers.utils.parseEther('2000'),
    })

    await hardhatSigner2.sendTransaction({
      to: userA.address,
      value: ethers.utils.parseEther('2000'),
    })

    await hardhatSigner3.sendTransaction({
      to: userB.address,
      value: ethers.utils.parseEther('2000'),
    })

    // Initial supply is 2^32 tokens
    initialSupply = ethers.BigNumber.from(2)
      .pow(32)
      .mul(ethers.BigNumber.from(10).pow(18))
  })

  beforeEach(async () => {
    const GovernanceToken = await ethers.getContractFactory('GovernanceToken')
    governanceToken = await GovernanceToken.connect(minter).deploy()
    await governanceToken.deployed()

    const MintManager = await ethers.getContractFactory('MintManager')
    mintManager = await MintManager.connect(minter).deploy(
      minter.address,
      governanceToken.address
    )
    await mintManager.deployed()

    await governanceToken.connect(minter).transferOwnership(mintManager.address)
  })

  describe('ERC20 initialisation', async () => {
    it('token metadata is correct', async () => {
      const tokenName = await governanceToken.name()
      expect(tokenName).to.equal('Optimism')

      const tokenSymbol = await governanceToken.symbol()
      expect(tokenSymbol).to.equal('OP')

      const tokenDecimals = await governanceToken.decimals()
      expect(tokenDecimals).to.equal(18)
    })

    it('initial token supply should be 0', async () => {
      const totalSupply = await governanceToken.totalSupply()
      expect(totalSupply).to.be.equal(0)
    })
  })

  describe('Managing token supply', async () => {
    it('owner can mint token', async () => {
      await mintManager
        .connect(minter)
        .mint(optimismMultisig.address, initialSupply)

      const balance = await governanceToken.balanceOf(optimismMultisig.address)
      expect(balance).to.equal(initialSupply)
    })

    it('timestamp for the next allowed mint is correct', async () => {
      const tx = await mintManager
        .connect(minter)
        .mint(optimismMultisig.address, initialSupply)

      const receipt = await ethers.provider.getTransactionReceipt(tx.hash)

      const timestamp = await getBlockTimestamp(receipt.blockNumber)
      const nextAllowedMintTime = await mintManager.mintPermittedAfter()

      expect(nextAllowedMintTime).to.equal(timestamp + SECONDS_IN_365_DAYS)
    })

    it('non-owner cannot mint token', async () => {
      await expect(
        governanceToken.connect(userA).mint(userA.address, 1)
      ).to.be.revertedWith('Ownable: caller is not the owner')
    })

    it('should not be able to mint before the next allowed mint time', async () => {
      await mintManager
        .connect(minter)
        .mint(optimismMultisig.address, initialSupply)

      // Try to mint immediately after token creation and fail
      await expect(
        mintManager.connect(minter).mint(optimismMultisig.address, 1)
      ).to.be.revertedWith('OP: minting not permitted yet')

      // Can mint successfully after 1 year
      await fastForwardDays(365)
      await mintManager.connect(minter).mint(optimismMultisig.address, 1)

      // Cannot mint before the second full year has passed
      await fastForwardDays(364)
      await expect(
        mintManager.connect(minter).mint(optimismMultisig.address, 1)
      ).to.be.revertedWith('OP: minting not permitted yet')

      // Can mint after the second full year has passed
      await fastForwardDays(1)
      await mintManager.connect(minter).mint(optimismMultisig.address, 1)
    })

    it('should be able to mint 2% supply per year', async () => {
      await mintManager
        .connect(minter)
        .mint(optimismMultisig.address, initialSupply)

      // Minting the full 2% after the first year
      let totalSupply = await governanceToken.totalSupply()
      let maxInflationAmount = totalSupply.mul(20).div(1000)

      await fastForwardDays(365)
      await mintManager
        .connect(minter)
        .mint(optimismMultisig.address, maxInflationAmount)

      let updatedTotalSupply = await governanceToken.totalSupply()
      let newTotalSupply = await totalSupply.add(maxInflationAmount)
      expect(updatedTotalSupply).to.equal(newTotalSupply)

      // Minting the full 2% after the second year
      await fastForwardDays(365)
      totalSupply = await governanceToken.totalSupply()
      maxInflationAmount = totalSupply.mul(20).div(1000)

      await mintManager
        .connect(minter)
        .mint(optimismMultisig.address, maxInflationAmount)

      updatedTotalSupply = await governanceToken.totalSupply()
      newTotalSupply = await totalSupply.add(maxInflationAmount)
      expect(updatedTotalSupply).to.equal(newTotalSupply)

      // Minting the full 2% after the third year
      await fastForwardDays(365)
      totalSupply = await governanceToken.totalSupply()
      maxInflationAmount = totalSupply.mul(20).div(1000)

      await mintManager
        .connect(minter)
        .mint(optimismMultisig.address, maxInflationAmount)

      updatedTotalSupply = await governanceToken.totalSupply()
      newTotalSupply = await totalSupply.add(maxInflationAmount)
      expect(updatedTotalSupply).to.equal(newTotalSupply)
    })

    it('should be able to mint less than 2% supply per year', async () => {
      await mintManager
        .connect(minter)
        .mint(optimismMultisig.address, initialSupply)

      await fastForwardDays(365)
      const inflationAmount = initialSupply.mul(20).div(1000).sub(1)

      await mintManager
        .connect(minter)
        .mint(optimismMultisig.address, inflationAmount)

      const updatedTotalSupply = await governanceToken.totalSupply()
      const newTotalSupply = initialSupply.add(inflationAmount)
      expect(updatedTotalSupply).to.equal(newTotalSupply)
    })

    it('should not be able to mint more than 2% supply per year', async () => {
      await mintManager
        .connect(minter)
        .mint(optimismMultisig.address, initialSupply)

      await fastForwardDays(369)
      const inflationAmount = initialSupply.mul(20).div(1000).add(1)

      await expect(
        mintManager
          .connect(minter)
          .mint(optimismMultisig.address, inflationAmount)
      ).to.be.revertedWith('OP: mint amount exceeds cap')
    })

    it('anyone can burn tokens for themselves', async () => {
      await mintManager.connect(minter).mint(minter.address, initialSupply)

      // Give userA 1000 tokens
      const userBalance = ethers.BigNumber.from(10).pow(18).mul(100)
      await governanceToken.connect(minter).transfer(userA.address, userBalance)

      // Burn 200 tokens
      await governanceToken.connect(userA).burn(200)
      const balance = await governanceToken.balanceOf(userA.address)
      expect(balance).to.equal(userBalance.sub(200))
    })

    it('users can burn tokens for others when approved', async () => {
      await mintManager.connect(minter).mint(minter.address, initialSupply)

      // Give userA 1000 tokens
      const userBalance = ethers.BigNumber.from(10).pow(18).mul(1000)
      await governanceToken.connect(minter).transfer(userA.address, userBalance)

      // UserA approves UserB for 200 tokens
      await governanceToken.connect(userA).approve(userB.address, 200)

      // UserB can burn approved 200 tokens
      await governanceToken.connect(userB).burnFrom(userA.address, 200)
      const balance = await governanceToken.balanceOf(userA.address)
      expect(balance).to.equal(userBalance.sub(200))
    })

    it("you cannot burn other users' tokens", async () => {
      await mintManager.connect(minter).mint(minter.address, initialSupply)

      // Give userA 1000 tokens
      const userBalance = ethers.BigNumber.from(10).pow(18).mul(1000)
      await governanceToken.connect(minter).transfer(userA.address, userBalance)

      // UserB fails to burn UserA's tokens
      await expect(
        governanceToken.connect(userB).burnFrom(userA.address, 200)
      ).to.be.revertedWith('ERC20: insufficient allowance')
      const balance = await governanceToken.balanceOf(userA.address)
      expect(balance).to.equal(userBalance)
    })
  })

  describe('Permit tests', async () => {
    it('can use permit for approve', async () => {
      // Check there is no allowance set
      const allowance = await governanceToken.allowance(
        userA.address,
        userB.address
      )
      expect(allowance).to.equal(0)

      const privateKey = userA._signingKey().privateKey
      const privateKeyBuffer = Buffer.from(privateKey.replace(/^0x/, ''), 'hex')

      const nonceUserA = await governanceToken.nonces(userA.address)
      const permittedAmount = ethers.BigNumber.from(10).pow(18).mul(1000)
      const data = buildDataPermit(
        network.chainId,
        governanceToken.address,
        userA.address,
        userB.address,
        permittedAmount.toString(),
        nonceUserA.toNumber()
      )

      const { v, r, s } = fromRpcSig(
        ethSigUtil.signTypedMessage(privateKeyBuffer, { data: data as any })
      )

      await governanceToken
        .connect(userB)
        .permit(
          userA.address,
          userB.address,
          permittedAmount,
          MAX_UINT256,
          v,
          r,
          s
        )

      const allowanceAfter = await governanceToken.allowance(
        userA.address,
        userB.address
      )
      expect(allowanceAfter).to.equal(permittedAmount)
    })

    it('cannot use invalid signature', async () => {
      const permittedAmount = ethers.BigNumber.from(10).pow(18).mul(1000)
      const invalidAmount = permittedAmount.add(1000)

      const privateKey = userA._signingKey().privateKey
      const privateKeyBuffer = Buffer.from(privateKey.replace(/^0x/, ''), 'hex')

      const nonceUserA = await governanceToken.nonces(userA.address)
      const data = buildDataPermit(
        network.chainId,
        governanceToken.address,
        userA.address,
        userB.address,
        permittedAmount.toString(),
        nonceUserA.toNumber()
      )

      const { v, r, s } = fromRpcSig(
        ethSigUtil.signTypedMessage(privateKeyBuffer, { data: data as any })
      )
      await expect(
        governanceToken
          .connect(userB)
          .permit(
            userA.address,
            userB.address,
            invalidAmount,
            MAX_UINT256,
            v,
            r,
            s
          )
      ).to.be.revertedWith('ERC20Permit: invalid signature')
    })
  })

  describe('Delegate voting tests', async () => {
    let userABalance: BigNumber

    beforeEach(async () => {
      await mintManager.connect(minter).mint(minter.address, initialSupply)
      // Give userA 1000 tokens
      userABalance = ethers.BigNumber.from(10).pow(18).mul(1000)
      await governanceToken
        .connect(minter)
        .transfer(userA.address, userABalance)
    })

    it('can delegate vote to self (with tx)', async () => {
      let userADelegate = await governanceToken.delegates(userA.address)
      expect(userADelegate).to.equal(ethers.constants.AddressZero)

      await governanceToken.connect(userA).delegate(userA.address)
      userADelegate = await governanceToken.delegates(userA.address)
      expect(userADelegate).to.equal(userA.address)
    })

    it('can delegate vote to self (with signature)', async () => {
      let userADelegate = await governanceToken.delegates(userA.address)
      expect(userADelegate).to.equal(ethers.constants.AddressZero)

      const privateKey = userA._signingKey().privateKey
      const privateKeyBuffer = Buffer.from(privateKey.replace(/^0x/, ''), 'hex')

      const nonce = await governanceToken.nonces(userA.address)
      const data = buildDataDelegation(
        network.chainId,
        governanceToken.address,
        userA.address,
        nonce.toNumber()
      )

      const { v, r, s } = fromRpcSig(
        ethSigUtil.signTypedMessage(privateKeyBuffer, { data: data as any })
      )

      await governanceToken
        .connect(userA)
        .delegateBySig(userA.address, nonce, MAX_UINT256, v, r, s)

      userADelegate = await governanceToken.delegates(userA.address)
      expect(userADelegate).to.equal(userA.address)
    })

    it('can delegate vote to third party (with tx)', async () => {
      // Check the delegate for userA is 0 and their votes are 0
      let userADelegate = await governanceToken.delegates(userA.address)
      expect(userADelegate).to.equal(ethers.constants.AddressZero)
      let userAVotes = await governanceToken.getVotes(userA.address)
      expect(userAVotes).to.equal(0)

      await governanceToken.connect(userA).delegate(userA.address)
      userADelegate = await governanceToken.delegates(userA.address)
      expect(userADelegate).to.equal(userA.address)
      userAVotes = await governanceToken.getVotes(userA.address)
      expect(userAVotes).to.equal(userABalance)
    })

    it('can delegate vote to third party (with signature)', async () => {
      // Check the delegate for userA is 0
      let userADelegate = await governanceToken.delegates(userA.address)
      expect(userADelegate).to.equal(ethers.constants.AddressZero)
      // Check the votes for both userA and userB are 0
      let userAVotes = await governanceToken.getVotes(userA.address)
      expect(userAVotes).to.equal(0)
      let userBVotes = await governanceToken.getVotes(userB.address)
      expect(userBVotes).to.equal(0)

      const privateKey = userA._signingKey().privateKey
      const privateKeyBuffer = Buffer.from(privateKey.replace(/^0x/, ''), 'hex')

      const nonce = await governanceToken.nonces(userA.address)
      const data = buildDataDelegation(
        network.chainId,
        governanceToken.address,
        userB.address,
        nonce.toNumber()
      )

      const { v, r, s } = fromRpcSig(
        ethSigUtil.signTypedMessage(privateKeyBuffer, { data: data as any })
      )

      await governanceToken
        .connect(userB)
        .delegateBySig(userB.address, nonce, MAX_UINT256, v, r, s)

      // Check the delegatee for userA is userB
      userADelegate = await governanceToken.delegates(userA.address)
      expect(userADelegate).to.equal(userB.address)

      // Check the userA votes are 0 and userB has all of userA's votes (through delegation)
      userAVotes = await governanceToken.getVotes(userA.address)
      expect(userAVotes).to.equal(0)
      userBVotes = await governanceToken.getVotes(userB.address)
      expect(userBVotes).to.equal(userABalance)
    })
  })
})
