import { ethers } from 'ethers'
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'

import {
  getDeploymentAddress,
  deployAndVerifyAndThen,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

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
    } else {
      throw new Error(
        `controller address MUST NOT be the deployer on live networks`
      )
    }
  }

  let finalOwner = hre.deployConfig.finalSystemOwner
  if (finalOwner === ethers.constants.AddressZero) {
    if (hre.network.config.live === false) {
      console.log(`WARNING!!!`)
      console.log(`WARNING!!!`)
      console.log(`WARNING!!!`)
      console.log(`WARNING!!! A proxy admin owner address was not provided.`)
      console.log(
        `WARNING!!! Make sure you are ONLY doing this on a test network.`
      )
      finalOwner = deployer
    } else {
      throw new Error(`must specify the finalSystemOwner on live networks`)
    }
  }

  await deployAndVerifyAndThen({
    hre,
    name: 'MigrationSystemDictator',
    args: [
      {
        globalConfig: {
          proxyAdmin: await getDeploymentAddress(hre, 'ProxyAdmin'),
          controller,
          finalOwner,
          addressManager: await getDeploymentAddress(hre, 'Lib_AddressManager'),
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
          l2OutputOracleImpl: await getDeploymentAddress(hre, 'L2OutputOracle'),
          optimismPortalImpl: await getDeploymentAddress(hre, 'OptimismPortal'),
          l1CrossDomainMessengerImpl: await getDeploymentAddress(
            hre,
            'L1CrossDomainMessenger'
          ),
          l1StandardBridgeImpl: await getDeploymentAddress(
            hre,
            'L1StandardBridge'
          ),
          optimismMintableERC20FactoryImpl: await getDeploymentAddress(
            hre,
            'OptimismMintableERC20Factory'
          ),
          l1ERC721BridgeImpl: await getDeploymentAddress(hre, 'L1ERC721Bridge'),
          portalSenderImpl: await getDeploymentAddress(hre, 'PortalSender'),
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
}

deployFn.tags = ['MigrationSystemDictator', 'migration']

export default deployFn
