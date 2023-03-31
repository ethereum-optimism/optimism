import { DeployFunction } from 'hardhat-deploy/dist/types'
import '@eth-optimism/hardhat-deploy-config'
import '@nomiclabs/hardhat-ethers'
import { ethers } from 'ethers'

import { predeploys } from '../src/constants'
import { assertContractVariable, deploy } from '../src/deploy-utils'

// TODO: should be proxy and should be companion network

const deployFn: DeployFunction = async (hre) => {
  const Artifact__L1ERC721Bridge = await hre.deployments.get('L1ERC721Bridge')

  await deploy({
    hre,
    name: 'L2ERC721Bridge',
    args: [predeploys.L2CrossDomainMessenger, Artifact__L1ERC721Bridge.address],
    postDeployAction: async (contract) => {
      await assertContractVariable(
        contract,
        'MESSENGER',
        ethers.utils.getAddress(predeploys.L2CrossDomainMessenger)
      )
      await assertContractVariable(
        contract,
        'OTHER_BRIDGE',
        ethers.utils.getAddress(Artifact__L1ERC721Bridge.address)
      )
    },
  })
}

deployFn.tags = ['L2ERC721BridgeImpl', 'l2']

export default deployFn
