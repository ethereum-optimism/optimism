/* Internal Imports */
import { remove0x, toVerifiedBytes, encodeHex, getLen } from '../utils'
import {
  Coder,
  Signature,
  Uint16,
  Uint8,
  Uint24,
  Address,
  Bytes32,
} from './types'

/***********************
 * TxTypes and TxData  *
 **********************/

export enum TxType {
  EIP155 = 0,
  EthSign = 1,
  createEOA = 2,
  none = 3,
}

export const txTypePlainText = {
  0: TxType.EIP155,
  1: TxType.EthSign,
  2: TxType.createEOA,
  3: TxType.none,
  EIP155: TxType.EIP155,
  EthSign: TxType.EthSign,
  CreateEOA: TxType.createEOA,
  None: TxType.none,
}

export interface DefaultEcdsaTxData {
  sig: Signature
  gasLimit: Uint16
  gasPrice: Uint8
  nonce: Uint24
  target: Address
  data: string
}

export interface EIP155TxData extends DefaultEcdsaTxData {}
export interface EthSignTxData extends DefaultEcdsaTxData {}

export interface CreateEOATxData {
  sig: Signature
  messageHash: Bytes32
}

/***********************
 * Encoding Positions  *
 **********************/

/*
 * The positions in the tx data for the different transaction types
 */

export const TX_TYPE_POSITION = { start: 0, end: 1 }

export const SIGNATURE_FIELD_POSITIONS = {
  r: { start: 1, end: 33 }, // 32 bytes
  s: { start: 33, end: 65 }, // 32 bytes
  v: { start: 65, end: 66 }, // 1 byte
}

export const CREATE_EOA_FIELD_POSITIONS = {
  txType: TX_TYPE_POSITION, // 1 byte
  sig: SIGNATURE_FIELD_POSITIONS, // 65 bytes
  messageHash: { start: 66, end: 98 }, // 32 bytes
}

export const DEFAULT_ECDSA_TX_FIELD_POSITIONS = {
  txType: TX_TYPE_POSITION, // 1 byte
  sig: SIGNATURE_FIELD_POSITIONS, // 65 bytes
  gasLimit: { start: 66, end: 69 }, // 3 bytes
  gasPrice: { start: 69, end: 72 }, // 3 byte
  nonce: { start: 72, end: 75 }, // 3 bytes
  target: { start: 75, end: 95 }, // 20 bytes
  data: { start: 95 }, // byte 95 onward
}

export const EIP155_TX_FIELD_POSITIONS = DEFAULT_ECDSA_TX_FIELD_POSITIONS
export const ETH_SIGN_TX_FIELD_POSITIONS = DEFAULT_ECDSA_TX_FIELD_POSITIONS

/***************
 * EcdsaCoders *
 **************/

// Coder for eip155; TODO: Write a library which can auto-encode & decode.
class DefaultEcdsaTxCoder implements Coder {
  constructor(readonly txType: TxType) {}

  public encode(txData: DefaultEcdsaTxData): string {
    const txType = encodeHex(
      this.txType,
      getLen(DEFAULT_ECDSA_TX_FIELD_POSITIONS.txType)
    )

    const r = toVerifiedBytes(
      txData.sig.r,
      getLen(DEFAULT_ECDSA_TX_FIELD_POSITIONS.sig.r)
    )
    const s = toVerifiedBytes(
      txData.sig.s,
      getLen(DEFAULT_ECDSA_TX_FIELD_POSITIONS.sig.s)
    )
    const v = encodeHex(
      txData.sig.v,
      getLen(DEFAULT_ECDSA_TX_FIELD_POSITIONS.sig.v)
    )

    const gasLimit = encodeHex(
      txData.gasLimit,
      getLen(DEFAULT_ECDSA_TX_FIELD_POSITIONS.gasLimit)
    )
    const gasPrice = encodeHex(
      txData.gasPrice,
      getLen(DEFAULT_ECDSA_TX_FIELD_POSITIONS.gasPrice)
    )
    const nonce = encodeHex(
      txData.nonce,
      getLen(DEFAULT_ECDSA_TX_FIELD_POSITIONS.nonce)
    )
    const target = toVerifiedBytes(
      txData.target,
      getLen(DEFAULT_ECDSA_TX_FIELD_POSITIONS.target)
    )
    // Make sure that the data is even
    if (txData.data.length % 2 !== 0) {
      throw new Error('Non-even hex string for tx data!')
    }
    const encoding =
      '0x' +
      txType +
      r +
      s +
      v +
      gasLimit +
      gasPrice +
      nonce +
      target +
      remove0x(txData.data)
    return encoding
  }

