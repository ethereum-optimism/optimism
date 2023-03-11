import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'

import { assertContractVariable, deploy } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const batcherHash = hre.ethers.utils
    .hexZeroPad(hre.deployConfig.batchSenderAddress, 32)
    .toLowerCase()

  const Artifact__OptimismPortalProxy = await hre.deployments.get(
    'OptimismPortalProxy'
  )

  await deploy({
    hre,
    name: 'SystemConfig',
    args: [
      hre.deployConfig.finalSystemOwner,
      hre.deployConfig.gasPriceOracleOverhead,
      hre.deployConfig.gasPriceOracleScalar,
      batcherHash,
      hre.deployConfig.l2GenesisBlockGasLimit,
      hre.deployConfig.p2pSequencerAddress,
      Artifact__OptimismPortalProxy.address,
    ],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'owner',
        hre.deployConfig.finalSystemOwner
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
      await assertContractVariable(contract, 'batcherHash', batcherHash)
      await assertContractVariable(
        contract,
        'unsafeBlockSigner',
        hre.deployConfig.p2pSequencerAddress
      )
    },
  })
}

deployFn.tags = ['SystemConfigImpl', 'setup']

export default deployFn
