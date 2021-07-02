import chai from 'chai'
import chaiAsPromised from 'chai-as-promised'
chai.use(chaiAsPromised)
const expect = chai.expect;
import chalk from 'chalk';

import { Contract, ContractFactory, BigNumber } from 'ethers'
import L2ERC721Json from '../artifacts-ovm/contracts/ERC721Mock.sol/ERC721Mock.json'
import { OptimismEnv } from './shared/env'

import * as fs from 'fs'

describe('NFT Test\n', async () => {

  it('should mint an ERC721 and transfer it from Bob to Alice', async () => {

  //   const owner = env.bobl2Wallet.address;
  //   const recipient = env.alicel2Wallet.address;
  //   const ownerName = "Henrietta Lacks"

  //   let meta = ownerName + '#' + Date.now().toString() + '#' + 'https://www.atcc.org/products/all/CCL-2.aspx';
  //   console.log(` ⚽️ ${chalk.red(`meta:`)} ${chalk.green(`${meta}`)}`)

  //   //mint one NFT
  //   let nft = await L2ERC721.mintNFT(
  //     recipient,
  //     meta,
  //     {gasLimit: 800000, gasPrice: 0}
  //   )
  //   await nft.wait()
  //   // console.log("ERC721:",nft)

  //   const balanceOwner = await L2ERC721.balanceOf(owner)
  //   const balanceRecipient = await L2ERC721.balanceOf(recipient)

  //   console.log(` ⚽️ ${chalk.red(`balanceOwner:`)} ${chalk.green(`${balanceOwner.toString()}`)}`)
  //   console.log(` ⚽️ ${chalk.red(`balanceRecipient:`)} ${chalk.green(`${balanceRecipient.toString()}`)}`)

  //   //Get the URL
  //   let nftURL = await L2ERC721.getTokenURI(
  //     BigNumber.from(String(0)),
  //     {gasLimit: 800000, gasPrice: 0}
  //   )
  //   console.log(` ⚽️ ${chalk.red(`nftURL:`)} ${chalk.green(`${nftURL}`)}`)

  //   //Should be 1
  //   let TID = await L2ERC721.getLastTID({gasLimit: 800000, gasPrice: 0})
  //   console.log(` ⚽️ ${chalk.red(`TID:`)} ${chalk.green(`${TID.toString()}`)}`)

  //   //mint a second NFT
  //   meta = ownerName + '#' + Date.now().toString() + '#' + 'https://www.atcc.org/products/all/CCL-185.aspx';
  //   nft = await L2ERC721.mintNFT(
  //     recipient,
  //     meta,
  //     {gasLimit: 800000, gasPrice: 0}
  //   )
  //   await nft.wait()

  //   //Should be 2
  //   TID = await L2ERC721.getLastTID({ gasLimit: 800000, gasPrice: 0 })
  //   console.log(` ⚽️ ${chalk.red(`TID:`)} ${chalk.green(`${TID.toString()}`)}`)

  //   //it('returns the amount of tokens owned by the given address', async function () {
  //   expect(await L2ERC721.balanceOf(owner)).to.deep.eq(BigNumber.from(String(0)));

  //   // Token 1 should be owned by recipient
  //   expect(await L2ERC721.ownerOf(BigNumber.from(String(1)))).to.deep.eq(recipient);

  //   //And Token 50 should not exist (at this point)
  //   expect(L2ERC721.ownerOf(BigNumber.from(String(50)))).to.be.eventually.rejectedWith("ERC721: owner query for nonexistent token");
  })

})