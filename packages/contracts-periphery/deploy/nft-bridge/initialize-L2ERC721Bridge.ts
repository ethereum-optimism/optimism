/* Imports: External */
import { Contract } from 'ethers'
import { DeployFunction } from 'hardhat-deploy/dist/types'
import { hexStringEquals, awaitCondition } from '@eth-optimism/core-utils'
import { predeploys } from '@eth-optimism/contracts'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()
  const signer = hre.ethers.provider.getSigner(deployer)

  const L1ERC721Bridge = await hre.companionNetworks['l1'].deployments.get(
    'L1ERC721Bridge'
  )
  const Deployment__L2ERC721Bridge = await hre.deployments.get('L2ERC721Bridge')
  const L2ERC721Bridge = new Contract(
    Deployment__L2ERC721Bridge.address,
    Deployment__L2ERC721Bridge.abi,
    signer
  )

  const tx = await L2ERC721Bridge.initialize(L1ERC721Bridge.address)
  await tx.wait()

  // Ensures that the L2 bridge has been initialized with the correct parameters
  await awaitCondition(
    async () => {
      return (
        hexStringEquals(
          await L2ERC721Bridge.messenger(),
          predeploys.L2CrossDomainMessenger
        ) &&
        hexStringEquals(
          await L2ERC721Bridge.l1ERC721Bridge(),
          L1ERC721Bridge.address
        )
      )
    },
    5000,
    100
  )
}

deployFn.tags = ['initialize-l2-erc721-bridge']

export default deployFn
