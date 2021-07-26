/* External Imports */
const { ethers, network } = require('hardhat')
const chai = require('chai')
const chaiAsPromised = require('chai-as-promised')
const { solidity } = require('ethereum-waffle')
const { expect } = chai

chai.use(solidity)
chai.use(chaiAsPromised)

chai.should()

describe(`ERC721`, () => {

  const nftName = 'TestNFT'
  const nftSymbol = 'TST'

  const nftName_D = 'TestNFT_D'
  const nftSymbol_D = 'TST_D'
    
  let account1
  let account2
  let account3

  before(`load accounts`, async () => {
    ;[ account1, account2, account3 ] = await ethers.getSigners()
  })
  
  let ERC721
  let ERC721_D
  let ERC721Reg

  let a1a
  let a2a
  let a3a

  let Factory__ERC721 

  before(`deploy ERC721 contracts`, async () => {

    Factory__ERC721 = await ethers.getContractFactory('ERC721Genesis')
    
    ERC721 = await Factory__ERC721.connect(account1).deploy(
      nftName,
      nftSymbol,
      ethers.BigNumber.from(String(0)), //starting index for the tokenIDs
      '0x0000000000000000000000000000000000000000',
      'Genesis',
      'OMGX_Rinkeby_28',
      { gasLimit: 246210000 }
    )

    await ERC721.deployTransaction.wait()

    console.log(`NFT ERC721 deployed to: $(ERC721.address)`)

    const Factory__ERC721Reg = await ethers.getContractFactory('ERC721Registry')

    ERC721Reg = await Factory__ERC721Reg.connect(account1).deploy(
      { gasLimit: 22320000 }
    )

    await ERC721Reg.deployTransaction.wait()

    a1a = await account1.getAddress()
    a2a = await account2.getAddress()
    a3a = await account3.getAddress()

    const balanceOwner = await ERC721.balanceOf(a1a)
    console.log("Owner balance:", balanceOwner)

    let symbol = await ERC721.symbol()
    console.log("NFT Symbol:", symbol)

    let name = await ERC721.name()
    console.log("NFT Name:", name)

    let genesis = await ERC721.getGenesis()
    console.log("NFT Genesis:", genesis)

    console.log(`ERC721 owner: ${a1a}`)

  })

  it(`should have a name`, async () => {
    const tokenName = await ERC721.name()
    expect(tokenName).to.equal(nftName)
  })

  it('should generate a new ERC721 and transfer it from Bob (a1a) to Alice (a2a)', async () => {

    const ownerName = "Henrietta Lacks"

    let meta = ownerName + '#' + Date.now().toString() + '#' + 'https://www.atcc.org/products/all/CCL-2.aspx';
    console.log(`meta: ${meta}`)
    
    console.log("Alice (a1a):",a2a)

    //mint one NFT
    let nft = await ERC721.mintNFT(
      a2a,
      meta,
      { gasLimit: 21170000 }
    )
    await nft.wait()
    
    //console.log("ERC721:",nft)

    const balanceBob = await ERC721.balanceOf(a1a)
    const balanceAlice = await ERC721.balanceOf(a2a)

    console.log(`balanceOwner: ${balanceBob.toString()}`)
    console.log(`balanceAlice: ${balanceAlice.toString()}`)

    //Get the URL
    let nftURL = await ERC721.getTokenURI(
      ethers.BigNumber.from(String(0)),
      { gasLimit: 21170000 }
    )
    console.log(`nftURL: ${nftURL}`)

    //Should be 1
    let TID = await ERC721.getLastTID(
      { gasLimit: 21170000 }
    )
    console.log(`TID:${TID.toString()}`)

    //mint a second NFT for account3 aka recipient2
    meta = ownerName + '#' + Date.now().toString() + '#' + 'https://www.atcc.org/products/all/CCL-185.aspx';
    nft = await ERC721.mintNFT(
      a3a,
      meta,
      { gasLimit: 21170000 }
    )
    await nft.wait()

    //mint a third NFT, this time for account2 aka recipient
    meta = ownerName + '#' + Date.now().toString() + '#' + 'https://www.atcc.org/products/all/CCL-185.aspx';
    nft = await ERC721.mintNFT(
      a2a,
      meta,
      { gasLimit: 21170000 }
    )
    await nft.wait()

    //Should be 3
    TID = await ERC721.getLastTID({ gasLimit: 800000 })
    console.log(`TID:${TID.toString()}`)

    //it('returns the amount of tokens owned by the given address', async function () {
    expect(await ERC721.balanceOf(a1a)).to.deep.eq(ethers.BigNumber.from(String(0)))
    //});

    // Alice (a1a) should have two NFTs, and the tokenID of the first one should be zero, and the second one 
    // should be 2
    expect(await ERC721.ownerOf(ethers.BigNumber.from(String(0)))).to.deep.eq(a2a)
    expect(await ERC721.ownerOf(ethers.BigNumber.from(String(1)))).to.deep.eq(a3a)
    expect(await ERC721.ownerOf(ethers.BigNumber.from(String(2)))).to.deep.eq(a2a)

    // Token 50 should not exist (at this point)
    expect(ERC721.ownerOf(ethers.BigNumber.from(String(50)))).to.be.eventually.rejectedWith("ERC721: owner query for nonexistent token");
  })

  it('should derive an NFT Factory from a genesis NFT', async () => {
    
    //Alice (a2a) Account #2 wishes to create a derivative NFT factory from a genesis NFT
    const tokenID = await ERC721.tokenOfOwnerByIndex(
      a2a,
      1
    )

    //determine the UUID
    const UUID = ERC721.address.substring(1, 6) + '_' + tokenID.toString() + '_' + a2a.substring(1, 6)

    console.log(`Alice's UUID: ${UUID}`)

    ERC721_D = await Factory__ERC721.connect(account2).deploy(
      nftName_D,
      nftSymbol_D,
      ethers.BigNumber.from(String(0)), //starting index for the tokenIDs
      ERC721.address, //the genesis NFT address
      UUID,
      'OMGX_Rinkeby_28',
      { gasLimit: 246500000 }
    )

    await ERC721_D.deployTransaction.wait()

    console.log(`Derived NFT deployed to: ${ERC721_D.address}`)

    const meta = 'ADA BYRON, COUNTESS OF LOVELACE' + '#' + Date.now().toString() + '#' + 'http://blogs.bodleian.ox.ac.uk/wp-content/uploads/sites/163/2015/10/AdaByron-1850-1000x1200-e1444805848856.jpg'
    nft = await ERC721_D.mintNFT(
      a3a,
      meta,
      { gasLimit: 21170000 }
    )
    await nft.wait()

  })

  it('should register the NFTs address in users wallet', async () => {

    await ERC721Reg.registerAddress(
      a2a, 
      ERC721.address,
      { gasLimit: 21170000 }
    )
    
    //but, a3a should have two flavors of NFT...
    await ERC721Reg.registerAddress(
      a3a, 
      ERC721.address,
      { gasLimit: 21170000 }
    )

    await ERC721Reg.registerAddress(
      a3a, 
      ERC721_D.address,
      { gasLimit: 21170000 }
    )

    const addresses_a2a = await ERC721Reg.lookupAddress(
      a2a, { gasLimit: 21170000 }
    )

    const addresses_a3a = await ERC721Reg.lookupAddress(
      a3a, { gasLimit: 21170000 }
    )

    console.log(`Addresses a2a: ${addresses_a2a}`)
    console.log(`Addresses a3a: ${addresses_a3a}`)

  })
})
