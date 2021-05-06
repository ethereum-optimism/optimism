import { expect } from 'chai'

import { Contract, ContractFactory, BigNumber, Wallet, utils, providers } from 'ethers'
import { Direction } from './shared/watcher-utils'

import L1ERC20Json from '../artifacts/contracts/ERC20.sol/ERC20.json'
import L1ERC20GatewayJson from '../artifacts/contracts/L1ERC20Gateway.sol/L1ERC20Gateway.json'
import L2DepositedERC20Json from '../artifacts-ovm/contracts/L2DepositedERC20.sol/L2DepositedERC20.json'
import L2ERC721Json from '../artifacts-ovm/contracts/NFT/ERC721Mock.sol/ERC721Mock.json'

import { OptimismEnv } from './shared/env'

import * as fs from 'fs'

describe('NFT Test', async () => {

  let Factory__L1ERC20: ContractFactory
  let Factory__L2DepositedERC20: ContractFactory
  let Factory__L1ERC20Gateway: ContractFactory
  let Factory__L2ERC721: ContractFactory

  let L1ERC20: Contract
  let L2DepositedERC20: Contract
  let L1ERC20Gateway: Contract
  let L2ERC721: Contract
  
  let env: OptimismEnv

  //Test ERC20 
  const initialAmount = utils.parseEther("10000000000")
  const tokenName = 'JLKN Test'
  const tokenDecimals = 18
  const tokenSymbol = 'JLKN'

  //Test Marc's BioBase NFT system
  const nftName = 'BioBase'
  const nftSymbol = 'BEE' //Bioeconomy Explodes

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

    Factory__L1ERC20 = new ContractFactory(
      L1ERC20Json.abi,
      L1ERC20Json.bytecode,
      env.bobl1Wallet
    )

    Factory__L2ERC721 = new ContractFactory(
      L2ERC721Json.abi,
      L2ERC721Json.bytecode,
      env.bobl2Wallet
    )

    Factory__L2DepositedERC20 = new ContractFactory(
      L2DepositedERC20Json.abi,
      L2DepositedERC20Json.bytecode,
      env.bobl2Wallet
    )

    Factory__L1ERC20Gateway = new ContractFactory(
      L1ERC20GatewayJson.abi,
      L1ERC20GatewayJson.bytecode,
      env.bobl1Wallet
    )

  })

  before(async () => {

    //Who? mints a new token and sets up the L1 and L2 infrastructure
    // Mint a new token on L1
    // [initialSupply, name, decimals, symbol]
    // this is owned by bobl1Wallet
    L1ERC20 = await Factory__L1ERC20.deploy(
      initialAmount,
      tokenName,
      tokenDecimals,
      tokenSymbol
    )
    await L1ERC20.deployTransaction.wait()
    console.log("L1ERC20 deployed to:", L1ERC20.address)

    // Who? sets up things on L2 for this new token
    // [l2MessengerAddress, name, symbol]
    L2DepositedERC20 = await Factory__L2DepositedERC20.deploy(
      env.watcher.l2.messengerAddress,
      tokenName,
      tokenSymbol
    )
    await L2DepositedERC20.deployTransaction.wait()
    console.log("L2DepositedERC20 deployed to:", L2DepositedERC20.address)
    
    // Who? deploys a gateway for this new token
    // [L1_ERC20.address, OVM_L2DepositedERC20.address, l1MessengerAddress]
    L1ERC20Gateway = await Factory__L1ERC20Gateway.deploy(
      L1ERC20.address,
      L2DepositedERC20.address,
      env.watcher.l1.messengerAddress,
    )
    await L1ERC20Gateway.deployTransaction.wait()
    console.log("L1ERC20Gateway deployed to:", L1ERC20Gateway.address)

    // Who initializes the contracts for the new token
    const initL2 = await L2DepositedERC20.init(L1ERC20Gateway.address);
    await initL2.wait();
    console.log('L2 ERC20 initialized:',initL2.hash);

    // Mint a new NFT on L2
    // [nftSymbol, nftName]
    // this is owned by bobl1Wallet
    L2ERC721 = await Factory__L2ERC721.deploy(
      nftSymbol,
      nftName
    )
    await L2ERC721.deployTransaction.wait()
    console.log("Marc's BioBase NFT L2ERC721 deployed to:", L2ERC721)
    console.log("Marc's BioBase NFT L2ERC721 deployed to:", L2ERC721.address)
    
  })

  before(async () => {
    //keep track of where things are for future use by the front end
    console.log("\n\n********************************\nSaving all key addresses")

    const addresses = {
      L1ERC20: L1ERC20.address,
      L2DepositedERC20: L2DepositedERC20.address,
      L1ERC20Gateway: L1ERC20Gateway.address,
      l1ETHGatewayAddress: env.L1ETHGateway.address,
      l1MessengerAddress: env.l1MessengerAddress,
      L2ERC721: L2ERC721.address
    }

    console.log(JSON.stringify(addresses, null, 2))

    fs.writeFile('./deployment/local/addresses.json', JSON.stringify(addresses, null, 2), err => {
      if (err) {
        console.log('Error writing addresses to file:', err)
      } else {
        console.log('Successfully wrote addresses to file')
      }
    })

    console.log('********************************\n\n')

  })

  it('should transfer ERC20 from Bob to Alice', async () => {
    
    const depositAmount = utils.parseEther("50")
    const preBalances = await getBalances("0x0000000000000000000000000000000000000000")
    
    console.log("\n Depositing...")

    const { tx, receipt } = await env.waitForXDomainTransaction(
      env.L1ETHGateway.deposit({ value: depositAmount }),
      Direction.L1ToL2
    )

    const l1FeePaid = receipt.gasUsed.mul(tx.gasPrice)
    const postBalances = await getBalances("0x0000000000000000000000000000000000000000")

    expect(postBalances.bobL2Balance).to.deep.eq(
      preBalances.bobL2Balance.add(depositAmount)
    )
    expect(postBalances.bobL1Balance).to.deep.eq(
      preBalances.bobL1Balance.sub(l1FeePaid.add(depositAmount))
    )

  })

  it('should mint a new ERC721 and transfer it from Bob to Alice', async () => {

    //const name = 'Non Fungible Token'
    //const symbol = 'NFT'
    //const firstTokenId = BigNumber.from('5042')
    
    //const baseURI = 'https://api.com/v1/'
    //const sampleUri = 'mock://mytoken'
    
    const owner = env.bobl2Wallet.address;
    const recipient = env.alicel2Wallet.address;

    let nft = await L2ERC721.mintNFT(
      recipient,
      BigNumber.from(1),
      'https://www.atcc.org/products/all/CCL-2.aspx'
    )
    await nft.wait()
    console.log("ERC721:",nft)

    nft = await L2ERC721.mintNFT(
      recipient,
      BigNumber.from(2),
      'https://www.atcc.org/products/all/CCL-2.aspx#characteristics'
    )
    await nft.wait()
    console.log("ERC7212:",nft)

    nft = await L2ERC721.mintNFT(
      recipient,
      BigNumber.from(42),
      'https://www.atcc.org/products/all/CCL-2.aspx#characte'
    )
    await nft.wait()
    console.log("ERC7212:",nft)

    const filter = {
      address: L2ERC721.address,
      topics: [
          // the name of the event, parnetheses containing the data type of each event, no spaces
          utils.id("Transfer(address,address,uint256)")
        ]
      }

    env.l2Provider.on(filter, (res) => {
        // do whatever you want here
        // I'm pretty sure this returns a promise, so don't forget to resolve it
        console.log("res:",res)
    })

    await L2ERC721.on("Transfer", (to, amount, from) => {
      console.log("Transfer:",to, amount, from);
    });

    const balanceOwner = await L2ERC721.balanceOf(owner);
    console.log("balanceOwner:",balanceOwner.toString())

    const balanceRecipient = await L2ERC721.balanceOf(recipient);
    console.log("balanceRecipient:",balanceRecipient.toString())

    let nftURL = await L2ERC721.getTokenURI(BigNumber.from(2));
    console.log("nftURL:",nftURL)
    nftURL = await L2ERC721.getTokenURI(BigNumber.from(42));
    console.log("nftURL:",nftURL)

    //it('returns the amount of tokens owned by the given address', async function () {
    //expect(await L2ERC721.balanceOf(owner)).to.deep.eq('1');
    //});

    //it('returns the owner of the given token ID', async function () {
    //expect(await L2ERC721.ownerOf(nft)).to.deep.eq(recipient);
    //});

  })
})