export interface EthereumAccount {
  address: string
  privateKey: string
}

/**
 * Checks whether an object is an EthereumAccount.
 * @param data Object to check.
 * @returns `true` if it's an EthereumAccount, `false` otherwise.
 */
export const isAccount = (data: any): data is EthereumAccount => {
  return data.address !== undefined && data.privateKey !== undefined
}
