import { BigNumber } from 'ethers'

export const defaultTransactionFactory = () => {
  return {
    to: '0x' + '1234'.repeat(10),
    gasLimit: 8_000_000,
    data: '0x',
    value: BigNumber.from(0),
  }
}
