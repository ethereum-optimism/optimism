import type { Address, Hex, Log } from 'viem'

export type CrossDomainMessage = {
  source: bigint
  destination: bigint
  nonce: bigint
  sender: Address
  target: Address
  message: Hex

  /** Log that was the source of the message */
  log: Log
}
