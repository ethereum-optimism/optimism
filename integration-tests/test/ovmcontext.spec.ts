import { expect } from './shared/setup'

/* Imports: External */
import { ethers } from 'hardhat'
import { injectL2Context, expectApprox } from '@eth-optimism/core-utils'
import { predeploys } from '@eth-optimism/contracts'
import { Contract, BigNumber } from 'ethers'

/* Imports: Internal */
import { l2Provider, l1Provider, IS_LIVE_NETWORK } from './shared/utils'
import { OptimismEnv } from './shared/env'
import { Direction } from './shared/watcher-utils'

/**
 * These tests cover the OVM execution contexts. In the OVM execution
 * of a L1 to L2 transaction, both `block.number` and `block.timestamp`
 * must be equal to the blocknumber/timestamp of the L1 transaction.
 */
describe('OVM Context: Layer 2 EVM Context', () => {
  const L2Provider = injectL2Context(l2Provider)
  let env: OptimismEnv
  before(async () => {
    env = await OptimismEnv.new()
  })

  let OVMMulticall: Contract
  let OVMContextStorage: Contract
  beforeEach(async () => {
    const OVMContextStorageFactory = await ethers.getContractFactory(
      'OVMContextStorage',
      env.l2Wallet
    )
    const OVMMulticallFactory = await ethers.getContractFactory(
      'OVMMulticall',
      env.l2Wallet
    )

    OVMContextStorage = await OVMContextStorageFactory.deploy()
    await OVMContextStorage.deployTransaction.wait()
    OVMMulticall = await OVMMulticallFactory.deploy()
    await OVMMulticall.deployTransaction.wait()
  })

  let numTxs = 5
  if (IS_LIVE_NETWORK) {
    // Tests take way too long if we don't reduce the number of txs here.
    numTxs = 1
  }

  it('enqueue: L1 contextual values are correctly set in L2', async () => {
    for (let i = 0; i < numTxs; i++) {
      // Send a transaction from L1 to L2. This will automatically update the L1 contextual
      // information like the L1 block number and L1 timestamp.
      const tx = await env.l1Messenger.sendMessage(
        OVMContextStorage.address,
        '0x',
        2_000_000
      )

      // Wait for the transaction to be sent over to L2.
      await tx.wait()
      const pair = await env.waitForXDomainTransaction(tx, Direction.L1ToL2)

      // Get the L1 block that the enqueue transaction was in so that
      // the timestamp can be compared against the layer two contract
      const l1Block = await l1Provider.getBlock(pair.receipt.blockNumber)
      const l2Block = await l2Provider.getBlock(pair.remoteReceipt.blockNumber)

      // block.number should return the value of the L2 block number.
      const l2BlockNumber = await OVMContextStorage.blockNumbers(i)
      expect(l2BlockNumber.toNumber()).to.deep.equal(l2Block.number)

      // L1BLOCKNUMBER opcode should return the value of the L1 block number.
      const l1BlockNumber = await OVMContextStorage.l1BlockNumbers(i)
      expect(l1BlockNumber.toNumber()).to.deep.equal(l1Block.number)

      // L1 and L2 blocks will have approximately the same timestamp.
      const timestamp = await OVMContextStorage.timestamps(i)
      expectApprox(timestamp.toNumber(), l1Block.timestamp, {
        percentUpperDeviation: 5,
      })
      expect(timestamp.toNumber()).to.deep.equal(l2Block.timestamp)

      // Difficulty should always be zero.
      const difficulty = await OVMContextStorage.difficulty(i)
      expect(difficulty.toNumber()).to.equal(0)

      // Coinbase should always be sequencer fee vault.
      const coinbase = await OVMContextStorage.coinbases(i)
      expect(coinbase).to.equal(predeploys.OVM_SequencerFeeVault)
    }
  }).timeout(150000) // this specific test takes a while because it involves L1 to L2 txs

  it('should set correct OVM Context for `eth_call`', async () => {
    for (let i = 0; i < numTxs; i++) {
      // Make an empty transaction to bump the latest block number.
      const dummyTx = await env.l2Wallet.sendTransaction({
        to: `0x${'11'.repeat(20)}`,
        data: '0x',
      })
      await dummyTx.wait()

      const block = await L2Provider.getBlockWithTransactions('latest')
      const [, returnData] = await OVMMulticall.callStatic.aggregate(
        [
          [
            OVMMulticall.address,
            OVMMulticall.interface.encodeFunctionData(
              'getCurrentBlockTimestamp'
            ),
          ],
          [
            OVMMulticall.address,
            OVMMulticall.interface.encodeFunctionData('getCurrentBlockNumber'),
          ],
          [
            OVMMulticall.address,
            OVMMulticall.interface.encodeFunctionData(
              'getCurrentL1BlockNumber'
            ),
          ],
        ],
        { blockTag: block.number }
      )

      const timestamp = BigNumber.from(returnData[0])
      const blockNumber = BigNumber.from(returnData[1])
      const l1BlockNumber = BigNumber.from(returnData[2])
      const tx = block.transactions[0] as any

      expect(tx.l1BlockNumber).to.deep.equal(l1BlockNumber.toNumber())
      expect(block.timestamp).to.deep.equal(timestamp.toNumber())
      expect(block.number).to.deep.equal(blockNumber.toNumber())
    }
  })

  /**
   * `rollup_getInfo` is a new RPC endpoint that is used to return the OVM
   * context. The data returned should match what is actually being used as the
   * OVM context.
   */

  it('should return same timestamp and blocknumbers between `eth_call` and `rollup_getInfo`', async () => {
    // As atomically as possible, call `rollup_getInfo` and OVMMulticall for the
    // blocknumber and timestamp. If this is not atomic, then the sequencer can
    // happend to update the timestamp between the `eth_call` and the `rollup_getInfo`
    const [info, [, returnData]] = await Promise.all([
      L2Provider.send('rollup_getInfo', []),
      OVMMulticall.callStatic.aggregate([
        [
          OVMMulticall.address,
          OVMMulticall.interface.encodeFunctionData('getCurrentBlockTimestamp'),
        ],
        [
          OVMMulticall.address,
          OVMMulticall.interface.encodeFunctionData('getCurrentL1BlockNumber'),
        ],
      ]),
    ])

    const timestamp = BigNumber.from(returnData[0])
    const blockNumber = BigNumber.from(returnData[1])

    expect(info.ethContext.blockNumber).to.deep.equal(blockNumber.toNumber())
    expect(info.ethContext.timestamp).to.deep.equal(timestamp.toNumber())
  })
})
