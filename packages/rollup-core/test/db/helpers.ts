import {
  INVALID_ADDRESS,
  keccak256FromUtf8,
  rsvToSignature,
  TestUtils,
  ZERO_ADDRESS
} from '@eth-optimism/core-utils'
import {Row} from '@eth-optimism/core-db'
import {BigNumber} from 'ethers/utils'
import {Block, TransactionResponse} from 'ethers/providers'

/* Internal Imports */
import {CHAIN_ID} from '../../src/app'
import {QueueOrigin} from '../../src/types/data'
import {RollupTransaction} from '../../src/types'

export const blockHash = keccak256FromUtf8('block hash')
export const parentHash = keccak256FromUtf8('parent hash')
export const blockNumber = 0
export const timestamp = 1
export const defaultData: string = '0xdeadbeef'
export const defaultFrom: string = ZERO_ADDRESS
export const defaultTo: string = INVALID_ADDRESS
export const defaultNonceString: string = '0x01'
export const defaultNonceNum: number = 1

export const gasUsed = new BigNumber(1)
export const gasLimit = new BigNumber(2)
export const gasPrice = new BigNumber(3)

export const l1Block: Block = {
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

export const createTx = (
  hash: string,
  blockNum: number = blockNumber,
  data: string = defaultData,
  from: string = defaultFrom,
  to: string = defaultTo,
  nonce: number = defaultNonceNum
): TransactionResponse => {
  return {
    data,
    timestamp,
    hash,
    blockNumber: blockNum,
    blockHash: keccak256FromUtf8('block hash'),
    gasLimit,
    confirmations: 1,
    to,
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

export const createRollupTx = (tx: TransactionResponse, queueOrigin: QueueOrigin, txIndex: number = 0, submissionIndex: number = 0, l1MessageSender?: string, logIndex: number = 0): RollupTransaction => {
  return {
    indexWithinSubmission: submissionIndex,
    target: tx.to,
    calldata: tx.data,
    sender: tx.from,
    l1MessageSender,
    gasLimit: gasLimit.toNumber(),
    l1Timestamp: tx.timestamp,
    l1BlockNumber: tx.blockNumber,
    l1TxIndex: txIndex,
    l1TxHash: tx.hash,
    l1TxLogIndex: logIndex,
    nonce: defaultNonceNum,
    queueOrigin,
    signature: rsvToSignature(tx.r, tx.s, tx.v)
  }
}

export const verifyL1BlockRes = (row: Row, block: Block, processed: boolean) => {
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
export const verifyL1TxRes = (row: Row, tx: TransactionResponse, index: number) => {
  row['block_number'].should.equal(tx.blockNumber.toString(10), `Tx ${index} block number mismatch!`)
  row['tx_hash'].should.equal(tx.hash, `Tx ${index} hash mismatch!`)
  row['tx_index'].should.equal(index, `Tx ${index} index mismatch!`)
  row['from_address'].should.equal(tx.from, `Tx ${index} from mismatch!`)
  row['to_address'].should.equal(tx.to, `Tx ${index} to mismatch!`)
  row['nonce'].should.equal(tx.nonce.toString(10), `Tx ${index} nonce mismatch!`)
  row['gas_limit'].should.equal(tx.gasLimit.toString(), `Tx ${index} gas limit mismatch!`)
  row['gas_price'].should.equal(tx.gasPrice.toString(), `Tx ${index} gas price mismatch!`)
  row['calldata'].should.equal(tx.data, `Tx ${index} calldata mismatch!`)
  row['signature'].should.equal(rsvToSignature(tx.r, tx.s, tx.v), `Tx ${index} signature mismatch!`)
}

export const verifyL1RollupTx = (row: Row, tx: RollupTransaction) => {
  TestUtils.nullSafeEquals(row['sender'], tx.sender, 'Sender mismatch!')
  TestUtils.nullSafeEquals(row['l1_message_sender'], tx.l1MessageSender, 'L1 Message sender mismatch!')
  TestUtils.nullSafeEquals(row['target'], tx.target, 'Target mismatch!')
  TestUtils.nullSafeEquals(row['calldata'], tx.calldata, 'Calldata mismatch!')
  TestUtils.nullSafeEquals(row['queue_origin'], tx.queueOrigin, 'Queue Origin mismatch!')
  TestUtils.nullSafeEquals(row['nonce'], tx.nonce.toString(10), 'Nonce mismatch!')
  TestUtils.nullSafeEquals(row['gas_limit'], tx.gasLimit.toString(10), 'GasLimit mismatch!')
  TestUtils.nullSafeEquals(row['signature'], tx.signature, 'Signature mismatch!')
  TestUtils.nullSafeEquals(row['index_within_submission'], tx.indexWithinSubmission, 'Index within submission mismatch!')
  TestUtils.nullSafeEquals(row['l1_tx_hash'], tx.l1TxHash, 'L1 Tx Hash mismatch!')
  TestUtils.nullSafeEquals(row['l1_tx_index'], tx.l1TxIndex, 'L1 Tx Index mismatch!')
  TestUtils.nullSafeEquals(row['l1_tx_log_index'], tx.l1TxLogIndex, 'L1 Tx Log Index mismatch!')
}