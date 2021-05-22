/* External Imports */
import { Signer } from 'ethers'
import { toHexString } from '@eth-optimism/core-utils'

export const deployContractCode = async (
  code: string,
  signer: Signer,
  gasLimit: number
): Promise<string> => {
  // "Magic" prefix to be prepended to the contract code. Contains a series of opcodes that will
  // copy the given code into memory and return it, thereby storing at the contract address.
  const prefix = '0x600D380380600D6000396000f3'
  const deployCode = prefix + toHexString(code).slice(2)

  const response = await signer.sendTransaction({
    to: null,
    data: deployCode,
    gasLimit,
  })

  const result = await response.wait()
  return result.contractAddress
}
