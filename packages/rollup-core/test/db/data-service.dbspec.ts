import '../setup'

/* External Imports */
import { PostgresDB, Row } from '@eth-optimism/core-db'
import {
  keccak256FromUtf8,
  ONE,
  TWO,
  ZERO_ADDRESS,
} from '@eth-optimism/core-utils'
import { BigNumber } from 'ethers/utils'

/* Internal Imports */
import { DefaultDataService } from '../../src/app/data'
import { Block, TransactionResponse } from 'ethers/providers'
import { CHAIN_ID } from '../../src/app'

const blockHash = keccak256FromUtf8('block hash')
const parentHash = keccak256FromUtf8('parent hash')
const blockNumber = 0
const timestamp = 1
const defaultData: string = '0xdeadbeef'
const defaultFrom: string = ZERO_ADDRESS
const defaultNonceString: string = '0x01'
const defaultNonceNum: number = 1

const gasLimit = new BigNumber(2)
const gasUsed = new BigNumber(1)
const gasPrice = new BigNumber(3)

const l1Block: Block = {
  hash: blockHash,
  parentHash,
  number: blockNumber,
  timestamp,
  nonce: defaultNonceString,
  difficulty: 1234,
  gasLimit,
  gasUsed,
  miner: 'miner',
  extraData: 'extra',
  transactions: [],
}

const getTransactionResponse = (
  hash: string,
  blockNum: number = blockNumber,
  data: string = defaultData,
  from: string = defaultFrom,
  nonce: number = defaultNonceNum
): TransactionResponse => {
  return {
    data,
    timestamp: 0,
    hash,
    blockNumber: blockNum,
    blockHash: keccak256FromUtf8('block hash'),
    gasLimit,
    confirmations: 1,
    from,
    nonce,
    gasPrice,
    value: new BigNumber(0),
    chainId: CHAIN_ID,
    v: 1,
    r: keccak256FromUtf8('r'),
    s: keccak256FromUtf8('s'),
    wait: (confirmations) => {
      return undefined
    },
  }
}

const verifyL1BlockRes = (row: Row, block: Block, processed: boolean) => {
  row['block_hash'].should.equal(block.hash, `Hash mismatch!`)
  row['parent_hash'].should.equal(block.parentHash, `Parent hash mismatch!`)
  row['block_number'].should.equal(
    block.number.toString(10),
    `Block number mismatch!`
  )
  row['block_timestamp'].should.equal(
    block.timestamp.toString(10),
    `Block timestamp mismatch!`
  )
  row['processed'].should.equal(processed, `processed mismatch!`)
}
const verifyL1TxRes = (row: Row, tx: TransactionResponse, index: number) => {
  row['tx_hash'].should.equal(tx.hash, `Tx ${index} hash mismatch!`)
  row['tx_index'].should.equal(index, `Index ${index} mismatch!`)
}

