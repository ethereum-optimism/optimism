import assert from 'assert'

import { DeployFunction } from 'hardhat-deploy/dist/types'
import { ethers } from 'ethers'
import '@eth-optimism/hardhat-deploy-config'

import { assertContractVariable, deploy } from '../scripts/deploy-utils'

const deployFn: DeployFunction = async (hre) => {
  const batcherHash = hre.ethers.utils
    .hexZeroPad(ethers.constants.AddressZero, 32)
    .toLowerCase()

  await deploy({
    hre,
    name: 'SystemConfig',
    args: [],
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
      assert(config.maxResourceLimit === 1)
      assert(config.elasticityMultiplier === 1)
      assert(config.baseFeeMaxChangeDenominator === 2)
      assert(config.systemTxMaxGas === 0)
      assert(ethers.utils.parseUnits('0', 'gwei').eq(config.minimumBaseFee))
      assert(config.maximumBaseFee.eq(ethers.BigNumber.from('0')))
    },
  })
}

deployFn.tags = ['SystemConfigImpl', 'setup', 'l1']

export default deployFn
