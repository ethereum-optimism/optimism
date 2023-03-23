import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import { ethers } from 'ethers'

import { assertContractVariable, deploy } from '../src/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const batcherHash = hre.ethers.utils
    .hexZeroPad(hre.deployConfig.batchSenderAddress, 32)
    .toLowerCase()

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
      {
        maxResourceLimit: 20_000_000,
        elasticityMultiplier: 10,
        baseFeeMaxChangeDenominator: 8,
        systemTxMaxGas: 1_000_000,
        minimumBaseFee: ethers.utils.parseUnits('1', 'gwei'),
        maximumBaseFee: ethers.BigNumber.from(
          '0xffffffffffffffffffffffffffffffff'
        ),
      },
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
