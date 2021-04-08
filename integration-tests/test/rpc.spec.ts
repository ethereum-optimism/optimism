import { ethers } from 'hardhat'
import { Wallet, BigNumber } from 'ethers'
import chai, { expect } from 'chai'
import { sleep } from './shared/utils'
import { injectL2Context } from './shared/l2provider'
import chaiAsPromised from 'chai-as-promised'
chai.use(chaiAsPromised)

describe('Basic RPC tests', () => {
  const DEFAULT_TRANSACTION = {
    to: '0x' + '1234'.repeat(10),
    gasLimit: 4000000,
    gasPrice: 0,
    data: '0x',
    value: 0,
  }

  const provider = injectL2Context(ethers.provider)
  const wallet = Wallet.createRandom().connect(provider)

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

  describe('eth_chainId', () => {
    it('should get the correct chainid', async () => {
      const { chainId } = await provider.getNetwork()
      expect(chainId).to.be.eq(420)
    })
  })

  describe('eth_gasPrice', () => {
    it('gas price should be 0', async () => {
      const expected = 0
      const price = await provider.getGasPrice()

      expect(price.toNumber()).to.equal(expected)
    })
  })

  describe('eth_estimateGas', () => {
    it('should return a gas estimate', async () => {
      // Repeat this test for a series of possible transaction sizes.
      for (const size of [0, 2, 8, 64, 256]) {
        const estimate = await provider.estimateGas({
          ...DEFAULT_TRANSACTION,
          data: '0x' + '00'.repeat(size),
        })

        // Ths gas estimation is set to always be the max gas limit - 1.
        expect(estimate.toNumber()).to.be.eq(8999999)
      }
    })
  })
})
