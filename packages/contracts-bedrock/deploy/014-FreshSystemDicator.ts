import { ethers } from 'ethers'
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'

import {
  getDeploymentAddress,
  deployAndVerifyAndThen,
  getContractFromArtifact,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  await deployAndVerifyAndThen({
    hre,
    name: 'FreshSystemDictator',
    args: [
      {
        globalConfig: {
          proxyAdmin: await getDeploymentAddress(hre, 'ProxyAdmin'),
          controller: deployer, // TODO
          finalOwner: hre.deployConfig.proxyAdminOwner,
          addressManager: ethers.constants.AddressZero,
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
            'L1CrossDomainMessengerProxy'
          ),
          l1StandardBridgeProxy: await getDeploymentAddress(
            hre,
            'L1StandardBridgeProxy'
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

  const ProxyAdmin = await getContractFromArtifact(hre, 'ProxyAdmin', {
    signerOrProvider: deployer,
  })
  const FreshSystemDictator = await getContractFromArtifact(
    hre,
    'FreshSystemDictator',
    {
      signerOrProvider: deployer,
    }
  )

  await ProxyAdmin.setOwner(FreshSystemDictator.address)
  await FreshSystemDictator.step1()
  await FreshSystemDictator.step2()
}

deployFn.tags = ['FreshSystemDictator', 'fresh']

export default deployFn
