/* Imports: External */
import { expectApprox, getChainId, sleep } from '@eth-optimism/core-utils'
import { Wallet, BigNumber, Contract, ContractFactory, constants } from 'ethers'
import { serialize } from '@ethersproject/transactions'
import { ethers } from 'hardhat'
import {
  TransactionReceipt,
  TransactionRequest,
} from '@ethersproject/providers'

/* Imports: Internal */
import { expect } from './shared/setup'
import {
  defaultTransactionFactory,
  fundUser,
  L2_CHAINID,
  gasPriceForL2,
  isHardhat,
  hardhatTest,
  envConfig,
  gasPriceOracleWallet,
} from './shared/utils'
import { OptimismEnv } from './shared/env'

describe('Basic RPC tests', () => {
  let env: OptimismEnv
  let wallet: Wallet

  let Reverter: Contract
  let ValueContext: Contract
  let revertMessage: string
  let revertingTx: TransactionRequest
  let revertingDeployTx: TransactionRequest

  before(async () => {
    env = await OptimismEnv.new()
    wallet = env.l2Wallet
    const Factory__Reverter = await ethers.getContractFactory(
      'Reverter',
      wallet
    )
    Reverter = await Factory__Reverter.connect(env.l2Wallet).deploy()
    await Reverter.deployTransaction.wait()
    revertMessage = await Reverter.revertMessage()
    revertingTx = {
      to: Reverter.address,
      data: Reverter.interface.encodeFunctionData('doRevert'),
    }
    const Factory__ConstructorReverter = await ethers.getContractFactory(
      'ConstructorReverter',
      wallet
    )
    revertingDeployTx = {
      data: Factory__ConstructorReverter.bytecode,
    }

    // Deploy a contract to check msg.value of the call
    const Factory__ValueContext: ContractFactory =
      await ethers.getContractFactory('ValueContext', wallet)
    ValueContext = await Factory__ValueContext.deploy()
    await ValueContext.deployTransaction.wait()
  })

  describe('eth_sendRawTransaction', () => {
    it('should correctly process a valid transaction', async () => {
      const tx = defaultTransactionFactory()
      tx.gasPrice = await gasPriceForL2()
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
        ...defaultTransactionFactory(),
        gasPrice: await gasPriceForL2(),
        chainId: (await wallet.getChainId()) + 1,
      }

      await expect(
        env.l2Provider.sendTransaction(await wallet.signTransaction(tx))
      ).to.be.rejectedWith('invalid transaction: invalid sender')
    })

    it('should accept a transaction without a chain ID', async () => {
      const tx = {
        ...defaultTransactionFactory(),
        nonce: await wallet.getTransactionCount(),
        gasPrice: await gasPriceForL2(),
        chainId: null, // Disables EIP155 transaction signing.
      }
      const signed = await wallet.signTransaction(tx)
      const response = await env.l2Provider.sendTransaction(signed)

      expect(response.chainId).to.equal(0)
      const v = response.v
      expect(v === 27 || v === 28).to.be.true
    })

    it('should accept a transaction with a value', async () => {
      const tx = {
        ...defaultTransactionFactory(),
        gasPrice: await gasPriceForL2(),
        chainId: await env.l2Wallet.getChainId(),
        data: '0x',
        value: ethers.utils.parseEther('0.1'),
      }

      const balanceBefore = await env.l2Provider.getBalance(
        env.l2Wallet.address
      )
      const result = await env.l2Wallet.sendTransaction(tx)
      const receipt = await result.wait()
      expect(receipt.status).to.deep.equal(1)

      const balAfter = await env.l2Provider.getBalance(env.l2Wallet.address)
      expect(balAfter.lte(balanceBefore.sub(ethers.utils.parseEther('0.1')))).to
        .be.true
    })

    it('should reject a transaction with higher value than user balance', async () => {
      const balance = await env.l2Wallet.getBalance()
      const tx = {
        ...defaultTransactionFactory(),
        gasPrice: await gasPriceForL2(),
        chainId: await env.l2Wallet.getChainId(),
        data: '0x',
        value: balance.add(ethers.utils.parseEther('1')),
      }

      await expect(env.l2Wallet.sendTransaction(tx)).to.be.rejectedWith(
        'invalid transaction: insufficient funds for gas * price + value'
      )
    })

    it('should correctly report OOG for contract creations', async () => {
      const factory = await ethers.getContractFactory('TestOOGInConstructor')

      await expect(factory.connect(wallet).deploy()).to.be.rejectedWith(
        'gas required exceeds allowance'
      )
    })

    it('should reject a transaction with too low of a fee', async () => {
      const isHH = await isHardhat()
      let gasPrice
      if (isHH) {
        gasPrice = await env.messenger.contracts.l2.OVM_GasPriceOracle.connect(
          gasPriceOracleWallet
        ).gasPrice()
        await env.messenger.contracts.l2.OVM_GasPriceOracle.connect(
          gasPriceOracleWallet
        ).setGasPrice(1000)
      }

      const tx = {
        ...defaultTransactionFactory(),
        gasPrice: 1,
      }

      await expect(env.l2Wallet.sendTransaction(tx)).to.be.rejectedWith(
        /gas price too low: 1 wei, use at least tx\.gasPrice = \d+ wei/
      )

      if (isHH) {
        // Reset the gas price to its original price
        await env.messenger.contracts.l2.OVM_GasPriceOracle.connect(
          gasPriceOracleWallet
        ).setGasPrice(gasPrice)
      }
    })

    it('should reject a transaction with too high of a fee', async () => {
      const gasPrice =
        await env.messenger.contracts.l2.OVM_GasPriceOracle.connect(
          gasPriceOracleWallet
        ).gasPrice()
      const largeGasPrice = gasPrice.mul(10)
      const tx = {
        ...defaultTransactionFactory(),
        gasPrice: largeGasPrice,
      }
      await expect(env.l2Wallet.sendTransaction(tx)).to.be.rejectedWith(
        `gas price too high: ${largeGasPrice.toString()} wei, use at most ` +
          `tx.gasPrice = ${gasPrice.toString()} wei`
      )
    })
  })

  describe('eth_call', () => {
    it('should correctly identify call out-of-gas', async () => {
      await expect(
        env.l2Provider.call({
          ...revertingTx,
          gasLimit: 1,
        })
      ).to.be.rejectedWith('out of gas')
    })

    it('should correctly return solidity revert data from a call', async () => {
      await expect(env.l2Provider.call(revertingTx)).to.be.revertedWith(
        revertMessage
      )
    })

    it('should produce error when called from ethers', async () => {
      await expect(Reverter.doRevert()).to.be.revertedWith(revertMessage)
    })

    it('should correctly return revert data from contract creation', async () => {
      await expect(env.l2Provider.call(revertingDeployTx)).to.be.revertedWith(
        revertMessage
      )
    })

    it('should correctly identify contract creation out of gas', async () => {
      await expect(
        env.l2Provider.call({
          ...revertingDeployTx,
          gasLimit: 1,
        })
      ).to.be.rejectedWith('out of gas')
    })

    it('should allow eth_calls with nonzero value', async () => {
      // Fund account to call from
      const from = wallet.address
      const value = 15
      await fundUser(env.messenger, value, from)

      // Do the call and check msg.value
      const data = ValueContext.interface.encodeFunctionData('getCallValue')
      const res = await env.l2Provider.call({
        to: ValueContext.address,
        from,
        data,
        value,
      })

      expect(res).to.eq(BigNumber.from(value))
    })

    // https://github.com/ethereum-optimism/optimism/issues/1998
    it('should use address(0) as the default "from" value', async () => {
      // Do the call and check msg.sender
      const data = ValueContext.interface.encodeFunctionData('getCaller')
      const res = await env.l2Provider.call({
        to: ValueContext.address,
        data,
      })

      const [paddedRes] = ValueContext.interface.decodeFunctionResult(
        'getCaller',
        res
      )

      expect(paddedRes).to.eq(constants.AddressZero)
    })

    it('should correctly use the "from" value', async () => {
      const from = wallet.address

      // Do the call and check msg.sender
      const data = ValueContext.interface.encodeFunctionData('getCaller')
      const res = await env.l2Provider.call({
        to: ValueContext.address,
        from,
        data,
      })

      const [paddedRes] = ValueContext.interface.decodeFunctionResult(
        'getCaller',
        res
      )
      expect(paddedRes).to.eq(from)
    })

    it('should be deterministic', async () => {
      let res = await ValueContext.callStatic.getSelfBalance()
      for (let i = 0; i < 10; i++) {
        const next = await ValueContext.callStatic.getSelfBalance()
        expect(res.toNumber()).to.deep.eq(next.toNumber())
        res = next
      }
    })
  })

  describe('eth_getTransactionReceipt', () => {
    it('correctly exposes revert data for contract calls', async () => {
      const req: TransactionRequest = {
        ...revertingTx,
        gasLimit: 8_000_000, // override gas estimation
      }

      const tx = await wallet.sendTransaction(req)

      let errored = false
      try {
        await tx.wait()
      } catch (e) {
        errored = true
      }
      expect(errored).to.be.true

      const receipt: TransactionReceipt =
        await env.l2Provider.getTransactionReceipt(tx.hash)

      expect(receipt.status).to.eq(0)
    })

    it('correctly exposes revert data for contract creations', async () => {
      const req: TransactionRequest = {
        ...revertingDeployTx,
        gasLimit: 8_000_000, // override gas estimation
      }

      const tx = await wallet.sendTransaction(req)

      let errored = false
      try {
        await tx.wait()
      } catch (e) {
        errored = true
      }
      expect(errored).to.be.true

      const receipt: TransactionReceipt =
        await env.l2Provider.getTransactionReceipt(tx.hash)

      expect(receipt.status).to.eq(0)
    })

    // Optimism special fields on the receipt
    it('includes L1 gas price and L1 gas used', async () => {
      const tx = await env.l2Wallet.populateTransaction({
        to: env.l2Wallet.address,
        gasPrice: await gasPriceForL2(),
      })

      const raw = serialize({
        nonce: parseInt(tx.nonce.toString(), 10),
        to: tx.to,
        gasLimit: tx.gasLimit,
        gasPrice: tx.gasPrice,
        type: tx.type,
        data: tx.data,
      })

      const l1Fee = await env.messenger.contracts.l2.OVM_GasPriceOracle.connect(
        gasPriceOracleWallet
      ).getL1Fee(raw)
      const l1GasPrice =
        await env.messenger.contracts.l2.OVM_GasPriceOracle.connect(
          gasPriceOracleWallet
        ).l1BaseFee()
      const l1GasUsed =
        await env.messenger.contracts.l2.OVM_GasPriceOracle.connect(
          gasPriceOracleWallet
        ).getL1GasUsed(raw)
      const scalar =
        await env.messenger.contracts.l2.OVM_GasPriceOracle.connect(
          gasPriceOracleWallet
        ).scalar()
      const decimals =
        await env.messenger.contracts.l2.OVM_GasPriceOracle.connect(
          gasPriceOracleWallet
        ).decimals()

      const scaled = scalar.toNumber() / 10 ** decimals.toNumber()

      const res = await env.l2Wallet.sendTransaction(tx)
      await res.wait()

      const json = await env.l2Provider.send('eth_getTransactionReceipt', [
        res.hash,
      ])

      expect(l1GasUsed).to.deep.equal(BigNumber.from(json.l1GasUsed))
      expect(l1GasPrice).to.deep.equal(BigNumber.from(json.l1GasPrice))
      expect(scaled.toString()).to.deep.equal(json.l1FeeScalar)
      expect(l1Fee).to.deep.equal(BigNumber.from(json.l1Fee))
    })
  })

  describe('eth_getTransactionByHash', () => {
    it('should be able to get all relevant l1/l2 transaction data', async () => {
      const tx = defaultTransactionFactory()
      tx.gasPrice = await gasPriceForL2()
      const result = await wallet.sendTransaction(tx)
      await result.wait()

      const transaction = (await env.l2Provider.getTransaction(
        result.hash
      )) as any
      expect(transaction.queueOrigin).to.equal('sequencer')
      expect(transaction.transactionIndex).to.be.eq(0)
      expect(transaction.gasLimit).to.be.deep.eq(BigNumber.from(tx.gasLimit))
    })
  })

  describe('eth_getBlockByHash', () => {
    it('should return the block and all included transactions', async () => {
      // Send a transaction and wait for it to be mined.
      const tx = defaultTransactionFactory()
      tx.gasPrice = await gasPriceForL2()
      const result = await wallet.sendTransaction(tx)
      const receipt = await result.wait()

      const block = (await env.l2Provider.getBlockWithTransactions(
        receipt.blockHash
      )) as any

      expect(block.number).to.not.equal(0)
      expect(typeof block.stateRoot).to.equal('string')
      expect(block.transactions.length).to.equal(1)
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
    //
    // Needs to be skipped on Prod networks because this test doesn't work when
    // other people are sending transactions to the Sequencer at the same time
    // as this test is running.
    hardhatTest(
      'should return the same result when new transactions are not applied',
      async () => {
        // Get latest block once to start.
        const prev = await env.l2Provider.getBlockWithTransactions('latest')
        // set wait to null to allow a deep object comparison
        prev.transactions[0].wait = null

        // Over ten seconds, repeatedly check the latest block to make sure nothing has changed.
        for (let i = 0; i < 5; i++) {
          const latest = await env.l2Provider.getBlockWithTransactions('latest')
          latest.transactions[0].wait = null
          // Check each key of the transaction individually
          // for easy debugging if one field changes
          for (const [key, value] of Object.entries(latest.transactions[0])) {
            expect(value).to.deep.equal(
              prev.transactions[0][key],
              `mismatch ${key}`
            )
          }
          expect(latest).to.deep.equal(prev)
          await sleep(2000)
        }
      }
    )
  })

  describe('eth_getBalance', () => {
    it('should get the OVM_ETH balance', async () => {
      const rpcBalance = await env.l2Provider.getBalance(env.l2Wallet.address)
      const contractBalance =
        await env.messenger.contracts.l2.OVM_ETH.balanceOf(env.l2Wallet.address)
      expect(rpcBalance).to.be.deep.eq(contractBalance)
    })
  })

  describe('eth_chainId', () => {
    it('should get the correct chainid', async () => {
      expect(await getChainId(env.l2Provider)).to.be.eq(L2_CHAINID)
    })
  })

  describe('eth_estimateGas', () => {
    it('simple send gas estimation is deterministic', async () => {
      let lastEstimate: BigNumber
      for (let i = 0; i < 10; i++) {
        const estimate = await env.l2Provider.estimateGas({
          to: defaultTransactionFactory().to,
          value: 0,
        })

        if (i > 0) {
          expect(lastEstimate).to.be.eq(estimate)
        }

        lastEstimate = estimate
      }
    })

    it('deterministic gas estimation for evm execution', async () => {
      let res = await ValueContext.estimateGas.getSelfBalance()
      for (let i = 0; i < 10; i++) {
        const next = await ValueContext.estimateGas.getSelfBalance()
        expect(res.toNumber()).to.deep.eq(next.toNumber())
        res = next
      }
    })

    it('should return a gas estimate for txs with empty data', async () => {
      const estimate = await env.l2Provider.estimateGas({
        to: defaultTransactionFactory().to,
        value: 0,
      })
      // Expect gas to be less than or equal to the target plus 1%
      expectApprox(estimate, 21000, { percentUpperDeviation: 1 })
    })

    it('should fail for a reverting call transaction', async () => {
      await expect(env.l2Provider.send('eth_estimateGas', [revertingTx])).to.be
        .reverted
    })

    it('should fail for a reverting deploy transaction', async () => {
      await expect(env.l2Provider.send('eth_estimateGas', [revertingDeployTx]))
        .to.be.reverted
    })
  })

  describe('debug_traceTransaction', () => {
    before(async function () {
      if (!envConfig.RUN_DEBUG_TRACE_TESTS) {
        this.skip()
      }
    })

    it('should match debug_traceBlock', async () => {
      const storage = await ethers.getContractFactory(
        'SimpleStorage',
        env.l2Wallet
      )
      const tx = (await storage.deploy()).deployTransaction
      const receipt = await tx.wait()

      const txTrace = await env.l2Provider.send('debug_traceTransaction', [
        receipt.transactionHash,
      ])
      const blockTrace = await env.l2Provider.send('debug_traceBlockByHash', [
        receipt.blockHash,
      ])
      expect(txTrace).to.deep.equal(blockTrace[0].result)
    })
  })

  describe('rollup_gasPrices', () => {
    it('should return the L1 and L2 gas prices', async () => {
      const result = await env.l2Provider.send('rollup_gasPrices', [])
      const l1GasPrice =
        await env.messenger.contracts.l2.OVM_GasPriceOracle.connect(
          gasPriceOracleWallet
        ).l1BaseFee()
      const l2GasPrice =
        await env.messenger.contracts.l2.OVM_GasPriceOracle.connect(
          gasPriceOracleWallet
        ).gasPrice()
      expect(BigNumber.from(result.l1GasPrice)).to.deep.eq(l1GasPrice)
      expect(BigNumber.from(result.l2GasPrice)).to.deep.eq(l2GasPrice)
    })
  })
})
