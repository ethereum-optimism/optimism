//import { BigNumber} from 'ethers'

const func = async (hre) => {

  const { deployments, getNamedAccounts } = hre
  const { deploy } = deployments
  const { deployer } = await getNamedAccounts()

  const nftName = 'Test NFT'
  const nftSymbol = 'TST'

  await deploy('ERC721Mock', {
    from: deployer,
    args: [
      nftSymbol,
      nftName,
      0,//BigNumber.from(String(0)), //starting index for the tokenIDs
      '0x0000000000000000000000000000000000000000',
      'Genesis',
      'OMGX_Rinkeby_28',
    ],
    log: true
  })
}

func.tags = ['ERC721']
module.exports = func