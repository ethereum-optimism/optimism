import '../setup'

/* External Imports */
import { DB, newInMemoryDB } from '@pigi/core-db'
import { keccak256, DefaultSignatureProvider } from '@pigi/core-utils'

/* Internal Imports */
import {
  abiEncodeTransition,
  DefaultRollupBlockSubmitter,
  RollupBlock,
  RollupBlockSubmitter,
} from '../../src'

const getIntFromDB = async (db: DB, key: Buffer): Promise<number> => {
  return parseInt((await db.get(key)).toString(), 10)
}

const getLastQueuedFromDB = async (db: DB): Promise<number> => {
  return getIntFromDB(db, DefaultRollupBlockSubmitter.LAST_QUEUED_KEY)
}

const getLastSubmittedFromDB = async (db: DB): Promise<number> => {
  return getIntFromDB(db, DefaultRollupBlockSubmitter.LAST_SUBMITTED_KEY)
}

const getLastConfirmedFromDB = async (db: DB): Promise<number> => {
  return getIntFromDB(db, DefaultRollupBlockSubmitter.LAST_CONFIRMED_KEY)
}

const initQueuedSubmittedConfirmed = async (
  db: DB,
  dummyContract: DummyContract,
  queued: number,
  submitted: number,
  confirmed: number,
  blocks: RollupBlock[] = []
): Promise<RollupBlockSubmitter> => {
  if (queued > 0) {
    await db.put(
      DefaultRollupBlockSubmitter.LAST_QUEUED_KEY,
      Buffer.from(queued.toString(10))
    )
  }
  if (submitted > 0) {
    await db.put(
      DefaultRollupBlockSubmitter.LAST_SUBMITTED_KEY,
      Buffer.from(submitted.toString(10))
    )
  }
  if (confirmed > 0) {
    await db.put(
      DefaultRollupBlockSubmitter.LAST_CONFIRMED_KEY,
      Buffer.from(confirmed.toString(10))
    )
  }

  for (const block of blocks) {
    await db.put(
      DefaultRollupBlockSubmitter.getBlockKey(block.blockNumber),
      DefaultRollupBlockSubmitter.serializeRollupBlockForStorage(block)
    )
  }

  const blockSubmitter: DefaultRollupBlockSubmitter = await DefaultRollupBlockSubmitter.create(
    db,
    // @ts-ignore
    dummyContract
  )

  // queued and confirmed won't change
  blockSubmitter.getLastQueued().should.equal(queued)
  blockSubmitter.getLastConfirmed().should.equal(confirmed)

  let expectedSubmitted: number
  switch (submitted) {
    case queued:
      expectedSubmitted = queued
      break
    case confirmed:
      expectedSubmitted = submitted + 1
      break
    default:
      expectedSubmitted = submitted
  }
  blockSubmitter.getLastSubmitted().should.equal(expectedSubmitted)

  if (expectedSubmitted !== submitted) {
    // Check that block was submitted
    dummyContract.blocksSubmitted.length.should.equal(1)
    dummyContract.blocksSubmitted[0][0].should.equal(
      abiEncodeTransition(blocks[0].transitions[0])
    )

    // Check that last submitted was persisted
    const lastSubmittedFromDB: number = await getLastSubmittedFromDB(db)
    lastSubmittedFromDB.should.equal(expectedSubmitted)
  }

  return blockSubmitter
}

