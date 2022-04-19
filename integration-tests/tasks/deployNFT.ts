import '@nomiclabs/hardhat-ethers'
import { ethers } from 'hardhat'

const main = async () => {
  // We get the contract to deploy
  const NFT = await ethers.getContractFactory('NFT')
  const nft = await NFT.deploy()
  await nft.deployed()

  console.log('NFT deployed to:', nft.address)
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error)
    process.exit(1)
  })
