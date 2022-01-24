import { BigNumber, Contract, ContractFactory, utils, Wallet } from 'ethers'
import { ethers } from 'hardhat'
import * as L2Artifact from '@eth-optimism/contracts/artifacts/contracts/standards/L2StandardERC20.sol/L2StandardERC20.json'

import { expect } from './shared/setup'
import { OptimismEnv } from './shared/env'
import { withdrawalTest } from './shared/utils'
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
    let tx = await env.l1Wallet.sendTransaction({
      to: otherWalletL1.address,
      value: utils.parseEther('0.01'),
    })
    await tx.wait()
    tx = await env.l2Wallet.sendTransaction({
      to: otherWalletL2.address,
      value: utils.parseEther('0.01'),
    })
    await tx.wait()

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
  })

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

  withdrawalTest(
    'should withdraw tokens from L2 to the depositor',
    async () => {
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
    }
  )

  withdrawalTest(
    'should withdraw tokens from L2 to the transfer recipient',
    async () => {
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
    }
  )

  // This test demonstrates that an apparent withdrawal bug is in fact non-existent.
  // Specifically, the L2 bridge does not check that the L2 token being burned corresponds
  // with the L1 token which is specified for the withdrawal.
  withdrawalTest(
    'should not allow an arbitrary L2 token to be withdrawn in exchange for a legitimate L1 token',
    async () => {
      before(async () => {
        // First deposit some of the L1 token to L2, so that there is something which could be stolen.
        const depositTx = await env.l1Bridge
          .connect(env.l1Wallet)
          .depositERC20(
            L1__ERC20.address,
            L2__ERC20.address,
            1000,
            2000000,
            '0x'
          )
        await env.waitForXDomainTransaction(depositTx, Direction.L1ToL2)
        expect(await L2__ERC20.balanceOf(env.l2Wallet.address)).to.deep.equal(
          BigNumber.from(1000)
        )
      })

      // Deploy a Fake L2 token, which:
      // - returns the address of a legitimate L1 token from its l1Token() getter.
      // - allows the L2 bridge to call its burn() function.
      const fakeToken = await (
        await ethers.getContractFactory('FakeL2StandardERC20', env.l2Wallet)
      ).deploy(L1__ERC20.address)
      await fakeToken.deployed()

      const balBefore = await L1__ERC20.balanceOf(otherWalletL1.address)

      // Withdraw some of the Fake L2 token, hoping to receive the same amount of the legitimate
      // token on L1.
      const withdrawalTx = await env.l2Bridge
        .connect(otherWalletL2)
        .withdrawTo(
          fakeToken.address,
          otherWalletL1.address,
          500,
          1_000_000,
          '0x'
        )
      await env.relayXDomainMessages(withdrawalTx)
      await env.waitForXDomainTransaction(withdrawalTx, Direction.L2ToL1)

      // Ensure that the L1 recipient address has not received any additional L1 token balance.
      expect(await L1__ERC20.balanceOf(otherWalletL1.address)).to.deep.equal(
        balBefore
      )
    }
  )
})
