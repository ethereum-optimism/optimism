import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'

import {
  assertContractVariable,
  deployAndVerifyAndThen,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const batcherHash = hre.ethers.utils.hexZeroPad(
    hre.deployConfig.batchSenderAddress,
    32
  )

  await deployAndVerifyAndThen({
    hre,
    name: 'SystemConfig',
    args: [
      hre.deployConfig.systemConfigOwner,
      hre.deployConfig.gasPriceOracleOverhead,
      hre.deployConfig.gasPriceOracleScalar,
      batcherHash,
      hre.deployConfig.l2GenesisBlockGasLimit,
    ],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'owner',
        hre.deployConfig.systemConfigOwner
      )
      await assertContractVariable(
        contract,
        'overhead',
        hre.deployConfig.gasPriceOracleOverhead
      )
      await assertContractVariable(
        contract,
        'scalar',
        hre.deployConfig.gasPriceOracleScalar
      )
      await assertContractVariable(
        contract,
        'batcherHash',
        batcherHash.toLowerCase()
      )
    },
  })
}

deployFn.tags = ['SystemConfigImpl']

export default deployFn
