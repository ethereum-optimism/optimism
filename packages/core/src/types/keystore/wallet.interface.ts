import { SignatureProvider } from './signatures.interface'

export interface Wallet extends SignatureProvider {
  listAccounts(): Promise<string[]>
  createAccount(password: string): Promise<string>
  unlockAccount(address: string, password: string): Promise<void>
  lockAccount(address: string): Promise<void>
}
