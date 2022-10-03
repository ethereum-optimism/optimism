/* Imports: External */
import { sleep } from '@eth-optimism/core-utils'
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { defaultHardhatNetworkHdAccountsConfigParams } from 'hardhat/internal/core/config/default-config'
import { normalizeHardhatNetworkAccountsConfig } from 'hardhat/internal/core/providers/util'

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
    const accounts = normalizeHardhatNetworkAccountsConfig(
      defaultHardhatNetworkHdAccountsConfigParams
    ).slice(0, 20)

    // Set balances on L1 for the predeploy deployers

    // 0xa961b0d6dce82db098cf70a42a14add3ee3db2d5
    await hre.network.provider.send('hardhat_setBalance', [
      '0xa961b0d6dce82db098cf70a42a14add3ee3db2d5',
      '0x8000000000000000000000',
    ])
    accounts.push({
      privateKey:
        '0xa6aecc98b63bafb0de3b29ae9964b14acb4086057808be29f90150214ebd4a0f',
      balance: '',
    })

    // 0xdfc82d475833a50de90c642770f34a9db7deb725
    await hre.network.provider.send('hardhat_setBalance', [
      '0xdfc82d475833a50de90c642770f34a9db7deb725',
      '0x8000000000000000000000',
    ])
    accounts.push({
      privateKey:
        '0x3b8d2345102cce2443acb240db6e87c8edd4bb3f821b17fab8ea2c9da08ea132',
      balance: '',
    })

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
        await L1StandardBridge.connect(wallet).depositETH(8_000_000, '0x', {
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

deployFn.tags = ['fund-accounts']

export default deployFn
