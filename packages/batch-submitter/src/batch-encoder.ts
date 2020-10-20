/* External Imports */
import { Signer, ContractFactory, Contract, BigNumber } from 'ethers'
import { TransactionResponse, Block } from '@ethersproject/abstract-provider'
import { keccak256 } from 'ethers/lib/utils'

/* Internal Imports */


/***********
 * Exports *
 ***********/

export interface BatchContext {
  numSequencedTransactions: number
  numSubsequentQueueTransactions: number
  timestamp: number
  blockNumber: number
}

export interface AppendSequencerBatchParams {
  shouldStartAtBatch: number // 5 bytes -- starts at batch
  totalElementsToAppend: number // 3 bytes -- total_elements_to_append
  contexts: BatchContext[] // total_elements[fixed_size[]]
  transactions: string[] // total_size_bytes[],total_size_bytes[]
}

type Bytes32 = string
type Uint16 = number
type Uint8 = number
type Uint24 = number
type Address = string

const TX_TYPE_POSITION = {start: 0, end: 1}
enum TxType {
    EIP155 = 0,
    createEOA = 1
}

const SIGNATURE_FIELD_POSITIONS = {
    v: {start: 1, end: 2},  // 1 byte
    r: {start: 2, end: 34}, // 32 bytes
    s: {start: 34, end: 66} // 32 bytes
}
interface Signature {
    v: string
    r: string
    s: string
}

// CreateEOA TxData
const CREATE_EOA_FIELD_POSITIONS = {
    txType: TX_TYPE_POSITION,          // 1 byte
    sig: SIGNATURE_FIELD_POSITIONS,    // 65 bytes
    messageHash: {start: 66, end: 98}, // 32 bytes
}
interface CreateEOATxData {
    sig: Signature
    messageHash: Bytes32
}

// EIP155 TxData
const EIP155_FIELD_POSITIONS = {
    txType: TX_TYPE_POSITION,       // 1 byte
    sig: SIGNATURE_FIELD_POSITIONS, // 65 bytes
    gasLimit: {start: 66, end: 68}, // 2 bytes
    gasPrice: {start: 68, end: 69}, // 1 byte
    nonce: {start: 69, end: 72},    // 3 bytes
    target: {start: 72, end: 92},   // 20 bytes
    data: {start: 92}               // byte 92 onward
}
interface EIP155TxData {
    sig: Signature 
    gasLimit: Uint16
    gasPrice: Uint8
    nonce: Uint24
    target: Address
    data: string
}

// Encoding helpers
const getLen = (pos: {start, end}) => (pos.end - pos.start)*2
const encodeHex = (val: any, len: number) => remove0x(
  BigNumber.from(val).toHexString()
).padStart(len, '0')

interface Coder {
    encode: Function,
    decode: Function
}

// Coder for CreateEOA; TODO: Write a library which can auto-encode & decode.
interface CreateEOACoder extends Coder {
    encode: (txData: CreateEOATxData)=>string
    decode: (txData: string)=>CreateEOATxData
}
const createEOATxDataCoder: CreateEOACoder = {
    encode: (txData: CreateEOATxData): string => {
        const txType = encodeHex(TxType.createEOA, getLen(CREATE_EOA_FIELD_POSITIONS.txType))

        const v = encodeHex(txData.sig.v, getLen(CREATE_EOA_FIELD_POSITIONS.sig.v))
        const r = encodeHex(txData.sig.r, getLen(CREATE_EOA_FIELD_POSITIONS.sig.r))
        const s = encodeHex(txData.sig.s, getLen(CREATE_EOA_FIELD_POSITIONS.sig.s))

        const messageHash = encodeHex(txData.messageHash, getLen(CREATE_EOA_FIELD_POSITIONS.messageHash))

        return (
            '0x' +
            txType +
            v +
            r +
            s +
            messageHash
        )
    },

    decode: (txData: string): CreateEOATxData => {
        txData = remove0x(txData)
        const sliceBytes = (position: {start, end?}): string => txData.slice(position.start*2, position.end*2)

        const pos = CREATE_EOA_FIELD_POSITIONS
        if (parseInt(sliceBytes(pos.txType), 16) !== TxType.createEOA) {
            throw new Error('Invalid tx type')
        }

        return {
            sig: {
                v: sliceBytes(pos.sig.v),
                r: sliceBytes(pos.sig.r),
                s: sliceBytes(pos.sig.s),
            },
            messageHash: sliceBytes(pos.messageHash),
        }
    }
}

