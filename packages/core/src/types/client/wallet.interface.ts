export interface Wallet {
  listAccounts(): Promise<string[]>
  createAccount(password: string): Promise<string>
  unlockAccount(address: string, password: string): Promise<void>
  lockAccount(address: string): Promise<void>
  sign(address: string, message: string): Promise<string>
}
