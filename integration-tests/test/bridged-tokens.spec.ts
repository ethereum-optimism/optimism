import { expect } from './shared/setup'

import { BigNumber, Contract, ContractFactory, utils, Wallet } from 'ethers'
import { ethers } from 'hardhat'
import * as L2Artifact from '@eth-optimism/contracts/artifacts/contracts/standards/L2StandardERC20.sol/L2StandardERC20.json'

import { OptimismEnv } from './shared/env'
import { isLiveNetwork, isMainnet } from './shared/utils'
import { Direction } from './shared/watcher-utils'

describe('Bridged tokens', () => {
  let env: OptimismEnv

  let otherWalletL1: Wallet
  let otherWalletL2: Wallet

  let L1Factory__ERC20: ContractFactory
  let L1__ERC20: Contract
  let L2Factory__ERC20: ContractFactory
  let L2__ERC20: Contract

  before(async () => {
    env = await OptimismEnv.new()

    const other = Wallet.createRandom()
    otherWalletL1 = other.connect(env.l1Wallet.provider)
    otherWalletL2 = other.connect(env.l2Wallet.provider)
    await env.l1Wallet.sendTransaction({
      to: otherWalletL1.address,
      value: utils.parseEther('0.01'),
    })
    await env.l2Wallet.sendTransaction({
      to: otherWalletL2.address,
      value: utils.parseEther('0.01'),
    })

    L1Factory__ERC20 = await ethers.getContractFactory('ERC20', env.l1Wallet)
    L2Factory__ERC20 = new ethers.ContractFactory(
      L2Artifact.abi,
      L2Artifact.bytecode
    )
    L2Factory__ERC20 = L2Factory__ERC20.connect(env.l2Wallet)
  })

  it('should deploy an ERC20 on L1', async () => {
    L1__ERC20 = await L1Factory__ERC20.deploy(1000000, 'OVM Test', 8, 'OVM')
    await L1__ERC20.deployed()
  })

  it('should deploy a paired token on L2', async () => {
    L2__ERC20 = await L2Factory__ERC20.deploy(
      '0x4200000000000000000000000000000000000010',
      L1__ERC20.address,
      'OVM Test',
      'OVM'
    )
    await L2__ERC20.deployed()
  })

  it('should approve the bridge', async () => {
    const tx = await L1__ERC20.approve(env.l1Bridge.address, 1000000)
    await tx.wait()
  })

  it('should deposit tokens into L2', async () => {
    const tx = await env.l1Bridge.depositERC20(
      L1__ERC20.address,
      L2__ERC20.address,
      1000,
      2000000,
      '0x'
    )
    await env.waitForXDomainTransaction(tx, Direction.L1ToL2)
    expect(await L1__ERC20.balanceOf(env.l1Wallet.address)).to.deep.equal(
      BigNumber.from(999000)
    )
    expect(await L2__ERC20.balanceOf(env.l2Wallet.address)).to.deep.equal(
      BigNumber.from(1000)
    )
  }).timeout(isLiveNetwork() ? 300_000 : 120_000)

  it('should transfer tokens on L2', async () => {
    const tx = await L2__ERC20.transfer(otherWalletL1.address, 500)
    await tx.wait()
    expect(await L2__ERC20.balanceOf(env.l2Wallet.address)).to.deep.equal(
      BigNumber.from(500)
    )
    expect(await L2__ERC20.balanceOf(otherWalletL2.address)).to.deep.equal(
      BigNumber.from(500)
    )
  })

  it('should withdraw tokens from L2 to the depositor', async function () {
    if (await isMainnet(env)) {
      console.log('Skipping withdrawals test on mainnet.')
      this.skip()
      return
    }

    const tx = await env.l2Bridge.withdraw(
      L2__ERC20.address,
      500,
      2000000,
      '0x'
    )
    await env.relayXDomainMessages(tx)
    await env.waitForXDomainTransaction(tx, Direction.L2ToL1)
    expect(await L1__ERC20.balanceOf(env.l1Wallet.address)).to.deep.equal(
      BigNumber.from(999500)
    )
    expect(await L2__ERC20.balanceOf(env.l2Wallet.address)).to.deep.equal(
      BigNumber.from(0)
    )
  }).timeout(isLiveNetwork() ? 300_000 : 120_000)

  it('should withdraw tokens from L2 to the transfer recipient', async function () {
    if (await isMainnet(env)) {
      console.log('Skipping withdrawals test on mainnet.')
      this.skip()
      return
    }

    const tx = await env.l2Bridge
      .connect(otherWalletL2)
      .withdraw(L2__ERC20.address, 500, 2000000, '0x')
    await env.relayXDomainMessages(tx)
    await env.waitForXDomainTransaction(tx, Direction.L2ToL1)
    expect(await L1__ERC20.balanceOf(otherWalletL1.address)).to.deep.equal(
      BigNumber.from(500)
    )
    expect(await L2__ERC20.balanceOf(otherWalletL2.address)).to.deep.equal(
      BigNumber.from(0)
    )
  }).timeout(isLiveNetwork() ? 300_000 : 120_000)
})
