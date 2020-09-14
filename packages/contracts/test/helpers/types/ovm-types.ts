export interface OVMAccount {
  nonce: number
  balance: number
  storageRoot: string
  codeHash: string
  ethAddress: string
}

export const toOVMAccount = (result: any[]): OVMAccount => {
  return {
    nonce: result[0].toNumber(),
    balance: result[1].toNumber(),
    storageRoot: result[2],
    codeHash: result[3],
    ethAddress: result[4],
  }
}
