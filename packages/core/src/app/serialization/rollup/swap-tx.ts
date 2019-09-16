/* Internal Imports */
import { abi } from '../../../app'
import { AbiEncodable } from '../../../types'

/**
 * Creates a SwapTx from an encoded SwapTx.
 * @param encoded The encoded SwapTx.
 * @returns the SwapTx.
 */
const fromEncoded = (encoded: string): AbiSwapTx => {
  const decoded = abi.decode(AbiSwapTx.abiTypes, encoded)
  return new AbiSwapTx(
    decoded[0],
    +decoded[1],
    decoded[2],
    decoded[3],
    decoded[4]
  )
}

/**
 * Represents a basic abi encodable SwapTx
 */
export class AbiSwapTx implements AbiEncodable {
  public static abiTypes = ['address', 'bool', 'uint32', 'uint32', 'uint']

  constructor(
    readonly sender: string,
    readonly tokenType: number,
    readonly inputAmount: number,
    readonly minOutputAmount: number,
    readonly timeout: number
  ) {
    // Attempt to encode to verify input is correct
    this.encoded
  }

  /**
   * @returns the abi encoded SwapTx.
   */
  get encoded(): string {
    // Note that we add an extra set of z
    return abi.encode(AbiSwapTx.abiTypes, [
      this.sender,
      this.tokenType,
      this.inputAmount,
      this.minOutputAmount,
      this.timeout,
    ])
  }

  /**
   * @returns the jsonified AbiSwapTx.
   */
  get jsonified(): any {
    return {
      sender: this.sender,
      tokenType: this.tokenType,
      inputAmount: this.inputAmount,
      minOutputAmount: this.minOutputAmount,
      timeout: this.timeout,
    }
  }

  /**
   * Casts a value to a SwapTx.
   * @param value Thing to cast to a SwapTx.
   * @returns the SwapTx.
   */
  public static from(value: string): AbiSwapTx {
    if (typeof value === 'string') {
      return fromEncoded(value)
    }

    throw new Error('Got invalid argument type when casting to SwapTx.')
  }
}
