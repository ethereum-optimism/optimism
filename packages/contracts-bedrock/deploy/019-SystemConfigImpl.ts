import assert from 'assert'

import { DeployFunction } from 'hardhat-deploy/dist/types'
import { ethers } from 'ethers'
import '@eth-optimism/hardhat-deploy-config'

import { assertContractVariable, deploy } from '../scripts/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const batcherHash = hre.ethers.utils
    .hexZeroPad(ethers.constants.AddressZero, 32)
    .toLowerCase()

  const minimumBaseFee = ethers.utils.parseUnits('1', 'gwei')
  const maximumBaseFee = ethers.BigNumber.from('2').pow(128).sub(1)

  await deploy({
    hre,
    name: 'SystemConfig',
    args: [
      '0x000000000000000000000000000000000000dEaD',
      0,
      0,
      batcherHash,
      20_000_000 + 1_000_000,
      ethers.constants.AddressZero,
      {
        maxResourceLimit: 20_000_000,
        elasticityMultiplier: 10,
        baseFeeMaxChangeDenominator: 8,
        minimumBaseFee,
        systemTxMaxGas: 1_000_000,
        maximumBaseFee: ethers.BigNumber.from('2').pow(128).sub(1),
      },
    ],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'owner',
        '0x000000000000000000000000000000000000dEaD'
      )
      await assertContractVariable(contract, 'overhead', 0)
      await assertContractVariable(contract, 'scalar', 0)
      await assertContractVariable(contract, 'batcherHash', batcherHash)
      await assertContractVariable(
        contract,
        'unsafeBlockSigner',
        ethers.constants.AddressZero
      )

      const config = await contract.resourceConfig()
      assert(config.maxResourceLimit === 20_000_000)
      assert(config.elasticityMultiplier === 10)
      assert(config.baseFeeMaxChangeDenominator === 8)
      assert(config.systemTxMaxGas === 1_000_000)
      assert(config.minimumBaseFee === minimumBaseFee.toNumber())
      assert(config.maximumBaseFee.eq(maximumBaseFee))
    },
  })
}

deployFn.tags = ['SystemConfigImpl', 'setup', 'l1']

export default deployFn
