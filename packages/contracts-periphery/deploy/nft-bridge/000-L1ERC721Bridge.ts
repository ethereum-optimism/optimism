/* Imports: External */
import { DeployFunction } from 'hardhat-deploy/dist/types'

const deployFn: DeployFunction = async (hre) => {
  const { deployer } = await hre.getNamedAccounts()

  let Artifact__L1CrossDomainMessenger = await hre.deployments.getOrNull(
    'Proxy__OVM_L1CrossDomainMessenger'
  )

  if (Artifact__L1CrossDomainMessenger === undefined) {
    console.log(
      `L1CrossDomainMessenger proxy contract not deployed on ${hre.network.name}`
    )
    console.log('Deploying Proxy...')
    await hre.deployments.deploy('Proxy', {
      from: deployer,
      args: [deployer],
      log: true,
    })

    Artifact__L1CrossDomainMessenger = await hre.deployments.get('Proxy')
  }

  const L2ERC721Bridge = await hre.companionNetworks['l2'].deployments.getOrNull(
    'L2ERC721Bridge'
  )

  let addr: string
  if (L2ERC721Bridge === undefined) {
    console.log(
      `L2ERC721Bridge not deployed on L2 companion network for ${hre.network.name}`
    )
    console.log('Must compute address ahead of time')

    console.log(hre.companionNetworks)

    // TODO: this is incorrect, need to fetch it from the companionNetworks
    // rpc

    addr = hre.ethers.utils.getContractAddress({
      from: deployer,
      //nonce: await hre.companionNetworks['l2'].provider.getTransactionCount(deployer),
      nonce: 0,
    })
  } else {
    addr = L2ERC721Bridge.address
  }

  await hre.deployments.deploy('L1ERC721Bridge', {
    from: deployer,
    args: [Artifact__L1CrossDomainMessenger.address, addr],
    log: true,
  })
}

deployFn.tags = ['l1-nft-bridge', 'L1ERC721Bridge']

export default deployFn