describe('DefaultRollupBlockSubmitter', () => {
  let dummyContract: DummyContract
  let db: DB
  let rollupBlock: RollupBlock
  let rollupBlock2: RollupBlock

  beforeEach(async () => {
    dummyContract = new DummyContract()
    db = newInMemoryDB()

    rollupBlock = {
      blockNumber: 1,
      transitions: [
        {
          stateRoot: keccak256(
            Buffer.from('some stuff to hash').toString('hex')
          ),
          senderSlotIndex: 0,
          recipientSlotIndex: 1,
          tokenType: 0,
          amount: 10,
          signature: await new DefaultSignatureProvider().sign('test'),
        },
      ],
    }

    rollupBlock2 = {
      blockNumber: 2,
      transitions: [
        {
          stateRoot: keccak256(
            Buffer.from('different stuff to hash').toString('hex')
          ),
          senderSlotIndex: 1,
          recipientSlotIndex: 0,
          tokenType: 1,
          amount: 100,
          signature: await new DefaultSignatureProvider().sign('test'),
        },
      ],
    }
  })

  describe('init()', () => {
    it('should init without error when DB empty', async () => {
      await initQueuedSubmittedConfirmed(db, dummyContract, 0, 0, 0)
    })

    it('should init without error when block one submitted but not confirmed', async () => {
      await initQueuedSubmittedConfirmed(db, dummyContract, 1, 1, 0, [
        rollupBlock,
      ])
    })

    it('should init without error when block 2 submitted but not confirmed', async () => {
      await initQueuedSubmittedConfirmed(db, dummyContract, 2, 2, 0, [
        rollupBlock,
        rollupBlock2,
      ])
    })

    it('should try to submit when one queued but not submitted', async () => {
      await initQueuedSubmittedConfirmed(db, dummyContract, 1, 0, 0, [
        rollupBlock,
      ])
    })

    it('should only try to submit one when two queued but not submitted', async () => {
      await initQueuedSubmittedConfirmed(db, dummyContract, 2, 0, 0, [
        rollupBlock,
        rollupBlock2,
      ])
    })
  })

  describe('submitBlock()', () => {
    it('should submit new block with no previous blocks', async () => {
      // @ts-ignore
      const blockSubmitter: RollupBlockSubmitter = await DefaultRollupBlockSubmitter.create(
        db,
        // @ts-ignore
        dummyContract
      )

      await blockSubmitter.submitBlock(rollupBlock)

      dummyContract.blocksSubmitted.length.should.equal(1)
      dummyContract.blocksSubmitted[0][0].should.equal(
        abiEncodeTransition(rollupBlock.transitions[0])
      )

      blockSubmitter.getLastConfirmed().should.equal(0)
      blockSubmitter.getLastSubmitted().should.equal(1)
      blockSubmitter.getLastQueued().should.equal(1)

      const lastQueuedFromDB: number = await getLastQueuedFromDB(db)
      lastQueuedFromDB.should.equal(1)

      const lastSubmittedFromDB: number = await getLastSubmittedFromDB(db)
      lastSubmittedFromDB.should.equal(1)
    })

    it('should submit new block with one previous block', async () => {
      const blockSubmitter = await initQueuedSubmittedConfirmed(
        db,
        dummyContract,
        1,
        1,
        1,
        [rollupBlock]
      )

      await blockSubmitter.submitBlock(rollupBlock2)

      dummyContract.blocksSubmitted.length.should.equal(1)
      dummyContract.blocksSubmitted[0][0].should.equal(
        abiEncodeTransition(rollupBlock2.transitions[0])
      )

      blockSubmitter.getLastConfirmed().should.equal(1)
      blockSubmitter.getLastSubmitted().should.equal(2)
      blockSubmitter.getLastQueued().should.equal(2)

      const lastQueuedFromDB: number = await getLastQueuedFromDB(db)
      lastQueuedFromDB.should.equal(2)

      const lastSubmittedFromDB: number = await getLastSubmittedFromDB(db)
      lastSubmittedFromDB.should.equal(2)
    })

    it('should ignore old block', async () => {
      const blockSubmitter = await initQueuedSubmittedConfirmed(
        db,
        dummyContract,
        1,
        1,
        1,
        [rollupBlock]
      )

      await blockSubmitter.submitBlock(rollupBlock)

      dummyContract.blocksSubmitted.length.should.equal(0)

      blockSubmitter.getLastConfirmed().should.equal(1)
      blockSubmitter.getLastSubmitted().should.equal(1)
      blockSubmitter.getLastQueued().should.equal(1)
    })

    it('should queue block when there is one pending', async () => {
      const blockSubmitter = await initQueuedSubmittedConfirmed(
        db,
        dummyContract,
        1,
        1,
        0,
        [rollupBlock]
      )

      dummyContract.blocksSubmitted.length = 0

      await blockSubmitter.submitBlock(rollupBlock2)

      dummyContract.blocksSubmitted.length.should.equal(0)

      blockSubmitter.getLastConfirmed().should.equal(0)
      blockSubmitter.getLastSubmitted().should.equal(1)
      blockSubmitter.getLastQueued().should.equal(2)

      const lastQueuedFromDB: number = await getLastQueuedFromDB(db)
      lastQueuedFromDB.should.equal(2)
    })
  })

  describe('handleNewRollupBlock()', () => {
    it('should do nothing when there are no pending blocks', async () => {
      // @ts-ignore
      const blockSubmitter: RollupBlockSubmitter = await DefaultRollupBlockSubmitter.create(
        db,
        // @ts-ignore
        dummyContract
      )

      await blockSubmitter.handleNewRollupBlock(1)

      blockSubmitter.getLastConfirmed().should.equal(0)
      blockSubmitter.getLastSubmitted().should.equal(0)
      blockSubmitter.getLastQueued().should.equal(0)
    })

    it('should confirm pending with empty queue', async () => {
      const blockSubmitter = await initQueuedSubmittedConfirmed(
        db,
        dummyContract,
        1,
        1,
        0,
        [rollupBlock]
      )

      await blockSubmitter.handleNewRollupBlock(1)

      blockSubmitter.getLastConfirmed().should.equal(1)
      blockSubmitter.getLastSubmitted().should.equal(1)
      blockSubmitter.getLastQueued().should.equal(1)

      const lastConfirmedFromDB: number = await getLastConfirmedFromDB(db)
      lastConfirmedFromDB.should.equal(1)
    })

    it('should confirm pending with one in queue', async () => {
      const blockSubmitter = await initQueuedSubmittedConfirmed(
        db,
        dummyContract,
        2,
        1,
        0,
        [rollupBlock, rollupBlock2]
      )

      await blockSubmitter.handleNewRollupBlock(1)

      dummyContract.blocksSubmitted.length.should.equal(1)
      dummyContract.blocksSubmitted[0][0].should.equal(
        abiEncodeTransition(rollupBlock2.transitions[0])
      )

      blockSubmitter.getLastConfirmed().should.equal(1)
      blockSubmitter.getLastSubmitted().should.equal(2)
      blockSubmitter.getLastQueued().should.equal(2)

      const lastConfirmedFromDB: number = await getLastConfirmedFromDB(db)
      lastConfirmedFromDB.should.equal(1)

      const lastSubmittedFromDB: number = await getLastSubmittedFromDB(db)
      lastSubmittedFromDB.should.equal(2)
    })

    it('should confirm pending with two in queue', async () => {
      const rollupBlock3: RollupBlock = {
        blockNumber: 3,
        transitions: [
          {
            stateRoot: keccak256(
              Buffer.from('much different stuff to hash').toString('hex')
            ),
            senderSlotIndex: 1,
            recipientSlotIndex: 0,
            tokenType: 1,
            amount: 100,
            signature: await new DefaultSignatureProvider().sign('test'),
          },
        ],
      }
      const blockSubmitter = await initQueuedSubmittedConfirmed(
        db,
        dummyContract,
        3,
        1,
        0,
        [rollupBlock, rollupBlock2, rollupBlock3]
      )

      await blockSubmitter.handleNewRollupBlock(1)

      dummyContract.blocksSubmitted.length.should.equal(1)
      dummyContract.blocksSubmitted[0][0].should.equal(
        abiEncodeTransition(rollupBlock2.transitions[0])
      )

      blockSubmitter.getLastConfirmed().should.equal(1)
      blockSubmitter.getLastSubmitted().should.equal(2)
      blockSubmitter.getLastQueued().should.equal(3)

      const lastConfirmedFromDB: number = await getLastConfirmedFromDB(db)
      lastConfirmedFromDB.should.equal(1)

      const lastSubmittedFromDB: number = await getLastSubmittedFromDB(db)
      lastSubmittedFromDB.should.equal(2)
    })
  })
})

class DummyContract {
  public blocksSubmitted: string[] = []

  public async submitBlock(abiEncodedLeaves: string): Promise<void> {
    this.blocksSubmitted.push(abiEncodedLeaves)
  }
}
