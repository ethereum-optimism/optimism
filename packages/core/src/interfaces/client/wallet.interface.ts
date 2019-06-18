export interface Wallet {
  listAccounts(): Promise<string[]>
  createAccount(password: string): Promise<void>
  unlockAccount(address: string, password: string): Promise<void>
  lockAccount(address: string): Promise<void>
  sign(address: string, message: string): Promise<string>
}