// Coder for eip155; TODO: Write a library which can auto-encode & decode.
interface EIP155Coder extends Coder {
    encode: (txData: EIP155TxData)=>string
    decode: (txData: string)=>EIP155TxData
}
const eip155TxDataCoder: EIP155Coder = {
    encode: (txData: EIP155TxData): string => {
        const txType = encodeHex(TxType.EIP155, getLen(EIP155_FIELD_POSITIONS.txType))

        const v = encodeHex(txData.sig.v, getLen(EIP155_FIELD_POSITIONS.sig.v))
        const r = encodeHex(txData.sig.r, getLen(EIP155_FIELD_POSITIONS.sig.r))
        const s = encodeHex(txData.sig.s, getLen(EIP155_FIELD_POSITIONS.sig.s))

        const gasLimit = encodeHex(txData.gasLimit, getLen(EIP155_FIELD_POSITIONS.gasLimit))
        const gasPrice = encodeHex(txData.gasPrice, getLen(EIP155_FIELD_POSITIONS.gasPrice))
        const nonce = encodeHex(txData.nonce, getLen(EIP155_FIELD_POSITIONS.nonce))
        const target = encodeHex(txData.target, getLen(EIP155_FIELD_POSITIONS.target))
        // Make sure that the data is even
        if (txData.data.length % 2 !== 0) {
            throw new Error('Non-even hex string for tx data!')
        }

        return (
            '0x' +
            txType +
            v +
            r +
            s +
            gasLimit +
            gasPrice +
            nonce +
            target +
            remove0x(txData.data)
        )
    },

    decode: (txData: string): EIP155TxData => {
        txData = remove0x(txData)
        const sliceBytes = (position: {start, end?}): string => txData.slice(position.start*2, position.end*2)

        const pos = EIP155_FIELD_POSITIONS
        if (parseInt(sliceBytes(pos.txType), 16) !== TxType.EIP155) {
            throw new Error('Invalid tx type')
        }

        return {
            sig: {
                v: sliceBytes(pos.sig.v),
                r: sliceBytes(pos.sig.r),
                s: sliceBytes(pos.sig.s),
            },
            gasLimit: parseInt(sliceBytes(pos.gasLimit), 16),
            gasPrice: parseInt(sliceBytes(pos.gasPrice), 16),
            nonce: parseInt(sliceBytes(pos.nonce), 16),
            target: sliceBytes(pos.target),
            data: txData.slice(pos.data.start*2),
        }
    }
}

/*
 * Encoding and decoding functions for all txData types.
 */
export const ctcCoder = {
    createEOATxData: createEOATxDataCoder,
    eip155TxData: eip155TxDataCoder
}

/*
 * OVM_CanonicalTransactionChainContract is a wrapper around a normal Ethers contract
 * where the `appendSequencerBatch(...)` function uses a specialized encoding for improved efficiency.
 */
export class OVM_CanonicalTransactionChainContract extends Contract {
    async appendSequencerBatch(batch: AppendSequencerBatchParams): Promise<TransactionResponse> {
        return appendSequencerBatch(this, batch)
    }
}


/**********************
 * Internal Functions *
 *********************/

const APPEND_SEQUENCER_BATCH_METHOD_ID = 'appendSequencerBatch()'

const appendSequencerBatch = async (
  OVM_CanonicalTransactionChain: Contract,
  batch: AppendSequencerBatchParams
): Promise<TransactionResponse> => {
  const methodId = keccak256(Buffer.from(APPEND_SEQUENCER_BATCH_METHOD_ID)).slice(2, 10)
  const calldata = encodeAppendSequencerBatch(batch)
  return OVM_CanonicalTransactionChain.signer.sendTransaction({
    to: OVM_CanonicalTransactionChain.address,
    data: '0x' + methodId + calldata,
  })
}

const encodeAppendSequencerBatch = (b: AppendSequencerBatchParams): string => {
  let encoding: string
  const encodedShouldStartAtBatch = encodeHex(b.shouldStartAtBatch, 10)
  const encodedTotalElementsToAppend = encodeHex(b.totalElementsToAppend, 6)

  const encodedContextsHeader = encodeHex(b.contexts.length, 6)
  const encodedContexts =
    encodedContextsHeader +
    b.contexts.reduce((acc, cur) => acc + encodeBatchContext(cur), '')

  const encodedTransactionData = b.transactions.reduce((acc, cur) => {
    if (cur.length % 2 !== 0)
      throw new Error('Unexpected uneven hex string value!')
    const encodedTxDataHeader = remove0x(
      BigNumber.from(remove0x(cur).length / 2).toHexString()
    ).padStart(6, '0')
    return acc + encodedTxDataHeader + remove0x(cur)
  }, '')
  return (
    encodedShouldStartAtBatch +
    encodedTotalElementsToAppend +
    encodedContexts +
    encodedTransactionData
  )
}

const encodeBatchContext = (context: BatchContext): string => {
  return (
    encodeHex(context.numSequencedTransactions, 6) +
    encodeHex(context.numSubsequentQueueTransactions, 6) +
    encodeHex(context.timestamp, 10) +
    encodeHex(context.blockNumber, 10)
  )
}

/**
 * Removes '0x' from a hex string.
 * @param str Hex string to remove '0x' from.
 * @returns String without the '0x' prefix.
 */
const remove0x = (str: string): string => {
  if (str.startsWith('0x')) {
    return str.slice(2)
  } else {
    return str
  }
}