import {
  INVALID_ADDRESS,
  keccak256FromUtf8,
  ONE,
  remove0x,
  rsvToSignature,
  TestUtils,
  ZERO,
  ZERO_ADDRESS,
} from '@eth-optimism/core-utils'
import { PostgresDB, RDB, Row } from '@eth-optimism/core-db'
import { BigNumber as BigNum } from 'ethers/utils'
import { Block, TransactionResponse } from 'ethers/providers'

/* Internal Imports */
import {
  CHAIN_ID,
  L2_ROLLUP_TX_SIZE_IN_BYTES_MINUS_CALLDATA,
} from '../../src/app'
import {
  BatchSubmissionStatus,
  DataService,
  QueueOrigin,
} from '../../src/types/data'
import { RollupTransaction, TransactionOutput } from '../../src/types'

export const blockHash = keccak256FromUtf8('block hash')
export const parentHash = keccak256FromUtf8('parent hash')
export const blockNumber = 0
export const defaultTimestamp = 1
export const defaultData: string = '0xdeadbeef'
export const defaultFrom: string = ZERO_ADDRESS
export const defaultTo: string = INVALID_ADDRESS
export const defaultNonceString: string = '0x01'
export const defaultNonceNum: number = 1
export const defaultSignature: string = `${blockHash}${remove0x(parentHash)}99`
export const defaultStateRoot: string = keccak256FromUtf8(blockHash)

export const defaultTxSizeInBytes: number =
  remove0x(defaultData).length / 2 + L2_ROLLUP_TX_SIZE_IN_BYTES_MINUS_CALLDATA

export const gasUsed = new BigNum(1)
export const gasLimit = new BigNum(2)
export const gasPrice = new BigNum(3)

export const l1Block: Block = {
  hash: blockHash,
  parentHash,
  number: blockNumber,
  timestamp: defaultTimestamp,
  nonce: defaultNonceString,
  difficulty: 1234,
  gasLimit,
  gasUsed,
  miner: 'miner',
  extraData: 'extra',
  transactions: [],
}

export const getTxSizeInBytes = (txOutput: TransactionOutput) => {
  return (
    remove0x(txOutput.calldata).length / 2 +
    L2_ROLLUP_TX_SIZE_IN_BYTES_MINUS_CALLDATA
  )
}

export const deleteAllData = async (postgres: PostgresDB): Promise<void> => {
  await postgres.execute(`DELETE FROM l2_tx_output`)
  await postgres.execute(`DELETE FROM state_commitment_chain_batch`)
  await postgres.execute(`DELETE FROM canonical_chain_batch`)
  await postgres.execute(`DELETE FROM l1_rollup_tx`)
  await postgres.execute(`DELETE FROM l1_rollup_state_root`)
  await postgres.execute(`DELETE FROM l1_rollup_state_root_batch`)
  await postgres.execute(`DELETE FROM geth_submission_queue`)
  await postgres.execute(`DELETE FROM l1_tx`)
  await postgres.execute(`DELETE FROM l1_block`)
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
    timestamp: defaultTimestamp,
    hash,
    blockNumber: blockNum,
    blockHash: keccak256FromUtf8('block hash'),
    gasLimit,
    confirmations: 1,
    to,
    from,
    nonce,
    gasPrice,
    value: new BigNum(0),
    chainId: CHAIN_ID,
    v: 1,
    r: keccak256FromUtf8('r'),
    s: keccak256FromUtf8('s'),
    wait: (confirmations) => {
      return undefined
    },
  }
}

export const createTxOutput = (
  hash: string,
  stateRoot: string = defaultStateRoot,
  blockNum: number = blockNumber,
  timestamp: number = defaultTimestamp,
  l1MessageSender?: string,
  from: string = defaultFrom,
  signature: string = defaultSignature,
  data: string = defaultData,
  to: string = defaultTo,
  nonce: number = defaultNonceNum
): TransactionOutput => {
  return {
    calldata: data,
    timestamp,
    transactionHash: hash,
    transactionIndex: 0,
    blockNumber: blockNum,
    gasLimit: ONE,
    gasPrice: ZERO,
    to,
    from,
    nonce,
    l1MessageSender,
    signature: defaultSignature,
    stateRoot,
  }
}

