import { toRpcHexString } from '@eth-optimism/core-utils'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'
import { BigNumber } from 'ethers'
import hre from 'hardhat'

export const impersonate = async (
  address: string,
  balance?: string | number | BigNumber
): Promise<SignerWithAddress> => {
  await hre.network.provider.request({
    method: 'hardhat_impersonateAccount',
    params: [address],
  })

  if (balance !== undefined) {
    await hre.network.provider.request({
      method: 'hardhat_setBalance',
      params: [address, toRpcHexString(BigNumber.from(balance))],
    })
  }

  return hre.ethers.getSigner(address)
}
