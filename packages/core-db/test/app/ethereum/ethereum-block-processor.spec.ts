import '../../setup'

/* External Imports */
import { getLogger, sleep } from '@eth-optimism/core-utils'
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

  beforeEach(async () => {
    provider = createMockProvider()
    wallets = getWallets(provider)
    ownerWallet = wallets[0]
    recipientWallet = wallets[1]

    log.debug(`Connection info: ${JSON.stringify(provider.connection)}`)

    tokenContract = await deployTokenContract(ownerWallet, initialSupply)

    // Wait for deploy block to be mined
    await sleep(25)

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

    blocks.length.should.equal(0)
  }).timeout(timeout)

  it('processes blocks starting at 1', async () => {
    blockProcessor = new EthereumBlockProcessor(db, 1)
    await blockProcessor.subscribe(provider, blockListener)

    let blocks: Block[] = await blockListener.waitForSyncToComplete()
    blocks.length.should.equal(0)

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
