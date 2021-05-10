import chai from 'chai'
import chaiAsPromised from 'chai-as-promised'
chai.use(chaiAsPromised)
const expect = chai.expect;

import { Contract, ContractFactory, BigNumber, Wallet, utils, providers } from 'ethers'
import { Direction } from './shared/watcher-utils'
import L2ERC721Json from '../artifacts-ovm/contracts/ERC721Mock.sol/ERC721Mock.json'
import { OptimismEnv } from './shared/env'

import * as fs from 'fs'

describe('NFT Test', async () => {

  let Factory__L2ERC721: ContractFactory
  let L2ERC721: Contract
  
  let env: OptimismEnv

  //Test Marc's BioBase NFT system
  const nftName = 'BioBase'
  const nftSymbol = 'BEE' //BioEconomy Explodes

  const getBalances = async (
    _address: string, 
    _env=env
   ) => {

    const aliceL1Balance = await _env.alicel1Wallet.getBalance()
    const aliceL2Balance = await _env.alicel2Wallet.getBalance()

    const bobL1Balance = await _env.bobl1Wallet.getBalance()
    const bobL2Balance = await _env.bobl2Wallet.getBalance()

    console.log("\nbobL1Balance:", bobL1Balance.toString())
    console.log("bobL2Balance:", bobL2Balance.toString())
    console.log("aliceL1Balance:", aliceL1Balance.toString())
    console.log("aliceL2Balance:", aliceL2Balance.toString())

    return {
      aliceL1Balance,
      aliceL2Balance,
      bobL1Balance,
      bobL2Balance,
    }
  }

  /************* BOB owns all the pools, and ALICE Mints a new token ***********/
  before(async () => {

    env = await OptimismEnv.new()

    Factory__L2ERC721 = new ContractFactory(
      L2ERC721Json.abi,
      L2ERC721Json.bytecode,
      env.bobl2Wallet
    )

  })

  before(async () => {

    // Mint a new NFT on L2
    // [nftSymbol, nftName]
    // this is owned by bobl1Wallet
    L2ERC721 = await Factory__L2ERC721.deploy(
      nftSymbol,
      nftName,
      BigNumber.from(String(0)) //starting index for the tokenIDs
    )
    await L2ERC721.deployTransaction.wait()
    console.log("Marc's BioBase NFT L2ERC721 deployed to:", L2ERC721.address)
    
  })

  before(async () => {

    fs.readFile('./deployment/local/addresses.json', 'utf8' , (err, data) => {
      
      if (err) {
        console.error(err)
        return
      }

      const addressArray = JSON.parse(data);      
      
      //this will either update or overwrite, depending, but either is fine 
      addressArray['L2ERC721'] = L2ERC721.address;

      fs.writeFile('./deployment/local/addresses.json', JSON.stringify(addressArray, null, 2), err => {
        if (err) {
          console.log('Error adding NFT address to file:', err)
        } else {
          console.log('Successfully added NFT address to file')
        }
      })
    })

  })

  it('should mint a new ERC721 and transfer it from Bob to Alice', async () => {
    
    const owner = env.bobl2Wallet.address;
    const recipient = env.alicel2Wallet.address;

    const ownerName = "Henrietta Lacks"

    //for some strange reason need a string here
    //no idea why that matters
    const tokenID = BigNumber.from(String(50));
    
    let meta = ownerName + "#" + Date.now().toString() + '#https://www.atcc.org/products/all/CCL-2.aspx';
    console.log("meta:",meta)

    //mint one NFT
    let nft = await L2ERC721.mintNFT(recipient,meta)
    await nft.wait()
    //console.log("ERC721:",nft)

    const balanceOwner = await L2ERC721.balanceOf(owner)
    const balanceRecipient = await L2ERC721.balanceOf(recipient)

    console.log("balanceOwner:",balanceOwner.toString())
    console.log("balanceRecipient:",balanceRecipient.toString())

    //Get the URL
    let nftURL = await L2ERC721.getTokenURI(BigNumber.from(String(0))) 
    console.log("nftURL:",nftURL)

    //Should be 1
    let TID = await L2ERC721.getLastTID() 
    console.log("TID:",TID.toString())

    //mint a second NFT
    meta = ownerName + "#" + Date.now().toString() + '#https://www.atcc.org/products/all/CCL-185.aspx';
    nft = await L2ERC721.mintNFT(recipient,meta)
    await nft.wait()

    //Should be 2
    TID = await L2ERC721.getLastTID() 
    console.log("TID:",TID.toString())

    //it('returns the amount of tokens owned by the given address', async function () {
    expect(await L2ERC721.balanceOf(owner)).to.deep.eq(BigNumber.from(String(0)));
    //});

    // Token 1 should be owned by recipient
    expect(await L2ERC721.ownerOf(BigNumber.from(String(1)))).to.deep.eq(recipient);

    //And Token 50 should not exist (at this point)
    expect(L2ERC721.ownerOf(BigNumber.from(String(50)))).to.be.eventually.rejectedWith("ERC721: owner query for nonexistent token");
  })

})