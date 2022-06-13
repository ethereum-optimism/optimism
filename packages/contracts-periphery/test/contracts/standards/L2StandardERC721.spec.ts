/* External Imports */
import { ethers } from 'hardhat'
import { Signer, Contract } from 'ethers'
import { smock, FakeContract } from '@defi-wonderland/smock'

/* Internal Imports */
import { expect } from '../../setup'

const TOKEN_ID = 10
const DUMMY_L1ERC721_ADDRESS: string =
  '0x2234223412342234223422342234223422342234'

describe('L2StandardERC721', () => {
  let l2BridgeImpersonator: Signer
  let alice: Signer
  let Fake__L2ERC721Bridge: FakeContract
  let L2StandardERC721: Contract
  let l2BridgeImpersonatorAddress: string
  let aliceAddress: string
  let baseUri: string
  let chainId: number

  before(async () => {
    ;[l2BridgeImpersonator, alice] = await ethers.getSigners()
    l2BridgeImpersonatorAddress = await l2BridgeImpersonator.getAddress()
    aliceAddress = await alice.getAddress()

    chainId = await alice.getChainId()
    baseUri = ''.concat(
      'ethereum:',
      DUMMY_L1ERC721_ADDRESS,
      '@',
      chainId.toString(),
      '/tokenURI?uint256='
    )

    L2StandardERC721 = await (
      await ethers.getContractFactory('L2StandardERC721')
    ).deploy(
      l2BridgeImpersonatorAddress,
      DUMMY_L1ERC721_ADDRESS,
      'L2ERC721',
      'ERC',
      { gasLimit: 4_000_000 } // Necessary to avoid an out-of-gas error
    )

    // Get a new fake L2 bridge
    Fake__L2ERC721Bridge = await smock.fake<Contract>(
      'L2ERC721Bridge',
      // This allows us to use an ethers override {from: Fake__L2ERC721Bridge.address} to mock calls
      { address: await l2BridgeImpersonator.getAddress() }
    )

    // mint an nft to alice
    await L2StandardERC721.connect(l2BridgeImpersonator).mint(
      aliceAddress,
      TOKEN_ID,
      {
        from: Fake__L2ERC721Bridge.address,
      }
    )
  })

  describe('constructor', () => {
    it('should be able to create a standard ERC721 contract with the correct parameters', async () => {
      expect(await L2StandardERC721.l2Bridge()).to.equal(
        l2BridgeImpersonatorAddress
      )
      expect(await L2StandardERC721.l1Token()).to.equal(DUMMY_L1ERC721_ADDRESS)
      expect(await L2StandardERC721.name()).to.equal('L2ERC721')
      expect(await L2StandardERC721.symbol()).to.equal('ERC')
      expect(await L2StandardERC721.baseTokenURI()).to.equal(baseUri)

      // alice has been minted an nft
      expect(await L2StandardERC721.ownerOf(TOKEN_ID)).to.equal(aliceAddress)
    })
  })

  describe('mint and burn', () => {
    it('should not allow anyone but the L2 bridge to mint and burn', async () => {
      await expect(
        L2StandardERC721.connect(alice).mint(aliceAddress, 100)
      ).to.be.revertedWith('Only L2 Bridge can mint and burn')
      await expect(
        L2StandardERC721.connect(alice).burn(aliceAddress, 100)
      ).to.be.revertedWith('Only L2 Bridge can mint and burn')
    })
  })

  describe('supportsInterface', () => {
    it('should return the correct interface support', async () => {
      const supportsERC165 = await L2StandardERC721.supportsInterface(
        0x01ffc9a7
      )
      expect(supportsERC165).to.be.true

      const supportsL2TokenInterface = await L2StandardERC721.supportsInterface(
        0x1d1d8b63
      )
      expect(supportsL2TokenInterface).to.be.true

      const supportsERC721Interface = await L2StandardERC721.supportsInterface(
        0x80ac58cd
      )
      expect(supportsERC721Interface).to.be.true

      const badSupports = await L2StandardERC721.supportsInterface(0xffffffff)
      expect(badSupports).to.be.false
    })
  })

  describe('tokenURI', () => {
    it('should return the correct token uri', async () => {
      const tokenUri = baseUri.concat(TOKEN_ID.toString())
      expect(await L2StandardERC721.tokenURI(TOKEN_ID)).to.equal(tokenUri)
    })
  })
})
