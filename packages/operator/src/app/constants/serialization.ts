export const ADDRESS_BYTE_SIZE = 20
export const START_BYTE_SIZE = 12
export const TYPE_BYTE_SIZE = 4
export const COIN_ID_BYTE_SIZE = START_BYTE_SIZE + TYPE_BYTE_SIZE
export const BLOCKNUMBER_BYTE_SIZE = 4
export const DEPOSIT_SENDER = '0x0000000000000000000000000000000000000000'
// For now, include a export constant which defines the total size of a transaction
export const TRANSFER_BYTE_SIZE =
  ADDRESS_BYTE_SIZE * 2 + TYPE_BYTE_SIZE + START_BYTE_SIZE * 2
export const SIGNATURE_BYTE_SIZE = 1 + 32 * 2
