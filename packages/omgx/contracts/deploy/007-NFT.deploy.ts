/* Imports: External */
import { DeployFunction, DeploymentSubmission } from 'hardhat-deploy/dist/types'
import { Contract, ContractFactory, utils, BigNumber} from 'ethers'
import { getContractFactory } from '@eth-optimism/contracts'
import chalk from 'chalk';

import L2ERC721Json from '../artifacts-ovm/contracts/ERC721Mock.sol/ERC721Mock.json'

let Factory__L2ERC721: ContractFactory
let L2ERC721: Contract

const nftName = 'TestNFT'
const nftSymbol = 'TST'

const deployFn: DeployFunction = async (hre) => {
    
  Factory__L2ERC721 = new ContractFactory(
    L2ERC721Json.abi,
    L2ERC721Json.bytecode,
    (hre as any).deployConfig.deployer_l2
  )

  // Mint a new NFT on L2
  // [nftSymbol, nftName]
  // this is owned by bobl1Wallet
  L2ERC721 = await Factory__L2ERC721.deploy(
    nftSymbol,
    nftName,
    BigNumber.from(String(0)), //starting index for the tokenIDs
    "", //the base URI is empty in this case
    {gasLimit: 800000, gasPrice: 0}
  )
  await L2ERC721.deployTransaction.wait()
  console.log(` 🌕 ${chalk.red('NFT L2ERC721 deployed to:')} ${chalk.green(L2ERC721.address)}`)

  const L2ERC721DeploymentSubmission: DeploymentSubmission = {
    ...L2ERC721,
    receipt: L2ERC721.receipt,
    address: L2ERC721.address,
    abi: L2ERC721.abi,
  }

  let owner = await L2ERC721.owner()
  console.log(` 🔒 ${chalk.red('ERC721 owner:')} ${chalk.green(owner)}`)

  await hre.deployments.save('L2ERC721', L2ERC721DeploymentSubmission)

}

deployFn.tags = ['L2ERC721', 'optional']

export default deployFn