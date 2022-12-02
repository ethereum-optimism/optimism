import assert from 'assert'

import { ethers } from 'ethers'
import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'

import {
  deployAndVerifyAndThen,
  getCriticalDeployConfig,
  getDeploymentAddress,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { controller, finalOwner } = await getCriticalDeployConfig(hre)

  // Put together the SystemDictator config. Requires loading a bunch of addresses so it's
  // relatively ugly, sorry!
  const config = {
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
      systemConfigProxy: await getDeploymentAddress(hre, 'SystemConfigProxy'),
    },
    implementationAddressConfig: {
      l2OutputOracleImpl: await getDeploymentAddress(hre, 'L2OutputOracle'),
      optimismPortalImpl: await getDeploymentAddress(hre, 'OptimismPortal'),
      l1CrossDomainMessengerImpl: await getDeploymentAddress(
        hre,
        'L1CrossDomainMessenger'
      ),
      l1StandardBridgeImpl: await getDeploymentAddress(hre, 'L1StandardBridge'),
      optimismMintableERC20FactoryImpl: await getDeploymentAddress(
        hre,
        'OptimismMintableERC20Factory'
      ),
      l1ERC721BridgeImpl: await getDeploymentAddress(hre, 'L1ERC721Bridge'),
      portalSenderImpl: await getDeploymentAddress(hre, 'PortalSender'),
      systemConfigImpl: await getDeploymentAddress(hre, 'SystemConfig'),
    },
    systemConfigConfig: {
      owner: hre.deployConfig.systemConfigOwner,
      overhead: hre.deployConfig.gasPriceOracleOverhead,
      scalar: hre.deployConfig.gasPriceOracleDecimals,
      batcherHash: hre.ethers.utils.hexZeroPad(
        hre.deployConfig.batchSenderAddress,
        32
      ),
      gasLimit: hre.deployConfig.l2GenesisBlockGasLimit,
    },
  }

  await deployAndVerifyAndThen({
    hre,
    name: 'SystemDictator',
    args: [config],
    postDeployAction: async (contract) => {
      const dictatorConfig = await contract.config()
      for (const [outerConfigKey, outerConfigValue] of Object.entries(config)) {
        for (const [innerConfigKey, innerConfigValue] of Object.entries(
          outerConfigValue
        )) {
          let have = dictatorConfig[outerConfigKey][innerConfigKey]
          let want = innerConfigValue as any

          if (ethers.utils.isAddress(want)) {
            want = want.toLowerCase()
            have = have.toLowerCase()
          } else if (typeof want === 'number') {
            want = ethers.BigNumber.from(want)
            have = ethers.BigNumber.from(have)
            assert(
              want.eq(have),
              `incorrect config for ${outerConfigKey}.${innerConfigKey}. Want: ${want}, have: ${have}`
            )
            return
          }

          assert(
            want === have,
            `incorrect config for ${outerConfigKey}.${innerConfigKey}. Want: ${want}, have: ${have}`
          )
        }
      }
    },
  })
}

deployFn.tags = ['SystemDictator']

export default deployFn
