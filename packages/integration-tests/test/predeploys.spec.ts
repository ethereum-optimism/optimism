/* Imports: External */
import { ethers } from 'ethers'
import { predeploys, getContractInterface } from '@eth-optimism/contracts'

/* Imports: Internal */
import { expect } from './shared/setup'
import { OptimismEnv } from './shared/env'

describe('predeploys', () => {
  let env: OptimismEnv
  before(async () => {
    env = await OptimismEnv.new()
  })

  describe('WETH9', () => {
    let weth9: ethers.Contract
    before(() => {
      weth9 = new ethers.Contract(
        predeploys.WETH9,
        getContractInterface('WETH9'),
        env.l2Wallet
      )
    })

    it('should have the correct name', async () => {
      expect(await weth9.name()).to.equal('Wrapped Ether')
    })

    it('should have the correct symbol', async () => {
      expect(await weth9.symbol()).to.equal('WETH')
    })

    it('should have the correct decimals', async () => {
      expect(await weth9.decimals()).to.equal(18)
    })
  })

  describe('OVM_ETH', () => {
    let ovmEth: ethers.Contract
    before(() => {
      ovmEth = new ethers.Contract(
        predeploys.OVM_ETH,
        getContractInterface('OVM_ETH'),
        env.l2Wallet
      )
    })

    it('should have the correct name', async () => {
      expect(await ovmEth.name()).to.equal('Ether')
    })

    it('should have the correct symbol', async () => {
      expect(await ovmEth.symbol()).to.equal('ETH')
    })

    it('should have the correct decimals', async () => {
      expect(await ovmEth.decimals()).to.equal(18)
    })
  })

  describe('L2CrossDomainMessenger', () => {
    let l2CrossDomainMessenger: ethers.Contract
    before(() => {
      l2CrossDomainMessenger = new ethers.Contract(
        predeploys.L2CrossDomainMessenger,
        getContractInterface('L2CrossDomainMessenger'),
        env.l2Wallet
      )
    })

    it('should throw when calling xDomainMessageSender', async () => {
      await expect(
        l2CrossDomainMessenger.xDomainMessageSender()
      ).to.be.revertedWith('xDomainMessageSender is not set')
    })
  })

  describe('L2StandardBridge', () => {
    let l2StandardBridge: ethers.Contract
    before(() => {
      l2StandardBridge = new ethers.Contract(
        predeploys.L2StandardBridge,
        getContractInterface('L2StandardBridge'),
        env.l2Wallet
      )
    })

    it('should have the correct messenger address', async () => {
      expect(await l2StandardBridge.messenger()).to.equal(
        predeploys.L2CrossDomainMessenger
      )
    })

    it('should have a nonzero bridge address', async () => {
      expect(await l2StandardBridge.l1TokenBridge()).to.not.equal(
        ethers.constants.AddressZero
      )
    })
  })

  describe('OVM_SequencerFeeVault', () => {
    let ovmSequencerFeeVault: ethers.Contract
    before(() => {
      ovmSequencerFeeVault = new ethers.Contract(
        predeploys.OVM_SequencerFeeVault,
        getContractInterface('OVM_SequencerFeeVault'),
        env.l2Wallet
      )
    })

    it('should have a nonzero l1FeeWallet', async () => {
      expect(await ovmSequencerFeeVault.l1FeeWallet()).to.not.equal(
        ethers.constants.AddressZero
      )
    })
  })
})
