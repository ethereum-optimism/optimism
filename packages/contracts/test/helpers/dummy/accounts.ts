/* External Imports */
import { BigNumber, constants } from 'ethers'

/* Internal Imports */
import { DUMMY_BYTES32 } from './bytes32'
import { NON_ZERO_ADDRESS } from '../constants'
import { OVMAccount } from '../types/ovm-types'

export const DUMMY_ACCOUNTS: Array<{
  address: string
  data: OVMAccount
}> = [
  {
    address: '0x1212121212121212121212121212121212121212',
    data: {
      nonce: BigNumber.from(123),
      balance: BigNumber.from(456),
      storageRoot: DUMMY_BYTES32[0],
      codeHash: DUMMY_BYTES32[1],
      ethAddress: constants.AddressZero,
    },
  },
  {
    address: '0x2121212121212121212121212121212121212121',
    data: {
      nonce: BigNumber.from(321),
      balance: BigNumber.from(654),
      storageRoot: DUMMY_BYTES32[2],
      codeHash: DUMMY_BYTES32[3],
      ethAddress: NON_ZERO_ADDRESS,
    },
  },
]