export const createRollupTx = (
  tx: TransactionResponse,
  queueOrigin: QueueOrigin,
  txIndex: number = 0,
  submissionIndex: number = 0,
  l1MessageSender?: string,
  logIndex: number = 0
): RollupTransaction => {
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
    signature: rsvToSignature(tx.r, tx.s, tx.v),
  }
}

export const verifyL1BlockRes = (
  row: Row,
  block: Block,
  processed: boolean
) => {
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
export const verifyL1TxRes = (
  row: Row,
  tx: TransactionResponse,
  index: number
) => {
  row['block_number'].should.equal(
    tx.blockNumber.toString(10),
    `Tx ${index} block number mismatch!`
  )
  row['tx_hash'].should.equal(tx.hash, `Tx ${index} hash mismatch!`)
  row['tx_index'].should.equal(index, `Tx ${index} index mismatch!`)
  row['from_address'].should.equal(tx.from, `Tx ${index} from mismatch!`)
  row['to_address'].should.equal(tx.to, `Tx ${index} to mismatch!`)
  row['nonce'].should.equal(
    tx.nonce.toString(10),
    `Tx ${index} nonce mismatch!`
  )
  row['gas_limit'].should.equal(
    tx.gasLimit.toString(),
    `Tx ${index} gas limit mismatch!`
  )
  row['gas_price'].should.equal(
    tx.gasPrice.toString(),
    `Tx ${index} gas price mismatch!`
  )
  row['calldata'].should.equal(tx.data, `Tx ${index} calldata mismatch!`)
  row['signature'].should.equal(
    rsvToSignature(tx.r, tx.s, tx.v),
    `Tx ${index} signature mismatch!`
  )
}

export const verifyL1RollupTx = (row: Row, tx: RollupTransaction) => {
  TestUtils.nullSafeEquals(row['sender'], tx.sender, 'Sender mismatch!')
  TestUtils.nullSafeEquals(
    row['l1_message_sender'],
    tx.l1MessageSender,
    'L1 Message sender mismatch!'
  )
  TestUtils.nullSafeEquals(row['target'], tx.target, 'Target mismatch!')
  TestUtils.nullSafeEquals(row['calldata'], tx.calldata, 'Calldata mismatch!')
  TestUtils.nullSafeEquals(
    row['queue_origin'],
    tx.queueOrigin,
    'Queue Origin mismatch!'
  )
  TestUtils.nullSafeEquals(
    row['nonce'],
    tx.nonce.toString(10),
    'Nonce mismatch!'
  )
  TestUtils.nullSafeEquals(
    row['gas_limit'],
    tx.gasLimit.toString(10),
    'GasLimit mismatch!'
  )
  TestUtils.nullSafeEquals(
    row['signature'],
    tx.signature,
    'Signature mismatch!'
  )
  TestUtils.nullSafeEquals(
    row['index_within_submission'],
    tx.indexWithinSubmission,
    'Index within submission mismatch!'
  )
  TestUtils.nullSafeEquals(
    row['l1_tx_hash'],
    tx.l1TxHash,
    'L1 Tx Hash mismatch!'
  )
  TestUtils.nullSafeEquals(
    row['l1_tx_index'],
    tx.l1TxIndex,
    'L1 Tx Index mismatch!'
  )
  TestUtils.nullSafeEquals(
    row['l1_tx_log_index'],
    tx.l1TxLogIndex,
    'L1 Tx Log Index mismatch!'
  )
}

export const verifyStateRoot = (
  row: Row,
  root: string,
  index: number,
  batchNumber: number
) => {
  row['state_root'].should.equal(root, `root ${index} mismatch!`)
  row['batch_index'].should.equal(index, `index ${index} mismatch!`)
  row['batch_number'].should.equal(
    batchNumber.toString(10),
    `batch number ${index} mismatch!`
  )
}

export const verifyL2TxOutput = (row: Row, tx: TransactionOutput) => {
  TestUtils.nullSafeEquals(
    row['block_number'],
    tx.blockNumber.toString(10),
    `Block Number mismatch!`
  )
  TestUtils.nullSafeEquals(
    row['block_timestamp'],
    tx.timestamp.toString(10),
    `Timestamp mismatch!`
  )
  TestUtils.nullSafeEquals(
    row['tx_index'],
    tx.transactionIndex,
    `Index mismatch!`
  )
  TestUtils.nullSafeEquals(row['tx_hash'], tx.transactionHash, `Hash mismatch!`)
  TestUtils.nullSafeEquals(row['sender'], tx.from, `Sender mismatch!`)
  TestUtils.nullSafeEquals(
    row['l1_message_sender'],
    tx.l1MessageSender,
    `L1 Message Sender mismatch!`
  )
  TestUtils.nullSafeEquals(row['target'], tx.to, `Target mismatch!`)
  TestUtils.nullSafeEquals(row['calldata'], tx.calldata, `Calldata mismatch!`)
  TestUtils.nullSafeEquals(
    row['nonce'],
    tx.nonce.toString(10),
    `Nonce mismatch!`
  )
  TestUtils.nullSafeEquals(
    row['signature'],
    tx.signature,
    `Signature mismatch!`
  )
  TestUtils.nullSafeEquals(
    row['state_root'],
    tx.stateRoot,
    `State Root mismatch!`
  )
}

export const insertTxOutput = async (
  dataService: DataService,
  tx: TransactionOutput,
  desiredTxBatchStatus?: string,
  desiredRootBatchStatus?: string
): Promise<void> => {
  await dataService.insertL2TransactionOutput(tx)

  let txBatchNumber: number
  if (!!desiredTxBatchStatus) {
    const txSize = getTxSizeInBytes(tx)
    txBatchNumber = await dataService.tryBuildCanonicalChainBatchNotPresentOnL1(
      txSize,
      txSize * 10
    )
    txBatchNumber.should.be.gte(0, 'canonical chain batch not built')

    if (desiredTxBatchStatus === BatchSubmissionStatus.SENT) {
      await dataService.markTransactionBatchSubmittedToL1(
        txBatchNumber,
        keccak256FromUtf8(txBatchNumber.toString(10))
      )
    }

    if (desiredTxBatchStatus === BatchSubmissionStatus.FINALIZED) {
      await dataService.markTransactionBatchFinalOnL1(
        txBatchNumber,
        keccak256FromUtf8(txBatchNumber.toString(10))
      )
    }
  }

  let stateRootBatchNumber: number
  if (!!desiredRootBatchStatus) {
    stateRootBatchNumber = await dataService.tryBuildL2OnlyStateCommitmentChainBatch(
      1,
      1
    )
    stateRootBatchNumber.should.be.gte(0, 'state root chain batch not built')

    if (desiredRootBatchStatus === BatchSubmissionStatus.SENT) {
      await dataService.markStateRootBatchSubmittedToL1(
        stateRootBatchNumber,
        keccak256FromUtf8(txBatchNumber.toString(10))
      )
    }

    if (desiredRootBatchStatus === BatchSubmissionStatus.FINALIZED) {
      await dataService.markStateRootBatchFinalOnL1(
        stateRootBatchNumber,
        keccak256FromUtf8(txBatchNumber.toString(10))
      )
    }
  }
}

export const selectStateRootBatchRes = async (
  rdb: RDB,
  batchNum: number
): Promise<Row[]> => {
  return rdb.select(
    `SELECT * 
            FROM l2_tx_output 
            WHERE state_commitment_chain_batch_number = ${batchNum}
            ORDER BY state_commitment_chain_batch_index ASC`
  )
}
