import '../../setup'

/* External Imports */
import { getLogger } from '@eth-optimism/core-utils'
import { Block } from 'ethers/providers'
import { createMockProvider, getWallets } from 'ethereum-waffle'

/* Internal Imports */
import { deployTokenContract, TestListener } from './utils'
import { EthereumBlockProcessor, newInMemoryDB } from '../../../src/app'
import { DB } from '../../../src/types'

const log = getLogger('ethereum-block-processor-test', true)

const timeout = 25_000
describe('Block Subscription', () => {
  let provider
  let wallets
  let ownerWallet
  let recipientWallet
  let db: DB

  const sendAmount = 100
  const initialSupply = 100_000

  let blockProcessor: EthereumBlockProcessor
  let blockListener: TestListener<Block>
  let tokenContract

  describe('Instant finalization', () => {
    beforeEach(async () => {
      provider = createMockProvider()
      wallets = getWallets(provider)
      ownerWallet = wallets[0]
      recipientWallet = wallets[1]

      log.debug(`Connection info: ${JSON.stringify(provider.connection)}`)

      tokenContract = await deployTokenContract(ownerWallet, initialSupply)

      db = newInMemoryDB()
      blockProcessor = new EthereumBlockProcessor(db)
      blockListener = new TestListener<Block>()
    })

    it('processes new blocks', async () => {
      await blockProcessor.subscribe(provider, blockListener, false)

      await tokenContract.transfer(
        ownerWallet.address,
        recipientWallet.address,
        sendAmount
      )

      const blocks: Block[] = await blockListener.waitForReceive(1)

      blocks.length.should.equal(1)
      blocks[0].transactions.length.should.equal(1)
    }).timeout(timeout)

    it('processes old blocks', async () => {
      await blockProcessor.subscribe(provider, blockListener)

      const blocks: Block[] = await blockListener.waitForSyncToComplete()

      blocks
        .map((x) => x.number)
        .sort()
        .should.deep.equal([0, 1], `Incorrect blocks received!`)
    }).timeout(timeout)

    it('honors earliest block', async () => {
      blockProcessor = new EthereumBlockProcessor(db, 1)
      await blockProcessor.subscribe(provider, blockListener)

      const blocks: Block[] = await blockListener.waitForSyncToComplete()

      blocks.length.should.equal(1, 'There should only be one block synced!')
      blocks[0].number.should.equal(1, 'Block 1 should have been finalized!')
      blocks[0].transactions.length.should.equal(
        1,
        'There should be 1 transactions in block 1'
      )
      const deployToAddressEmpty = !(blocks[0].transactions[0] as any).to
      deployToAddressEmpty.should.equal(
        true,
        'The "to" address for the deploy tx should be null'
      )
    }).timeout(timeout)

    it('processes blocks starting at 1', async () => {
      blockProcessor = new EthereumBlockProcessor(db, 1)
      await blockProcessor.subscribe(provider, blockListener)

      let blocks: Block[] = await blockListener.waitForSyncToComplete()
      blocks.length.should.equal(1, 'Block 1 should have arrived')

      await tokenContract.transfer(
        ownerWallet.address,
        recipientWallet.address,
        sendAmount * 2
      )

      blocks = await blockListener.waitForReceive(2)
      blocks.length.should.equal(2, `Incorrect number of blocks received!`)

      blocks
        .map((x) => x.number)
        .sort()
        .should.deep.equal([1, 2], `Incorrect blocks received!`)
      blocks
        .filter((x) => x.number === 2)[0]
        .transactions.length.should.equal(1, `Tx Length incorrect!`)
    }).timeout(timeout)

    it('processes old and new blocks', async () => {
      await blockProcessor.subscribe(provider, blockListener)

      await tokenContract.transfer(
        ownerWallet.address,
        recipientWallet.address,
        sendAmount * 2
      )

      const blocks: Block[] = await blockListener.waitForReceive(3)

      blocks.length.should.equal(3, `Incorrect number of blocks received!`)

      blocks
        .map((x) => x.number)
        .sort()
        .should.deep.equal([0, 1, 2], `Incorrect blocks received!`)
      blocks
        .filter((x) => x.number === 2)[0]
        .transactions.length.should.equal(1, `Tx Length incorrect!`)
    }).timeout(timeout)
  })

  describe('Delayed finalization', () => {
    const confirmsUntilFinal = 2
    beforeEach(async () => {
      provider = createMockProvider()
      wallets = getWallets(provider)
      ownerWallet = wallets[0]
      recipientWallet = wallets[1]

      log.debug(`Connection info: ${JSON.stringify(provider.connection)}`)

      db = newInMemoryDB()
      blockProcessor = new EthereumBlockProcessor(db, 0, confirmsUntilFinal)
      blockListener = new TestListener<Block>()
    })

    it('does not process un-finalized block', async () => {
      await blockProcessor.subscribe(provider, blockListener, false)

      const blocks: Block[] = await blockListener.waitForReceive(1, 5_000)

      blocks.length.should.equal(0)
    }).timeout(timeout)

    it('finalizes blocks after enough confirms', async () => {
      await blockProcessor.subscribe(provider, blockListener, false)

      tokenContract = await deployTokenContract(ownerWallet, initialSupply)

      const blocks: Block[] = await blockListener.waitForReceive(1, 5_000)

      blocks.length.should.equal(1, 'Should have received 1 finalized block!')
      blocks[0].number.should.equal(0, 'Block 0 should have been finalized!')
      blocks[0].transactions.length.should.equal(
        0,
        'There should be 0 transactions in block 0'
      )
    }).timeout(timeout)

    it('finalizes multiple blocks after enough confirms', async () => {
      await blockProcessor.subscribe(provider, blockListener, false)

      tokenContract = await deployTokenContract(ownerWallet, initialSupply)
      await tokenContract.transfer(
        ownerWallet.address,
        recipientWallet.address,
        sendAmount * 2
      )

      const blocks: Block[] = await blockListener.waitForReceive(2, 5_000)

      blocks.length.should.equal(2, 'Should have received 2 finalized block!')
      blocks[0].number.should.equal(0, 'Block 0 should have been finalized!')
      blocks[0].transactions.length.should.equal(
        0,
        'There should be 0 transactions in block 0'
      )

      blocks[1].number.should.equal(1, 'Block 1 should have been finalized!')
      blocks[1].transactions.length.should.equal(
        1,
        'There should be 1 transactions in block 1'
      )
      const deployToAddressEmpty = !(blocks[1].transactions[0] as any).to
      deployToAddressEmpty.should.equal(
        true,
        'The "to" address for the deploy tx should be null'
      )
    }).timeout(timeout)

    describe('Syncing past blocks', () => {
      it('does not finalize past blocks if not final', async () => {
        await blockProcessor.subscribe(provider, blockListener, true)

        const blocks: Block[] = await blockListener.waitForReceive(1, 5_000)

        blocks.length.should.equal(0, 'Should not have finalized a block!')
      }).timeout(timeout)

      it('finalizes past block', async () => {
        tokenContract = await deployTokenContract(ownerWallet, initialSupply)

        await blockProcessor.subscribe(provider, blockListener, true)

        const blocks: Block[] = await blockListener.waitForReceive(1, 5_000)

        blocks.length.should.equal(1, 'Should have received 1 finalized block!')
        blocks[0].number.should.equal(0, 'Block 0 should have been finalized!')
        blocks[0].transactions.length.should.equal(
          0,
          'There should be 0 transactions in block 0'
        )
      }).timeout(timeout)

      it('finalizes past and future blocks', async () => {
        tokenContract = await deployTokenContract(ownerWallet, initialSupply)

        await blockProcessor.subscribe(provider, blockListener, true)
        await tokenContract.transfer(
          ownerWallet.address,
          recipientWallet.address,
          sendAmount * 2
        )

        const blocks: Block[] = await blockListener.waitForReceive(2, 5_000)

        blocks.length.should.equal(2, 'Should have received 2 finalized block!')
        blocks[0].number.should.equal(0, 'Block 0 should have been finalized!')
        blocks[0].transactions.length.should.equal(
          0,
          'There should be 0 transactions in block 0'
        )

        blocks[1].number.should.equal(1, 'Block 1 should have been finalized!')
        blocks[1].transactions.length.should.equal(
          1,
          'There should be 1 transactions in block 1'
        )
        const deployToAddressEmpty = !(blocks[1].transactions[0] as any).to
        deployToAddressEmpty.should.equal(
          true,
          'The "to" address for the deploy tx should be null'
        )
      }).timeout(timeout)

      describe('Future earliest block', () => {
        it('does not finalize blocks before the earliest block', async () => {
          tokenContract = await deployTokenContract(ownerWallet, initialSupply)

          blockProcessor = new EthereumBlockProcessor(db, 1, confirmsUntilFinal)
          await blockProcessor.subscribe(provider, blockListener)

          const blocks: Block[] = await blockListener.waitForSyncToComplete()
          blocks.length.should.equal(0)
        }).timeout(timeout)

        it('finalizes blocks after the earliest block', async () => {
          tokenContract = await deployTokenContract(ownerWallet, initialSupply)

          await tokenContract.transfer(
            ownerWallet.address,
            recipientWallet.address,
            sendAmount * 2
          )

          blockProcessor = new EthereumBlockProcessor(db, 1, confirmsUntilFinal)
          await blockProcessor.subscribe(provider, blockListener)

          const blocks: Block[] = await blockListener.waitForSyncToComplete()
          blocks.length.should.equal(1, `Incorrect number of blocks received!`)

          blocks[0].number.should.equal(
            1,
            'Block 1 should have been finalized!'
          )
          blocks[0].transactions.length.should.equal(
            1,
            'There should be 1 transactions in block 1'
          )
          const deployToAddressEmpty = !(blocks[0].transactions[0] as any).to
          deployToAddressEmpty.should.equal(
            true,
            'The "to" address for the deploy tx should be null'
          )
        })
      })
    })
  })
})
