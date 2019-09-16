/* Internal Imports */
import { abi } from '../../../app'
import { AbiEncodable } from '../../../types'

/**
 * Creates a TransferNewAccountTx from an encoded TransferNewAccountTx.
 * @param encoded The encoded TransferNewAccountTx.
 * @returns the TransferNewAccountTx.
 */
const fromEncoded = (encoded: string): AbiSwapTransition => {
  const decoded = abi.decode(AbiSwapTransition.abiTypes, encoded)
  return new AbiSwapTransition(
    decoded[0],
    decoded[1],
    decoded[2],
    +decoded[3],
    decoded[4],
    decoded[5],
    decoded[6],
    decoded[7]
  )
}

/**
 * Represents a basic abi encodable TransferNewAccountTx
 */
export class AbiSwapTransition implements AbiEncodable {
  public static abiTypes = [
    'bytes32',
    'uint32',
    'uint32',
    'bool',
    'uint32',
    'uint32',
    'uint',
    'bytes',
  ]

  constructor(
    readonly stateRoot: string,
    readonly senderSlot: number,
    readonly recipientSlot: number,
    readonly tokenType: number,
    readonly inputAmount: number,
    readonly minOutputAmount: number,
    readonly timeout: number,
    readonly signature: string
  ) {
    // Attempt to encode to verify input is correct
    this.encoded
  }

  /**
   * @returns the abi encoded TransferNewAccountTx.
   */
  get encoded(): string {
    return abi.encode(AbiSwapTransition.abiTypes, [
      this.stateRoot,
      this.senderSlot,
      this.recipientSlot,
      this.tokenType,
      this.inputAmount,
      this.minOutputAmount,
      this.timeout,
      this.signature,
    ])
  }

  /**
   * @returns the jsonified AbiSwapTransition.
   */
  get jsonified(): any {
    return {
      stateRoot: this.stateRoot,
      senderSlot: this.senderSlot,
      recipientSlot: this.recipientSlot,
      tokenType: this.tokenType,
      inputAmount: this.inputAmount,
      minOutputAmount: this.minOutputAmount,
      timeout: this.timeout,
      signature: this.signature,
    }
  }

  /**
   * Casts a value to a TransferNewAccountTx.
   * @param value Thing to cast to a TransferNewAccountTx.
   * @returns the TransferNewAccountTx.
   */
  public static from(value: string): AbiSwapTransition {
    if (typeof value === 'string') {
      return fromEncoded(value)
    }

    throw new Error(
      'Got invalid argument type when casting to TransferNewAccountTx.'
    )
  }
}
