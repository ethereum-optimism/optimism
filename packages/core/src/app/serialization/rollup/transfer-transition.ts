/* Internal Imports */
import { abi } from '../../../app'
import { AbiEncodable } from '../../../types'

/**
 * Creates a TransferStoredAccount from an encoded TransferStoredAccount.
 * @param encoded The encoded TransferStoredAccount.
 * @returns the TransferStoredAccount.
 */
const fromEncoded = (encoded: string): AbiTransferTransition => {
  const [
    stateRoot,
    senderSlot,
    recipientSlot,
    tokenType,
    amount,
    signature,
  ] = abi.decode(AbiTransferTransition.abiTypes, encoded)
  return new AbiTransferTransition(
    stateRoot,
    senderSlot,
    recipientSlot,
    +tokenType,
    amount,
    signature
  )
}

/**
 * Represents a basic abi encodable TransferStoredAccount
 */
export class AbiTransferTransition implements AbiEncodable {
  public static abiTypes = [
    'bytes32',
    'uint32',
    'uint32',
    'bool',
    'uint32',
    'bytes',
  ]

  constructor(
    readonly stateRoot: string,
    readonly senderSlot: number,
    readonly recipientSlot: number,
    readonly tokenType: number,
    readonly amount: number,
    readonly signature: string
  ) {
    // Attempt to encode to verify input is correct
    this.encoded
  }

  /**
   * @returns the abi encoded TransferStoredAccount.
   */
  get encoded(): string {
    return abi.encode(AbiTransferTransition.abiTypes, [
      this.stateRoot,
      this.senderSlot,
      this.recipientSlot,
      this.tokenType,
      this.amount,
      this.signature,
    ])
  }

  /**
   * @returns the jsonified AbiTransferTransition.
   */
  get jsonified(): any {
    return {
      stateRoot: this.stateRoot,
      senderSlot: this.senderSlot,
      recipientSlot: this.recipientSlot,
      tokenType: this.tokenType,
      amount: this.amount,
      signature: this.signature,
    }
  }

  /**
   * Casts a value to a TransferStoredAccount.
   * @param value Thing to cast to a TransferStoredAccount.
   * @returns the TransferStoredAccount.
   */
  public static from(value: string): AbiTransferTransition {
    if (typeof value === 'string') {
      return fromEncoded(value)
    }

    throw new Error(
      'Got invalid argument type when casting to TransferStoredAccount.'
    )
  }
}
