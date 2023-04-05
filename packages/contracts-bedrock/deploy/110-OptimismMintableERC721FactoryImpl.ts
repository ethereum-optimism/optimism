import { DeployFunction } from 'hardhat-deploy/dist/types'
import { ethers } from 'ethers'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'

import { assertContractVariable, deploy } from '../src/deploy-utils'
import { predeploys } from '../src/constants'

const deployFn: DeployFunction = async (hre) => {
  const OptimismMintableERC721Factory = await hre.ethers.getContractAt(
    'OptimismMintableERC721Factory',
    predeploys.OptimismMintableERC721Factory
  )
  const remoteChainId = await OptimismMintableERC721Factory.REMOTE_CHAIN_ID()

  await deploy({
    hre,
    name: 'OptimismMintableERC721Factory',
    args: [predeploys.L2StandardBridge, remoteChainId],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'BRIDGE',
        ethers.utils.getAddress(predeploys.L2StandardBridge)
      )
      await assertContractVariable(contract, 'REMOTE_CHAIN_ID', remoteChainId)
    },
  })
}

deployFn.tags = ['OptimismMintableERC721FactoryImpl', 'l2']

export default deployFn
