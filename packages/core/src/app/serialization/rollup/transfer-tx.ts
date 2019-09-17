/* Internal Imports */
import { abi } from '../../../app'
import { AbiEncodable } from '../../../types'

/**
 * Creates a TransferTx from an encoded TransferTx.
 * @param encoded The encoded TransferTx.
 * @returns the TransferTx.
 */
const fromEncoded = (encoded: string): AbiTransferTx => {
  const [sender, recipient, tokenType, amount] = abi.decode(
    AbiTransferTx.abiTypes,
    encoded
  )
  return new AbiTransferTx(sender, recipient, +tokenType, amount)
}

/**
 * Represents a basic abi encodable TransferTx
 */
export class AbiTransferTx implements AbiEncodable {
  public static abiTypes = ['address', 'address', 'bool', 'uint32']

  constructor(
    readonly sender: string,
    readonly recipient: string,
    readonly tokenType: number,
    readonly amount: number
  ) {
    // Attempt to encode to verify input is correct
    this.encoded
  }

  /**
   * @returns the abi encoded TransferTx.
   */
  get encoded(): string {
    return abi.encode(AbiTransferTx.abiTypes, [
      this.sender,
      this.recipient,
      this.tokenType,
      this.amount,
    ])
  }

  /**
   * @returns the jsonified AbiTransferTx.
   */
  get jsonified(): any {
    return {
      sender: this.sender,
      recipient: this.recipient,
      tokenType: this.tokenType,
      amount: this.amount,
    }
  }

  /**
   * Casts a value to a TransferTx.
   * @param value Thing to cast to a TransferTx.
   * @returns the TransferTx.
   */
  public static from(value: string): AbiTransferTx {
    if (typeof value === 'string') {
      return fromEncoded(value)
    }

    throw new Error('Got invalid argument type when casting to TransferTx.')
  }
}
