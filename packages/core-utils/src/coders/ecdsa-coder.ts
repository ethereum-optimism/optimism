/* Internal Imports */
import { add0x, remove0x, toVerifiedBytes, encodeHex, getLen } from '../common'
import { Coder, Signature, Uint16, Uint8, Uint24, Address } from './types'

/***********************
 * TxTypes and TxData  *
 **********************/

export enum TxType {
  EIP155 = 0,
  EthSign = 1,
  EthSign2 = 2,
}

export const txTypePlainText = {
  0: TxType.EIP155,
  1: TxType.EthSign,
  2: TxType.EthSign2,
  EIP155: TxType.EIP155,
  EthSign: TxType.EthSign,
}

export interface DefaultEcdsaTxData {
  sig: Signature
  gasLimit: Uint16
  gasPrice: Uint8
  nonce: Uint24
  target: Address
  data: string
  type: TxType
}

export interface EIP155TxData extends DefaultEcdsaTxData {}
export interface EthSignTxData extends DefaultEcdsaTxData {}

/***********************
 * Encoding Positions  *
 **********************/

/*
 * The positions in the tx data for the different transaction types
 */

export const TX_TYPE_POSITION = { start: 0, end: 1 }

/*
 * The positions in the tx data for the EIP155TxData and EthSignTxData
 */

export const SIGNATURE_FIELD_POSITIONS = {
  r: { start: 1, end: 33 }, // 32 bytes
  s: { start: 33, end: 65 }, // 32 bytes
  v: { start: 65, end: 66 }, // 1 byte
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
export const CTC_TX_GAS_PRICE_MULT_FACTOR = 1_000_000

/***************
 * EcdsaCoders *
 **************/

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
    if (txData.gasPrice % CTC_TX_GAS_PRICE_MULT_FACTOR !== 0) {
      throw new Error(`Gas Price ${txData.gasPrice} cannot be encoded`)
    }
    const gasPrice = encodeHex(
      txData.gasPrice / CTC_TX_GAS_PRICE_MULT_FACTOR,
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
    if (parseInt(sliceBytes(pos.txType), 16) !== this.txType) {
      throw new Error('Invalid tx type')
    }

    return {
      sig: {
        r: add0x(sliceBytes(pos.sig.r)),
        s: add0x(sliceBytes(pos.sig.s)),
        v: parseInt(sliceBytes(pos.sig.v), 16),
      },
      gasLimit: parseInt(sliceBytes(pos.gasLimit), 16),
      gasPrice:
        parseInt(sliceBytes(pos.gasPrice), 16) * CTC_TX_GAS_PRICE_MULT_FACTOR,
      nonce: parseInt(sliceBytes(pos.nonce), 16),
      target: add0x(sliceBytes(pos.target)),
      data: add0x(txData.slice(pos.data.start * 2)),
      type: this.txType,
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

class EthSign2TxCoder extends DefaultEcdsaTxCoder {
  constructor() {
    super(TxType.EthSign2)
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

/*************
 * ctcCoder  *
 ************/

const encode = (data: EIP155TxData): string => {
  if (data.type === TxType.EIP155) {
    return new Eip155TxCoder().encode(data)
  }
  if (data.type === TxType.EthSign) {
    return new EthSignTxCoder().encode(data)
  }
  return null
}

const decode = (data: string | Buffer): EIP155TxData => {
  if (Buffer.isBuffer(data)) {
    data = data.toString()
  }
  data = remove0x(data)
  const type = parseInt(data.slice(0, 2), 16)
  if (type === TxType.EIP155) {
    return new Eip155TxCoder().decode(data)
  }
  if (type === TxType.EthSign) {
    return new EthSignTxCoder().decode(data)
  }
  if (type === TxType.EthSign2) {
    return new EthSign2TxCoder().decode(data)
  }
  return null
}

/*
 * Encoding and decoding functions for all txData types.
 */
export const ctcCoder = {
  eip155TxData: new Eip155TxCoder(),
  ethSignTxData: new EthSignTxCoder(),
  ethSign2TxData: new EthSign2TxCoder(),
  encode,
  decode,
}
