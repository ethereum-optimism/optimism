import { awaitCondition } from '@eth-optimism/core-utils'
import { ethers } from 'ethers'
import { DeployFunction } from 'hardhat-deploy/dist/types'

import {
  getDeploymentAddress,
  deployAndVerifyAndThen,
  getContractFromArtifact,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  let isLiveDeployer = false
  let controller = hre.deployConfig.controller
  if (controller === ethers.constants.AddressZero) {
    if (hre.network.config.live === false) {
      console.log(`WARNING!!!`)
      console.log(`WARNING!!!`)
      console.log(`WARNING!!!`)
      console.log(`WARNING!!! A controller address was not provided.`)
      console.log(
        `WARNING!!! Make sure you are ONLY doing this on a test network.`
      )
      controller = deployer
      isLiveDeployer = true
    } else {
      throw new Error(
        `controller address MUST NOT be the deployer on live networks`
      )
    }
  }

  await deployAndVerifyAndThen({
    hre,
    name: 'MigrationSystemDictator',
    args: [
      {
        globalConfig: {
          proxyAdmin: await getDeploymentAddress(hre, 'ProxyAdmin'),
          controller: deployer, // TODO
          finalOwner: hre.deployConfig.proxyAdminOwner,
          addressManager: hre.deployConfig.addressManager,
        },
        proxyAddressConfig: {
          l2OutputOracleProxy: await getDeploymentAddress(
            hre,
            'L2OutputOracleProxy'
          ),
          optimismPortalProxy: await getDeploymentAddress(
            hre,
            'OptimismPortalProxy'
          ),
          l1CrossDomainMessengerProxy: await getDeploymentAddress(
            hre,
            'Proxy__OVM_L1CrossDomainMessenger'
          ),
          l1StandardBridgeProxy: await getDeploymentAddress(
            hre,
            'Proxy__OVM_L1StandardBridge'
          ),
          optimismMintableERC20FactoryProxy: await getDeploymentAddress(
            hre,
            'OptimismMintableERC20FactoryProxy'
          ),
          l1ERC721BridgeProxy: await getDeploymentAddress(
            hre,
            'L1ERC721BridgeProxy'
          ),
        },
        implementationAddressConfig: {
          l2OutputOracleImpl: await getDeploymentAddress(
            hre,
            'L2OutputOracleImpl'
          ),
          optimismPortalImpl: await getDeploymentAddress(
            hre,
            'OptimismPortalImpl'
          ),
          l1CrossDomainMessengerImpl: await getDeploymentAddress(
            hre,
            'L1CrossDomainMessengerImpl'
          ),
          l1StandardBridgeImpl: await getDeploymentAddress(
            hre,
            'L1StandardBridgeImpl'
          ),
          optimismMintableERC20FactoryImpl: await getDeploymentAddress(
            hre,
            'OptimismMintableERC20FactoryImpl'
          ),
          l1ERC721BridgeImpl: await getDeploymentAddress(
            hre,
            'L1ERC721BridgeImpl'
          ),
          portalSenderImpl: await getDeploymentAddress(hre, 'PortalSenderImpl'),
        },
        l2OutputOracleConfig: {
          l2OutputOracleGenesisL2Output:
            hre.deployConfig.l2OutputOracleGenesisL2Output,
          l2OutputOracleProposer: hre.deployConfig.l2OutputOracleProposer,
          l2OutputOracleOwner: hre.deployConfig.l2OutputOracleOwner,
        },
      },
    ],
    postDeployAction: async () => {
      // TODO: Assert all the config was set correctly.
    },
  })

  const ProxyAdmin = await getContractFromArtifact(hre, 'ProxyAdmin', {
    signerOrProvider: deployer,
  })
  const MigrationSystemDictator = await getContractFromArtifact(
    hre,
    'MigrationSystemDictator',
    {
      signerOrProvider: deployer,
    }
  )

  console.log(
    `Transferring ownership of ProxyAdmin to MigrationSystemDictator...`
  )
  await ProxyAdmin.setOwner(MigrationSystemDictator.address)

  // Transfer ownership of the AddressManager to MigrationSystemDictator.
  const AddressManager = await getContractFromArtifact(hre, 'AddressManager', {
    signerOrProvider: deployer,
  })
  if (isLiveDeployer) {
    console.log(
      `Transferring ownership of AddressManager to the MigrationSystemDictator...`
    )
    await AddressManager.transferOwnership(MigrationSystemDictator.address)
  } else {
    console.log(
      `Please transfer ownership of the AddressManager to the MigrationSystemDictator located at: ${MigrationSystemDictator.address}`
    )
  }
  await awaitCondition(async () => {
    const owner = await AddressManager.owner()
    return owner === MigrationSystemDictator.address
  })

  // Transfer ownership of the L1CrossDomainMessenger to MigrationSystemDictator.
  const L1CrossDomainMessenger = await getContractFromArtifact(
    hre,
    'Proxy__OVM_L1CrossDomainMessenger',
    {
      iface: 'L1CrossDomainMessenger',
      signerOrProvider: deployer,
    }
  )
  if (isLiveDeployer) {
    console.log(
      `Transferring ownership of L1CrossDomainMessenger to the MigrationSystemDictator...`
    )
    await L1CrossDomainMessenger.transferOwnership(
      MigrationSystemDictator.address
    )
  } else {
    console.log(
      `Please transfer ownership of the L1CrossDomainMessenger to the MigrationSystemDictator located at: ${MigrationSystemDictator.address}`
    )
  }
  await awaitCondition(async () => {
    const owner = await L1CrossDomainMessenger.owner()
    return owner === MigrationSystemDictator.address
  })

  // Transfer ownership of the L1StandardBridge (proxy) to MigrationSystemDictator.
  const L1StandardBridge = await getContractFromArtifact(
    hre,
    'Proxy__OVM_L1StandardBridge',
    {
      signerOrProvider: deployer,
    }
  )
  if (isLiveDeployer) {
    console.log(
      `Transferring ownership of L1StandardBridge to the MigrationSystemDictator...`
    )
    await L1StandardBridge.setOwner(MigrationSystemDictator.address)
  } else {
    console.log(
      `Please transfer ownership of the L1StandardBridge (proxy) to the MigrationSystemDictator located at: ${MigrationSystemDictator.address}`
    )
  }
  await awaitCondition(async () => {
    const owner = await L1StandardBridge.owner()
    return owner === MigrationSystemDictator.address
  })

  for (let i = 1; i <= 6; i++) {
    if (isLiveDeployer) {
      console.log(`Executing step ${i}...`)
      await MigrationSystemDictator[`step${i}`]()
    } else {
      console.log(`Please execute step ${i}...`)
      await awaitCondition(async () => {
        const step = await MigrationSystemDictator.step()
        return step.toNumber() === i
      })
    }
  }
}

deployFn.tags = ['MigrationSystemDictator', 'migration']

export default deployFn
