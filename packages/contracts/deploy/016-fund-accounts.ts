/* Imports: External */
import { sleep } from '@eth-optimism/core-utils'
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import { getContractFromArtifact, isHardhatNode } from '../src/deploy-utils'
import { names } from '../src/address-names'

// This is a TEMPORARY way to fund the default hardhat accounts on L2. The better way to do this is
// to make a modification to hardhat-ovm. However, I don't have the time right now to figure the
// details of how to make that work cleanly. This is fine in the meantime.
const deployFn: DeployFunction = async (hre) => {
  // Only execute this step if we're on the hardhat chain ID.
  if (await isHardhatNode(hre)) {
    const L1StandardBridge = await getContractFromArtifact(
      hre,
      names.managed.contracts.Proxy__OVM_L1StandardBridge,
      {
        iface: 'L1StandardBridge',
      }
    )

    // Default has 20 accounts but we restrict to 20 accounts manually as well just to prevent
    // future problems if the number of default accounts increases for whatever reason.
    const signers = (await hre.ethers.getSigners()).slice(0, 20)

    // Fund the accounts in parallel to speed things up.
    await Promise.all(
      signers.map(async (signer, index) => {
        // Add a sleep here to avoid any potential issues with spamming hardhat. Not sure if this
        // is strictly necessary but it can't hurt.
        await sleep(200 * index)

        const balance = await signer.getBalance()
        const depositAmount = balance.div(2) // Deposit half of the wallet's balance into L2.
        await L1StandardBridge.connect(signer).depositETH(8_000_000, '0x', {
          value: depositAmount,
          gasLimit: 2_000_000, // Idk, gas estimation was broken and this fixes it.
        })

        console.log(
          `✓ Funded ${signer.address} on L2 with ${hre.ethers.utils.formatEther(
            depositAmount
          )} ETH`
        )
      })
    )
  }
}

deployFn.tags = ['fund-accounts']

export default deployFn
