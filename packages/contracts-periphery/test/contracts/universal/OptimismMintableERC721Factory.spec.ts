/* External Imports */
import { ethers } from 'hardhat'
import { Signer, ContractFactory, Contract } from 'ethers'
import {
  smock,
  MockContractFactory,
  MockContract,
} from '@defi-wonderland/smock'

/* Internal Imports */
import { expect } from '../../setup'

const DUMMY_L2_BRIDGE_ADDRESS: string = ethers.utils.getAddress(
  '0x' + 'acdc'.repeat(10)
)

describe('OptimismMintableERC721Factory', () => {
  let signer: Signer
  let Factory__L1ERC721: MockContractFactory<ContractFactory>
  let L1ERC721: MockContract<Contract>
  let OptimismMintableERC721Factory: Contract
  let baseURI: string
  const remoteChainId = 100

  beforeEach(async () => {
    ;[signer] = await ethers.getSigners()

    // deploy an ERC721 contract on L1
    Factory__L1ERC721 = await smock.mock(
      '@openzeppelin/contracts/token/ERC721/ERC721.sol:ERC721'
    )
    L1ERC721 = await Factory__L1ERC721.deploy('L1ERC721', 'ERC')

    OptimismMintableERC721Factory = await (
      await ethers.getContractFactory('OptimismMintableERC721Factory')
    ).deploy(DUMMY_L2_BRIDGE_ADDRESS, remoteChainId)

    baseURI = ''.concat(
      'ethereum:',
      L1ERC721.address.toLowerCase(),
      '@',
      remoteChainId.toString(),
      '/tokenURI?uint256='
    )
  })

  it('should be deployed with the correct constructor argument', async () => {
    expect(await OptimismMintableERC721Factory.bridge()).to.equal(
      DUMMY_L2_BRIDGE_ADDRESS
    )
  })

  it('should be able to create a standard ERC721 contract', async () => {
    const tx =
      await OptimismMintableERC721Factory.createStandardOptimismMintableERC721(
        L1ERC721.address,
        'L2ERC721',
        'ERC'
      )
    const receipt = await tx.wait()

    // Get the OptimismMintableERC721Created event
    const erc721CreatedEvent = receipt.events[0]

    // Expect there to be an event emitted for the standard token creation
    expect(erc721CreatedEvent.event).to.be.eq('OptimismMintableERC721Created')

    // Get the L2 ERC721 address from the emitted event and check it was created correctly
    const l2ERC721Address = erc721CreatedEvent.args.localToken
    const OptimismMintableERC721 = new Contract(
      l2ERC721Address,
      (await ethers.getContractFactory('OptimismMintableERC721')).interface,
      signer
    )

    expect(await OptimismMintableERC721.bridge()).to.equal(
      DUMMY_L2_BRIDGE_ADDRESS
    )
    expect(await OptimismMintableERC721.remoteToken()).to.equal(
      L1ERC721.address
    )
    expect(await OptimismMintableERC721.name()).to.equal('L2ERC721')
    expect(await OptimismMintableERC721.symbol()).to.equal('ERC')
    expect(await OptimismMintableERC721.baseTokenURI()).to.equal(baseURI)

    expect(
      await OptimismMintableERC721Factory.isStandardOptimismMintableERC721(
        OptimismMintableERC721.address
      )
    ).to.equal(true)
  })

  it('should not be able to create a standard token with a 0 address for l1 token', async () => {
    await expect(
      OptimismMintableERC721Factory.createStandardOptimismMintableERC721(
        ethers.constants.AddressZero,
        'L2ERC721',
        'ERC'
      )
    ).to.be.revertedWith(
      'OptimismMintableERC721Factory: L1 token address cannot be address(0)'
    )
  })
})
