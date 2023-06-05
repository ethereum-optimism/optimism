import assert from 'assert'

import { ethers, BigNumber } from 'ethers'
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { awaitCondition } from '@eth-optimism/core-utils'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'

import {
  getContractsFromArtifacts,
  getDeploymentAddress,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  // Load the contracts we need to interact with.
  const [
    SystemDictator,
    SystemDictatorProxy,
    SystemDictatorProxyWithSigner,
    SystemDictatorImpl,
  ] = await getContractsFromArtifacts(hre, [
    {
      name: 'SystemDictatorProxy',
      iface: 'SystemDictator',
      signerOrProvider: deployer,
    },
    {
      name: 'SystemDictatorProxy',
    },
    {
      name: 'SystemDictatorProxy',
      signerOrProvider: deployer,
    },
    {
      name: 'SystemDictator',
      signerOrProvider: deployer,
    },
  ])

  // Load the dictator configuration.
  const config = {
    globalConfig: {
      proxyAdmin: await getDeploymentAddress(hre, 'ProxyAdmin'),
      controller: hre.deployConfig.controller,
      finalOwner: hre.deployConfig.finalSystemOwner,
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
      owner: hre.deployConfig.finalSystemOwner,
      overhead: hre.deployConfig.gasPriceOracleOverhead,
      scalar: hre.deployConfig.gasPriceOracleScalar,
      batcherHash: hre.ethers.utils.hexZeroPad(
        hre.deployConfig.batchSenderAddress,
        32
      ),
      gasLimit: hre.deployConfig.l2GenesisBlockGasLimit,
      unsafeBlockSigner: hre.deployConfig.p2pSequencerAddress,
      // The resource config is not exposed to the end user
      // to simplify deploy config. It may be introduced in the future.
      resourceConfig: {
        maxResourceLimit: 20_000_000,
        elasticityMultiplier: 10,
        baseFeeMaxChangeDenominator: 8,
        minimumBaseFee: ethers.utils.parseUnits('1', 'gwei'),
        systemTxMaxGas: 1_000_000,
        maximumBaseFee: BigNumber.from(
          '0xffffffffffffffffffffffffffffffff'
        ).toString(),
      },
    },
  }

  // Update the implementation if necessary.
  if (
    (await SystemDictatorProxy.callStatic.implementation({
      from: ethers.constants.AddressZero,
    })) !== SystemDictatorImpl.address
  ) {
    console.log('Upgrading the SystemDictator proxy...')

    // Upgrade and initialize the proxy.
    await SystemDictatorProxyWithSigner.upgradeToAndCall(
      SystemDictatorImpl.address,
      SystemDictatorImpl.interface.encodeFunctionData('initialize', [config])
    )

    // Wait for the transaction to execute properly.
    await awaitCondition(
      async () => {
        return (
          (await SystemDictatorProxy.callStatic.implementation({
            from: ethers.constants.AddressZero,
          })) === SystemDictatorImpl.address
        )
      },
      30000,
      1000
    )

    // Verify that the contract was initialized correctly.
    const dictatorConfig = await SystemDictator.config()
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
  }

  // Update the owner if necessary.
  if (
    (await SystemDictatorProxy.callStatic.admin({
      from: ethers.constants.AddressZero,
    })) !== hre.deployConfig.controller
  ) {
    console.log('Transferring ownership of the SystemDictator proxy...')

    // Transfer ownership to the controller address.
    await SystemDictatorProxyWithSigner.changeAdmin(hre.deployConfig.controller)

    // Wait for the transaction to execute properly.
    await awaitCondition(
      async () => {
        return (
          (await SystemDictatorProxy.callStatic.admin({
            from: ethers.constants.AddressZero,
          })) === hre.deployConfig.controller
        )
      },
      30000,
      1000
    )
  }
}

deployFn.tags = ['SystemDictatorImpl', 'setup', 'l1']

export default deployFn
