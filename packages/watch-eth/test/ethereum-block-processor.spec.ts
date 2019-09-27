import './setup'

/* External Imports */
import { Block } from 'ethers/providers'
import { DB, getLogger, newInMemoryDB, sleep } from '@pigi/core'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

/* Internal Imports */
import { deployTokenContract, TestListener } from './utils'
import { EthereumBlockProcessor } from '../src'

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

    const blocks: Block[] = await blockListener.waitForReceive(2)

    blocks.length.should.equal(2)
    blocks[0].number.should.equal(0)
    blocks[1].number.should.equal(1)
  }).timeout(timeout)

  it('processes old blocks starting at 1', async () => {
    blockProcessor = new EthereumBlockProcessor(db, 1)
    await blockProcessor.subscribe(provider, blockListener)

    const blocks: Block[] = await blockListener.waitForReceive(1)

    blocks.length.should.equal(1)
    blocks[0].number.should.equal(1)
  }).timeout(timeout)

  it('processes old and new blocks', async () => {
    await blockProcessor.subscribe(provider, blockListener)

    await tokenContract.transfer(
      ownerWallet.address,
      recipientWallet.address,
      sendAmount * 2
    )

    const blocks: Block[] = await blockListener.waitForReceive(3)

    blocks.length.should.equal(3)
    blocks[0].number.should.equal(0)
    blocks[1].number.should.equal(1)
    blocks[2].number.should.equal(2)
    blocks[2].transactions.length.should.equal(1)
  }).timeout(timeout)
})
