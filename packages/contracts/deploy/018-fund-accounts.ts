/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

/* Imports: Internal */
import { getDeployedContract } from '../src/hardhat-deploy-ethers'
import { ethers } from 'hardhat'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

   // Fund all default hardhat signers  with ETH if deploying on a local hardhat network
   const { chainId } = await ethers.provider.getNetwork()
   if (chainId === 31337) {
     const Proxy__OVM_L1ETHGateway = await getDeployedContract(
       hre,
       'Proxy__OVM_L1ETHGateway',
       {
         signerOrProvider: deployer,
         iface: 'OVM_L1ETHGateway',
       }
     )
     const signers = await ethers.getSigners()
     for (const signer of signers) {
       const to = await signer.getAddress()
       const amount = '100'
       const value = ethers.utils.parseEther(amount)
       await Proxy__OVM_L1ETHGateway.depositTo(to, { value })
       console.log(`✓ Funded ${to} on L2 with ${amount} ETH`)
     }
   }
}

deployFn.dependencies = ['Proxy__OVM_L1ETHGateway']
deployFn.tags = ['fund-accounts']

export default deployFn
