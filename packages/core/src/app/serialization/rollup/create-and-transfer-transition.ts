/* Internal Imports */
import { abi } from '../../../app'
import { AbiEncodable } from '../../../types'

/**
 * Creates a TransferNewAccountTx from an encoded TransferNewAccountTx.
 * @param encoded The encoded TransferNewAccountTx.
 * @returns the TransferNewAccountTx.
 */
const fromEncoded = (encoded: string): AbiCreateAndTransferTransition => {
  const [
    stateRoot,
    senderSlot,
    recipientSlot,
    createdAccountPubkey,
    tokenTypeBool,
    amount,
    signature,
  ] = abi.decode(AbiCreateAndTransferTransition.abiTypes, encoded)
  return new AbiCreateAndTransferTransition(
    stateRoot,
    senderSlot,
    recipientSlot,
    createdAccountPubkey,
    +tokenTypeBool,
    amount,
    signature
  )
}

/**
 * Represents a basic abi encodable TransferNewAccountTx
 */
export class AbiCreateAndTransferTransition implements AbiEncodable {
  public static abiTypes = [
    'bytes32',
    'uint32',
    'uint32',
    'address',
    'bool',
    'uint32',
    'bytes',
  ]

  constructor(
    readonly stateRoot: string,
    readonly senderSlot: number,
    readonly recipientSlot: number,
    readonly createdAccountPubkey: string,
    readonly tokenType: number,
    readonly amount: number,
    readonly signature: string
  ) {
    // Attempt to encode to verify input is correct
    this.encoded
  }

  /**
   * @returns the abi encoded TransferNewAccountTx.
   */
  get encoded(): string {
    return abi.encode(AbiCreateAndTransferTransition.abiTypes, [
      this.stateRoot,
      this.senderSlot,
      this.recipientSlot,
      this.createdAccountPubkey,
      this.tokenType,
      this.amount,
      this.signature,
    ])
  }

  /**
   * @returns the jsonified AbiCreateAndTransferTransition.
   */
  get jsonified(): any {
    return {
      stateRoot: this.stateRoot,
      senderSlot: this.senderSlot,
      recipientSlot: this.recipientSlot,
      createdAccountPubkey: this.createdAccountPubkey,
      tokenType: this.tokenType,
      amount: this.amount,
      signature: this.signature,
    }
  }

  /**
   * Casts a value to a TransferNewAccountTx.
   * @param value Thing to cast to a TransferNewAccountTx.
   * @returns the TransferNewAccountTx.
   */
  public static from(value: string): AbiCreateAndTransferTransition {
    if (typeof value === 'string') {
      return fromEncoded(value)
    }

    throw new Error(
      'Got invalid argument type when casting to TransferNewAccountTx.'
    )
  }
}