describe('Data Service (will fail if postgres is not running with expected schema)', () => {
  let dataService: DefaultDataService
  let postgres: PostgresDB
  before(async () => {
    postgres = new PostgresDB('0.0.0.0', 5432, 'test', 'test', 'rollup')
    dataService = new DefaultDataService(postgres)
  })

  beforeEach(async () => {
    await postgres.execute(`DELETE FROM l2_tx_output`)
    await postgres.execute(`DELETE FROM state_commitment_chain_batch`)
    await postgres.execute(`DELETE FROM canonical_chain_batch`)
    await postgres.execute(`DELETE FROM l1_rollup_tx`)
    await postgres.execute(`DELETE FROM l1_rollup_state_root`)
    await postgres.execute(`DELETE FROM l1_rollup_state_root_batch`)
    await postgres.execute(`DELETE FROM geth_submission_queue`)
    await postgres.execute(`DELETE FROM l1_tx`)
    await postgres.execute(`DELETE FROM l1_block`)
  })

  describe('insertL1Block', () => {
    it('Should insert an L1 block', async () => {
      await dataService.insertL1Block(l1Block)

      const res = await postgres.select(`SELECT * from l1_block`)
      res.length.should.equal(1, `No L1 Block rows!`)
      verifyL1BlockRes(res[0], l1Block, false)
    })

    it('Should insert a processed L1 block', async () => {
      await dataService.insertL1Block(l1Block, true)

      const res = await postgres.select(`SELECT * from l1_block`)
      verifyL1BlockRes(res[0], l1Block, true)
    })
  })

  describe('insertL1Transactions', () => {
    it('Should insert a L1 transactions', async () => {
      await dataService.insertL1Block(l1Block)

      const tx1 = getTransactionResponse(keccak256FromUtf8('tx 1'))
      const tx2 = getTransactionResponse(keccak256FromUtf8('tx 2'))
      await dataService.insertL1Transactions([tx1, tx2])

      const res = await postgres.select(
        `SELECT * from l1_tx ORDER BY tx_index ASC`
      )
      res.length.should.equal(2, `No L1 Tx rows!`)
      verifyL1TxRes(res[0], tx1, 0)
      verifyL1TxRes(res[1], tx2, 1)
    })
  })

  describe('insertL1BlockAndTransactions', () => {
    it('Should insert an L1 block and transactions', async () => {
      const tx1 = getTransactionResponse(keccak256FromUtf8('tx 1'))
      const tx2 = getTransactionResponse(keccak256FromUtf8('tx 2'))

      await dataService.insertL1BlockAndTransactions(l1Block, [tx1, tx2])
      const blockRes = await postgres.select(`SELECT * from l1_block`)
      blockRes.length.should.equal(1, `No L1 Block rows!`)
      verifyL1BlockRes(blockRes[0], l1Block, false)

      const txRes = await postgres.select(
        `SELECT * from l1_tx ORDER BY tx_index ASC`
      )
      txRes.length.should.equal(2, `No L1 Tx rows!`)
      verifyL1TxRes(txRes[0], tx1, 0)
      verifyL1TxRes(txRes[1], tx2, 1)
    })

    it('Should insert an L1 block and transactions (processed)', async () => {
      const tx1 = getTransactionResponse(keccak256FromUtf8('tx 1'))
      const tx2 = getTransactionResponse(keccak256FromUtf8('tx 2'))

      await dataService.insertL1BlockAndTransactions(l1Block, [tx1, tx2], true)
      const blockRes = await postgres.select(`SELECT * from l1_block`)
      blockRes.length.should.equal(1, `No L1 Block rows!`)
      verifyL1BlockRes(blockRes[0], l1Block, true)

      const txRes = await postgres.select(
        `SELECT * from l1_tx ORDER BY tx_index ASC`
      )
      txRes.length.should.equal(2, `No L1 Tx rows!`)
      verifyL1TxRes(txRes[0], tx1, 0)
      verifyL1TxRes(txRes[1], tx2, 1)
    })
  })

  describe('insertL1RollupTransactions', () => {
    it('Should insert an L1 block and transactions', async () => {
      const tx1 = getTransactionResponse(keccak256FromUtf8('tx 1'))
      const tx2 = getTransactionResponse(keccak256FromUtf8('tx 2'))

      await dataService.insertL1BlockAndTransactions(l1Block, [tx1, tx2])
      const blockRes = await postgres.select(`SELECT * from l1_block`)
      blockRes.length.should.equal(1, `No L1 Block rows!`)
      verifyL1BlockRes(blockRes[0], l1Block, false)

      const txRes = await postgres.select(
        `SELECT * from l1_tx ORDER BY tx_index ASC`
      )
      txRes.length.should.equal(2, `No L1 Tx rows!`)
      verifyL1TxRes(txRes[0], tx1, 0)
      verifyL1TxRes(txRes[1], tx2, 1)
    })
  })
})
