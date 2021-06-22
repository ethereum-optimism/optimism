/* Imports: External */
import { sleep } from '@eth-optimism/core-utils'
import { DeployFunction } from 'hardhat-deploy/dist/types'
import {
  defaultHardhatNetworkHdAccountsConfigParams,
  defaultHardhatNetworkParams,
} from 'hardhat/internal/core/config/default-config'
import { normalizeHardhatNetworkAccountsConfig } from 'hardhat/internal/core/providers/util'

/* Imports: Internal */
import { getDeployedContract } from '../src/hardhat-deploy-ethers'

// This is a TEMPORARY way to fund the default hardhat accounts on L2. The better way to do this is
// to make a modification to hardhat-ovm. However, I don't have the time right now to figure the
// details of how to make that work cleanly. This is fine in the meantime.
const deployFn: DeployFunction = async (hre) => {
  // Only execute this step if we're on the hardhat chain ID.
  const { chainId } = await hre.ethers.provider.getNetwork()
  if (chainId === defaultHardhatNetworkParams.chainId) {
    const OVM_L1StandardBridge = await getDeployedContract(
      hre,
      'Proxy__OVM_L1StandardBridge',
      {
        iface: 'OVM_L1StandardBridge',
      }
    )

    // Default has 20 accounts but we restrict to 20 accounts manually as well just to prevent
    // future problems if the number of default accounts increases for whatever reason.
    const accounts = normalizeHardhatNetworkAccountsConfig(
      defaultHardhatNetworkHdAccountsConfigParams
    ).slice(0, 20)

    // Fund the accounts in parallel to speed things up.
    await Promise.all(
      accounts.map(async (account, index) => {
        // Add a sleep here to avoid any potential issues with spamming hardhat. Not sure if this
        // is strictly necessary but it can't hurt.
        await sleep(200 * index)

        const wallet = new hre.ethers.Wallet(
          account.privateKey,
          hre.ethers.provider
        )
        const balance = await wallet.getBalance()
        const depositAmount = balance.div(2) // Deposit half of the wallet's balance into L2.
        await OVM_L1StandardBridge.connect(wallet).depositETH(8_000_000, '0x', {
          value: depositAmount,
          gasLimit: 2_000_000, // Idk, gas estimation was broken and this fixes it.
        })
        console.log(
          `✓ Funded ${wallet.address} on L2 with ${hre.ethers.utils.formatEther(
            depositAmount
          )} ETH`
        )
      })
    )
  }
}

deployFn.dependencies = ['Proxy__OVM_L1ETHGateway']
deployFn.tags = ['fund-accounts']

export default deployFn
