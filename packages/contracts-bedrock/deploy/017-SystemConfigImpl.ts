import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'

import {
  assertContractVariable,
  deploy,
  getContractsFromArtifacts,
} from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  const batcherHash = hre.ethers.utils
    .hexZeroPad(hre.deployConfig.batchSenderAddress, 32)
    .toLowerCase()

  const Artifact__OptimismPortalProxy = await hre.deployments.get(
    'OptimismPortalProxy'
  )

  const [OptimismPortal] = await getContractsFromArtifacts(hre, [
    {
      name: 'OptimismPortal',
      signerOrProvider: deployer,
    },
  ])

  const MAX_RESOURCE_LIMIT = await OptimismPortal.MAX_RESOURCE_LIMIT()
  const minGasLimit = MAX_RESOURCE_LIMIT.add(1_000_000)
  if (minGasLimit.lt(hre.deployConfig.l2GenesisBlockGasLimit)) {
    throw new Error(`Initial L2 gas limit is too low`)
  }

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
      await assertContractVariable(
        contract,
        'gasLimit',
        hre.deployConfig.l2GenesisBlockGasLimit
      )
      await assertContractVariable(contract, 'batcherHash', batcherHash)
      await assertContractVariable(
        contract,
        'unsafeBlockSigner',
        hre.deployConfig.p2pSequencerAddress
      )
      await assertContractVariable(
        contract,
        'PORTAL',
        Artifact__OptimismPortalProxy.address
      )
    },
  })
}

deployFn.tags = ['SystemConfigImpl', 'setup']

export default deployFn
