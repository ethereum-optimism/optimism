/* External Imports */
import { BigNumber } from 'ethers'

export interface OVMAccount {
  nonce: number | BigNumber
  balance: number | BigNumber
  storageRoot: string
  codeHash: string
  ethAddress: string
}
