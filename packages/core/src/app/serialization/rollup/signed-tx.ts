/* Internal Imports */
import { abi } from '../../../app'
import { AbiEncodable } from '../../../types'
import { AbiTransferTx, AbiSwapTx } from '.'

/**
 * Creates a SignedTx from an encoded SignedTx.
 * @param encoded The encoded SignedTx.
 * @returns the SignedTx.
 */
const fromEncoded = (encoded: string): AbiSignedTx => {
  const decoded = abi.decode(AbiSignedTx.abiTypes, encoded)
  // Check to see if the tx is a transfer
  try {
    const transferTx = AbiTransferTx.from(decoded[1])
    return new AbiSignedTx(decoded[0], transferTx)
  } catch (err) {
    // If it's not a transfer, it must be a swap
    const swapTx = AbiSwapTx.from(decoded[1])
    return new AbiSignedTx(decoded[0], swapTx)
  }
}

/**
 * Represents a basic abi encodable SignedTx
 */
export class AbiSignedTx implements AbiEncodable {
  public static abiTypes = ['bytes', 'bytes']

  constructor(
    readonly signature: string,
    readonly tx: AbiTransferTx | AbiSwapTx
  ) {
    // Attempt to encode to verify input is correct
    this.encoded
    // TODO: Verify the signature against the tx
  }

  /**
   * @returns the abi encoded SignedTx.
   */
  get encoded(): string {
    return abi.encode(AbiSignedTx.abiTypes, [this.signature, this.tx.encoded])
  }

  /**
   * @returns the jsonified AbiSignedTx.
   */
  get jsonified(): any {
    return {
      signature: this.signature,
      tx: this.tx,
    }
  }

  /**
   * Casts a value to a SignedTx.
   * @param value Thing to cast to a SignedTx.
   * @returns the SignedTx.
   */
  public static from(value: string): AbiSignedTx {
    if (typeof value === 'string') {
      return fromEncoded(value)
    }

    throw new Error('Got invalid argument type when casting to SignedTx.')
  }
}
