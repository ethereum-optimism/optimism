/* Imports: External */
import { BigNumber, Contract, ContractFactory, utils, Wallet } from 'ethers'
import { ethers } from 'hardhat'
import { getContractFactory } from '@eth-optimism/contracts'
import { MessageStatus } from '@eth-optimism/sdk'

/* Imports: Internal */
import { expect } from './shared/setup'
import { OptimismEnv } from './shared/env'
import { withdrawalTest } from './shared/utils'

describe('Bridged tokens', () => {
  let env: OptimismEnv
  before(async () => {
    env = await OptimismEnv.new()
  })

  let otherWalletL1: Wallet
  let otherWalletL2: Wallet
  before(async () => {
    const other = Wallet.createRandom()
    otherWalletL1 = other.connect(env.l1Wallet.provider)
    otherWalletL2 = other.connect(env.l2Wallet.provider)

    const tx1 = await env.l1Wallet.sendTransaction({
      to: otherWalletL1.address,
      value: utils.parseEther('0.01'),
    })
    await tx1.wait()
    const tx2 = await env.l2Wallet.sendTransaction({
      to: otherWalletL2.address,
      value: utils.parseEther('0.01'),
    })
    await tx2.wait()
  })

  let L1Factory__ERC20: ContractFactory
  let L2Factory__ERC20: ContractFactory
  before(async () => {
    L1Factory__ERC20 = await ethers.getContractFactory('ERC20', env.l1Wallet)
    L2Factory__ERC20 = getContractFactory('L2StandardERC20', env.l2Wallet)
  })

  // This is one of the only stateful integration tests in which we don't set up a new contract
  // before each test. We do this because the test is more of an "actor-based" test where we're
  // going through a series of actions and confirming that the actions are performed correctly at
  // every step.
  let L1__ERC20: Contract
  let L2__ERC20: Contract
  before(async () => {
    // Deploy the L1 ERC20
    L1__ERC20 = await L1Factory__ERC20.deploy(1000000, 'OVM Test', 8, 'OVM')
    await L1__ERC20.deployed()

    // Deploy the L2 ERC20
    L2__ERC20 = await L2Factory__ERC20.deploy(
      '0x4200000000000000000000000000000000000010',
      L1__ERC20.address,
      'OVM Test',
      'OVM'
    )
    await L2__ERC20.deployed()

    // Approve the L1 ERC20 to spend our money
    const tx = await L1__ERC20.approve(
      env.messenger.contracts.l1.L1StandardBridge.address,
      1000000
    )
    await tx.wait()
  })

  it('should deposit tokens into L2', async () => {
    await env.messenger.waitForMessageReceipt(
      await env.messenger.depositERC20(
        L1__ERC20.address,
        L2__ERC20.address,
        1000
      )
    )

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
      const tx = await env.messenger.withdrawERC20(
        L1__ERC20.address,
        L2__ERC20.address,
        500
      )

      await env.messenger.waitForMessageStatus(
        tx,
        MessageStatus.READY_FOR_RELAY
      )

      await env.messenger.finalizeMessage(tx)
      await env.messenger.waitForMessageReceipt(tx)

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
      const tx = await env.messenger.withdrawERC20(
        L1__ERC20.address,
        L2__ERC20.address,
        500,
        {
          signer: otherWalletL2,
        }
      )

      await env.messenger.waitForMessageStatus(
        tx,
        MessageStatus.READY_FOR_RELAY
      )

      await env.messenger.finalizeMessage(tx)
      await env.messenger.waitForMessageReceipt(tx)

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
      // First deposit some of the L1 token to L2, so that there is something which could be stolen.
      await env.messenger.waitForMessageReceipt(
        await env.messenger.depositERC20(
          L1__ERC20.address,
          L2__ERC20.address,
          1000
        )
      )

      expect(await L2__ERC20.balanceOf(env.l2Wallet.address)).to.deep.equal(
        BigNumber.from(1000)
      )

      // Deploy a Fake L2 token, which:
      // - returns the address of a legitimate L1 token from its l1Token() getter.
      // - allows the L2 bridge to call its burn() function.
      const fakeToken = await (
        await ethers.getContractFactory('FakeL2StandardERC20', env.l2Wallet)
      ).deploy(
        L1__ERC20.address,
        env.messenger.contracts.l2.L2StandardBridge.address
      )
      await fakeToken.deployed()

      const balBefore = await L1__ERC20.balanceOf(otherWalletL1.address)

      // Withdraw some of the Fake L2 token, hoping to receive the same amount of the legitimate
      // token on L1.
      const withdrawalTx = await env.messenger.withdrawERC20(
        L1__ERC20.address,
        fakeToken.address,
        500,
        {
          signer: otherWalletL2,
        }
      )

      await env.messenger.waitForMessageStatus(
        withdrawalTx,
        MessageStatus.READY_FOR_RELAY
      )

      await env.messenger.finalizeMessage(withdrawalTx)
      await env.messenger.waitForMessageReceipt(withdrawalTx)

      // Ensure that the L1 recipient address has not received any additional L1 token balance.
      expect(await L1__ERC20.balanceOf(otherWalletL1.address)).to.deep.equal(
        balBefore
      )
    }
  )
})
