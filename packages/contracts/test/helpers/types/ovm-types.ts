export interface OVMAccount {
  nonce: number
  balance: number
  storageRoot: string
  codeHash: string
  ethAddress: string
}

/**
 * Converts a raw ethers result to an OVM account.
 * @param result Raw ethers transaction result.
 * @returns Converted OVM account.
 */
export const toOVMAccount = (result: any[]): OVMAccount => {
  return {
    nonce: result[0].toNumber(),
    balance: result[1].toNumber(),
    storageRoot: result[2],
    codeHash: result[3],
    ethAddress: result[4],
  }
}
