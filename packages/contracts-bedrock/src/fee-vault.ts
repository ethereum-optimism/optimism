import { ethers } from 'ethers'

/**
 * Returns if we should withdraw from a given fee vault.
 * FeeVault is a parent contract, exposing a common interface across fee vaults.
 * See: https://github.com/ethereum-optimism/optimism/blob/develop/packages/contracts-bedrock/contracts/universal/FeeVault.sol
 *
 * @param address The address of the fee vault.
 * @returns If we should withdraw from the fee vault.
 */
export const withdrawFeeVault = async (
  address: string,
  signer: ethers.Signer,
  provider: ethers.providers.Provider
): Promise<boolean> => {
  // Construct Fee Vault Contract
  const feeVault = new ethers.Contract(
    address,
    new ethers.utils.Interface([
      'function withdraw()',
      'function RECIPIENT() view returns address',
      'function MIN_WITHDRAWAL_AMOUNT() view returns uint256',
    ]),
    signer
  )

  // Print fee vault info
  const feeRecipient = await feeVault.RECIPIENT()
  const amount = await provider.getBalance(feeVault.address)
  const amountInETH = ethers.utils.formatEther(amount)
  console.log(
    `Vault [${address}] has ${amountInETH} ETH to be extracted to recipient ${feeRecipient}`
  )

  // If there is enough eth in the fee vault, withdraw it
  const activationThreshold = await feeVault.MIN_WITHDRAWAL_AMOUNT()

  console.log(
    `Activation threshold: ${ethers.utils.formatEther(activationThreshold)}`
  )

  return amount.gt(activationThreshold)
}
