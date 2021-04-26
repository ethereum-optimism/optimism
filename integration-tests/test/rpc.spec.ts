import { injectL2Context } from '@eth-optimism/core-utils'
import { Wallet, BigNumber, ethers } from 'ethers'
import chai, { expect } from 'chai'
import { sleep, l2Provider, GWEI } from './shared/utils'
import chaiAsPromised from 'chai-as-promised'
import { OptimismEnv } from './shared/env'
chai.use(chaiAsPromised)

describe('Basic RPC tests', () => {
  let env: OptimismEnv

  const DEFAULT_TRANSACTION = {
    to: '0x' + '1234'.repeat(10),
    gasLimit: 4000000,
    gasPrice: 0,
    data: '0x',
    value: 0,
  }

  const provider = injectL2Context(l2Provider)
  const wallet = Wallet.createRandom().connect(provider)

  before(async () => {
    env = await OptimismEnv.new()
  })

  describe('eth_sendRawTransaction', () => {
    it('should correctly process a valid transaction', async () => {
      const tx = DEFAULT_TRANSACTION
      const nonce = await wallet.getTransactionCount()
      const result = await wallet.sendTransaction(tx)

      expect(result.from).to.equal(wallet.address)
      expect(result.nonce).to.equal(nonce)
      expect(result.gasLimit.toNumber()).to.equal(tx.gasLimit)
      expect(result.gasPrice.toNumber()).to.equal(tx.gasPrice)
      expect(result.data).to.equal(tx.data)
    })

    it('should not accept a transaction with the wrong chain ID', async () => {
      const tx = {
        ...DEFAULT_TRANSACTION,
        chainId: (await wallet.getChainId()) + 1,
      }

      await expect(
        provider.sendTransaction(await wallet.signTransaction(tx))
      ).to.be.rejectedWith('invalid transaction: invalid sender')
    })

    it('should not accept a transaction without a chain ID', async () => {
      const tx = {
        ...DEFAULT_TRANSACTION,
        chainId: null, // Disables EIP155 transaction signing.
      }

      await expect(
        provider.sendTransaction(await wallet.signTransaction(tx))
      ).to.be.rejectedWith('Cannot submit unprotected transaction')
    })

    it('should accept a transaction with a value', async () => {
      const tx = {
        ...DEFAULT_TRANSACTION,
        chainId: await wallet.getChainId(),
        data: '0x',
        value: ethers.utils.parseEther('5'),
      }

      const balanceBefore = await provider.getBalance(wallet.address)
      await wallet.sendTransaction(tx)

      expect(await provider.getBalance(wallet.address)).to.deep.equal(
        balanceBefore.sub(ethers.utils.parseEther('5'))
      )
    })

    it('should reject a transaction with higher value than user balance', async () => {
      const tx = {
        ...DEFAULT_TRANSACTION,
        chainId: await wallet.getChainId(),
        data: '0x',
        value: ethers.utils.parseEther('100'), // wallet only has 10 eth by default
      }

      await expect(wallet.sendTransaction(tx)).to.be.rejectedWith(
        'invalid transaction: insufficient funds for gas * price + value'
      )
    })
  })

  describe('eth_getTransactionByHash', () => {
    it('should be able to get all relevant l1/l2 transaction data', async () => {
      const tx = DEFAULT_TRANSACTION
      const result = await wallet.sendTransaction(tx)
      await result.wait()

      const transaction = (await provider.getTransaction(result.hash)) as any
      expect(transaction.txType).to.equal('EIP155')
      expect(transaction.queueOrigin).to.equal('sequencer')
      expect(transaction.transactionIndex).to.be.eq(0)
      expect(transaction.gasLimit).to.be.deep.eq(BigNumber.from(tx.gasLimit))
    })
  })

  describe('eth_getBlockByHash', () => {
    it('should return the block and all included transactions', async () => {
      // Send a transaction and wait for it to be mined.
      const tx = DEFAULT_TRANSACTION
      const result = await wallet.sendTransaction(tx)
      const receipt = await result.wait()

      const block = (await provider.getBlockWithTransactions(
        receipt.blockHash
      )) as any

      expect(block.number).to.not.equal(0)
      expect(typeof block.stateRoot).to.equal('string')
      expect(block.transactions.length).to.equal(1)
      expect(block.transactions[0].txType).to.equal('EIP155')
      expect(block.transactions[0].queueOrigin).to.equal('sequencer')
      expect(block.transactions[0].l1TxOrigin).to.equal(null)
    })
  })

  describe('eth_getBlockByNumber', () => {
    // There was a bug that causes transactions to be reingested over
    // and over again only when a single transaction was in the
    // canonical transaction chain. This test catches this by
    // querying for the latest block and then waits and then queries
    // the latest block again and then asserts that they are the same.
    it('should return the same result when new transactions are not applied', async () => {
      // Get latest block once to start.
      const prev = await provider.getBlockWithTransactions('latest')

      // Over ten seconds, repeatedly check the latest block to make sure nothing has changed.
      for (let i = 0; i < 5; i++) {
        const latest = await provider.getBlockWithTransactions('latest')
        expect(latest).to.deep.equal(prev)
        await sleep(2000)
      }
    })
  })

  describe('eth_getBalance', () => {
    it('should get the OVM_ETH balance', async () => {
      const rpcBalance = await provider.getBalance(env.l2Wallet.address)
      const contractBalance = await env.ovmEth.balanceOf(env.l2Wallet.address)
      expect(rpcBalance).to.be.deep.eq(contractBalance)
    })
  })

  describe('eth_chainId', () => {
    it('should get the correct chainid', async () => {
      const { chainId } = await provider.getNetwork()
      expect(chainId).to.be.eq(420)
    })
  })

  describe('eth_gasPrice', () => {
    it('gas price should be 1 gwei', async () => {
      expect(await provider.getGasPrice()).to.be.deep.equal(GWEI)
    })
  })

  describe('eth_estimateGas (returns the fee)', () => {
    it('should return a gas estimate that grows with the size of data', async () => {
      const dataLen = [0, 2, 8, 64, 256]

      let last = BigNumber.from(0)
      // Repeat this test for a series of possible transaction sizes.
      for (const len of dataLen) {
        const estimate = await l2Provider.estimateGas({
          ...DEFAULT_TRANSACTION,
          data: '0x' + '00'.repeat(len),
          from: '0x' + '1234'.repeat(10),
        })

        expect(estimate.gt(last)).to.be.true
        last = estimate
      }
    })
  })
})
