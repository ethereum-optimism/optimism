// import 'ethers'
import { ethers } from 'ethers'

// new function to log an account an it's balance
export const describeFinding = (
  account: string,
  actual: ethers.BigNumber,
  threshold: ethers.BigNumber
) => {
  return `Balance of account ${account} is (${ethers.utils.formatEther(
    actual
  )} eth) below threshold (${ethers.utils.formatEther(threshold)} eth)`
}
