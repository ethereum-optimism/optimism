/* Internal Imports */
import { Keystore } from '../common'

export interface WalletDB {
  putKeystore(keystore: Keystore): Promise<void>
  getKeystore(address: string): Promise<Keystore>
  hasKeystore(address: string): Promise<boolean>
  listAddresses(): Promise<string[]>
}
