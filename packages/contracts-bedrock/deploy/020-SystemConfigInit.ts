import { DeployFunction } from 'hardhat-deploy/dist/types'
import { BigNumber } from 'ethers'
import '@eth-optimism/hardhat-deploy-config'
import 'hardhat-deploy'

import { defaultResourceConfig } from '../src/constants'
import { getContractsFromArtifacts } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const [proxyAdmin, systemConfigProxy, systemConfigImpl] =
    await getContractsFromArtifacts(hre, [
      {
        name: 'ProxyAdmin',
        signerOrProvider: deployer,
      },
      {
        name: 'SystemConfigProxy',
        iface: 'SystemConfig',
        signerOrProvider: deployer,
      },
      {
        name: 'SystemConfig',
      },
    ])

  const batcherHash = hre.ethers.utils
    .hexZeroPad(hre.deployConfig.batchSenderAddress, 32)
    .toLowerCase()

  const l2GenesisBlockGasLimit = BigNumber.from(
    hre.deployConfig.l2GenesisBlockGasLimit
  )
  const l2GasLimitLowerBound = BigNumber.from(
    defaultResourceConfig.systemTxMaxGas +
      defaultResourceConfig.maxResourceLimit
  )
  if (l2GenesisBlockGasLimit.lt(l2GasLimitLowerBound)) {
    throw new Error(
      `L2 genesis block gas limit must be at least ${l2GasLimitLowerBound}`
    )
  }

  try {
    const tx = await proxyAdmin.upgradeAndCall(
      systemConfigProxy.address,
      systemConfigImpl.address,
      systemConfigImpl.interface.encodeFunctionData('initialize', [
        hre.deployConfig.finalSystemOwner,
        hre.deployConfig.gasPriceOracleOverhead,
        hre.deployConfig.gasPriceOracleScalar,
        batcherHash,
        l2GenesisBlockGasLimit,
        hre.deployConfig.p2pSequencerAddress,
        defaultResourceConfig,
      ])
    )
    await tx.wait()
  } catch (e) {
    console.log('SystemConfig already initialized')
    console.log(e)
  }

  const version = await systemConfigProxy.callStatic.version()
  console.log(`SystemConfig version: ${version}`)

  console.log('Upgraded SystemConfig')
}

deployFn.tags = ['SystemConfigInitialize', 'l1']

export default deployFn
