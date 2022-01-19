import { expect } from 'chai'
import { ethers } from 'hardhat'
import { Contract, ContractFactory, Signer, BigNumber } from 'ethers'
import { applyL1ToL2Alias } from '@eth-optimism/core-utils'

import { DepositFeed__factory, DepositFeed } from '../../typechain'

const ZERO_ADDRESS = '0x' + '00'.repeat(20)
const ZERO_BIGNUMBER = BigNumber.from(0)
const NON_ZERO_ADDRESS = '0x' + '11'.repeat(20)
const NON_ZERO_GASLIMIT = BigNumber.from(50_000)
const NON_ZERO_VALUE = BigNumber.from(100)
const NON_ZERO_DATA = '0x' + '11'.repeat(42)

const decodeDepositEvent = async (
  depositFeed: DepositFeed
): Promise<{
  from: string
  to: string
  mint: BigNumber
  value: BigNumber
  gasLimit: BigNumber
  isCreation: boolean
  data: string
}> => {
  const events = await depositFeed.queryFilter(
    depositFeed.filters.TransactionDeposited()
  )

  const eventArgs = events[events.length - 1].args

  return {
    from: eventArgs.from,
    to: eventArgs.to,
    mint: eventArgs.mint,
    value: eventArgs.value,
    gasLimit: eventArgs.gasLimit,
    isCreation: eventArgs.isCreation,
    data: eventArgs.data,
  }
}