  public decode(txData: string): DefaultEcdsaTxData {
    txData = remove0x(txData)
    const sliceBytes = (position: { start; end? }): string =>
      txData.slice(position.start * 2, position.end * 2)

    const pos = DEFAULT_ECDSA_TX_FIELD_POSITIONS
    if (parseInt(sliceBytes(pos.txType), 16) !== TxType.EIP155) {
      throw new Error('Invalid tx type')
    }

    return {
      sig: {
        r: sliceBytes(pos.sig.r),
        s: sliceBytes(pos.sig.s),
        v: sliceBytes(pos.sig.v),
      },
      gasLimit: parseInt(sliceBytes(pos.gasLimit), 16),
      gasPrice: parseInt(sliceBytes(pos.gasPrice), 16),
      nonce: parseInt(sliceBytes(pos.nonce), 16),
      target: sliceBytes(pos.target),
      data: txData.slice(pos.data.start * 2),
    }
  }
}

class EthSignTxCoder extends DefaultEcdsaTxCoder {
  constructor() {
    super(TxType.EthSign)
  }

  public encode(txData: EthSignTxData): string {
    return super.encode(txData)
  }

  public decode(txData: string): EthSignTxData {
    return super.decode(txData)
  }
}

class Eip155TxCoder extends DefaultEcdsaTxCoder {
  constructor() {
    super(TxType.EIP155)
  }

  public encode(txData: EIP155TxData): string {
    return super.encode(txData)
  }

  public decode(txData: string): EIP155TxData {
    return super.decode(txData)
  }
}

class CreateEOATxDataCoder implements Coder {
  public encode(txData: CreateEOATxData): string {
    const txType = encodeHex(
      TxType.createEOA,
      getLen(CREATE_EOA_FIELD_POSITIONS.txType)
    )

    const v = encodeHex(txData.sig.v, getLen(CREATE_EOA_FIELD_POSITIONS.sig.v))
    const r = toVerifiedBytes(
      txData.sig.r,
      getLen(CREATE_EOA_FIELD_POSITIONS.sig.r)
    )
    const s = toVerifiedBytes(
      txData.sig.s,
      getLen(CREATE_EOA_FIELD_POSITIONS.sig.s)
    )

    const messageHash = txData.messageHash

    return '0x' + txType + r + s + v + messageHash
  }

  public decode(txData: string): CreateEOATxData {
    txData = remove0x(txData)
    const sliceBytes = (position: { start; end? }): string =>
      txData.slice(position.start * 2, position.end * 2)

    const pos = CREATE_EOA_FIELD_POSITIONS
    if (parseInt(sliceBytes(pos.txType), 16) !== TxType.createEOA) {
      throw new Error('Invalid tx type')
    }

    return {
      sig: {
        r: sliceBytes(pos.sig.r),
        s: sliceBytes(pos.sig.s),
        v: sliceBytes(pos.sig.v),
      },
      messageHash: sliceBytes(pos.messageHash),
    }
  }
}

/*************
 * ctcCoder  *
 ************/

/*
 * Encoding and decoding functions for all txData types.
 */
export const ctcCoder = {
  createEOATxData: new CreateEOATxDataCoder(),
  eip155TxData: new Eip155TxCoder(),
  ethSignTxData: new EthSignTxCoder(),
}