describe('DepositFeed', () => {
  let signer: Signer
  let signerAddress: string
  let depositFeed: DepositFeed
  before(async () => {
    ;[signer] = await ethers.getSigners()
    signerAddress = await signer.getAddress()
    depositFeed = await new DepositFeed__factory(signer).deploy()
    await depositFeed.deployed()
  })
  it('Should revert if a contract creation has a non-zero destination address', async () => {
    await expect(
      depositFeed.depositTransaction(
        NON_ZERO_ADDRESS,
        NON_ZERO_VALUE,
        NON_ZERO_GASLIMIT,
        true,
        '0x'
      )
    ).to.be.revertedWith('NonZeroCreationTarget()')
  })

  describe('Should emit the correct log values...', async () => {
    it('when an EOA deposits a transaction with 0 value.', async () => {
      const receipt = await(
        await depositFeed.depositTransaction(
          ZERO_ADDRESS,
          ZERO_BIGNUMBER,
          NON_ZERO_GASLIMIT,
          false,
          NON_ZERO_DATA
        )
      ).wait()
      await expect(receipt.status).to.equal(1)

      const eventArgs = await decodeDepositEvent(depositFeed)

      expect(eventArgs).to.deep.equal({
        from: signerAddress,
        to: ZERO_ADDRESS,
        mint: ZERO_BIGNUMBER,
        value: ZERO_BIGNUMBER,
        gasLimit: NON_ZERO_GASLIMIT,
        isCreation: false,
        data: NON_ZERO_DATA,
      })
    })

    it('when a contract deposits a transaction with 0 value.', async () => {
      // Deploy a dummy contract so we can impersonate it
      const dummy = await (await ethers.getContractFactory('Dummy')).deploy()
      await dummy.deployed()

      await expect(
        dummy.forward(
          depositFeed.address,
          depositFeed.interface.encodeFunctionData('depositTransaction', [
            NON_ZERO_ADDRESS,
            ZERO_BIGNUMBER,
            NON_ZERO_GASLIMIT,
            false,
            NON_ZERO_DATA,
          ])
        )
      ).to.not.be.reverted

      const eventArgs = await decodeDepositEvent(depositFeed)

      expect(eventArgs).to.deep.equal({
        from: applyL1ToL2Alias(dummy.address),
        to: NON_ZERO_ADDRESS,
        value: ZERO_BIGNUMBER,
        mint: ZERO_BIGNUMBER,
        gasLimit: NON_ZERO_GASLIMIT,
        isCreation: false,
        data: NON_ZERO_DATA,
      })
    })

    it('when an EOA deposits a contract creation with 0 value.', async () => {
      const receipt = await (
        await depositFeed.depositTransaction(
          ZERO_ADDRESS,
          ZERO_BIGNUMBER,
          NON_ZERO_GASLIMIT,
          true,
          NON_ZERO_DATA
        )
      ).wait()
      await expect(receipt.status).to.equal(1)

      const eventArgs = await decodeDepositEvent(depositFeed)

      expect(eventArgs).to.deep.equal({
        from: signerAddress,
        to: ZERO_ADDRESS,
        value: ZERO_BIGNUMBER,
        mint: ZERO_BIGNUMBER,
        gasLimit: NON_ZERO_GASLIMIT,
        isCreation: true,
        data: NON_ZERO_DATA,
      })
    })

    it('when a contract deposits a contract creation with 0 value.', async () => {
      // Deploy a dummy contract so we can impersonate it
      const dummy = await (await ethers.getContractFactory('Dummy')).deploy()
      await dummy.deployed()

      const receipt = await (
        await dummy.forward(
          depositFeed.address,
          depositFeed.interface.encodeFunctionData('depositTransaction', [
            ZERO_ADDRESS,
            ZERO_BIGNUMBER,
            NON_ZERO_GASLIMIT,
            true,
            NON_ZERO_DATA,
          ])
        )
      ).wait()
      await expect(receipt.status).to.equal(1)

      const eventArgs = await decodeDepositEvent(depositFeed)

      expect(eventArgs).to.deep.equal({
        from: applyL1ToL2Alias(dummy.address),
        to: ZERO_ADDRESS,
        value: ZERO_BIGNUMBER,
        mint: ZERO_BIGNUMBER,
        gasLimit: NON_ZERO_GASLIMIT,
        isCreation: true,
        data: NON_ZERO_DATA,
      })
    })

    describe('and increase its eth balance...', async () => {
      it('when an EOA deposits a transaction with an ETH value.', async () => {
        const balBefore = await ethers.provider.getBalance(depositFeed.address)
        const receipt = await (
          await depositFeed.depositTransaction(
            NON_ZERO_ADDRESS,
            ZERO_BIGNUMBER,
            NON_ZERO_GASLIMIT,
            false,
            '0x',
            {
              value: NON_ZERO_VALUE,
            }
          )
        ).wait()
        await expect(receipt.status).to.equal(1)

        const balAfter = await ethers.provider.getBalance(depositFeed.address)

        const eventArgs = await decodeDepositEvent(depositFeed)

        expect(balAfter.sub(balBefore)).to.equal(NON_ZERO_VALUE)
        expect(eventArgs).to.deep.equal({
          from: signerAddress,
          to: NON_ZERO_ADDRESS,
          value: ZERO_BIGNUMBER,
          mint: NON_ZERO_VALUE,
          gasLimit: NON_ZERO_GASLIMIT,
          isCreation: false,
          data: '0x',
        })
      })

      it('when a contract deposits a transaction with an ETH value.', async () => {
        // Deploy a dummy contract so we can impersonate it
        const dummy = await (await ethers.getContractFactory('Dummy')).deploy()
        await dummy.deployed()
        // this is not emitting an event!
        await expect(
          dummy.forward(
            depositFeed.address,
            depositFeed.interface.encodeFunctionData('depositTransaction', [
              NON_ZERO_ADDRESS,
              ZERO_BIGNUMBER,
              NON_ZERO_GASLIMIT,
              false,
              NON_ZERO_DATA,
            ]),
            {
              value: NON_ZERO_VALUE,
            }
          )
        ).to.not.be.reverted

        const eventArgs = await decodeDepositEvent(depositFeed)

        expect(eventArgs).to.deep.equal({
          from: applyL1ToL2Alias(dummy.address),
          to: NON_ZERO_ADDRESS,
          value: ZERO_BIGNUMBER,
          mint: NON_ZERO_VALUE,
          gasLimit: NON_ZERO_GASLIMIT,
          isCreation: false,
          data: NON_ZERO_DATA,
        })
      })

      it('when an EOA deposits a contract creation with an ETH value.', async () => {
        const balBefore = await ethers.provider.getBalance(depositFeed.address)
        const receipt = await (
          await depositFeed.depositTransaction(
            ZERO_ADDRESS,
            ZERO_BIGNUMBER,
            NON_ZERO_GASLIMIT,
            true,
            '0x',
            {
              value: NON_ZERO_VALUE,
            }
          )
        ).wait()
        await expect(receipt.status).to.equal(1)

        const balAfter = await ethers.provider.getBalance(depositFeed.address)
        const eventArgs = await decodeDepositEvent(depositFeed)

        expect(balAfter.sub(balBefore)).to.equal(NON_ZERO_VALUE)
        expect(eventArgs).to.deep.equal({
          from: signerAddress,
          to: ZERO_ADDRESS,
          value: ZERO_BIGNUMBER,
          mint: NON_ZERO_VALUE,
          gasLimit: NON_ZERO_GASLIMIT,
          isCreation: true,
          data: '0x',
        })
      })

      it('when a contract deposits a contract creation with an ETH value.', async () => {
        // Deploy a dummy contract so we can impersonate it
        const dummy = await (await ethers.getContractFactory('Dummy')).deploy()
        await dummy.deployed()

        const balBefore = await ethers.provider.getBalance(depositFeed.address)
        const receipt = await (
          await dummy.forward(
            depositFeed.address,
            depositFeed.interface.encodeFunctionData('depositTransaction', [
              ZERO_ADDRESS,
              ZERO_BIGNUMBER,
              NON_ZERO_GASLIMIT,
              true,
              NON_ZERO_DATA,
            ]),
            {
              value: NON_ZERO_VALUE,
            }
          )
        ).wait()
        await expect(receipt.status).to.equal(1)

        const balAfter = await ethers.provider.getBalance(depositFeed.address)
        const eventArgs = await decodeDepositEvent(depositFeed)

        expect(balAfter.sub(balBefore)).to.equal(NON_ZERO_VALUE)
        expect(eventArgs).to.deep.equal({
          from: applyL1ToL2Alias(dummy.address),
          to: ZERO_ADDRESS,
          value: ZERO_BIGNUMBER,
          mint: NON_ZERO_VALUE,
          gasLimit: NON_ZERO_GASLIMIT,
          isCreation: true,
          data: NON_ZERO_DATA,
        })
      })
    })
  })
})
