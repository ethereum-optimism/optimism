/* External Imports */
import { ethers } from 'hardhat'
import { Signer, ContractFactory, Contract } from 'ethers'
import {
  smock,
  MockContractFactory,
  MockContract,
} from '@defi-wonderland/smock'

/* Internal Imports */
import { expect } from '../../../setup'

const L2_ERC721_BRIDGE_ADDRESS: string =
  '0xA779A0cA89556A9dffD47527F0aad1c2e0d66e46'

describe('L2StandardERC721Factory', () => {
  let signer: Signer
  let Factory__L1ERC721: MockContractFactory<ContractFactory>
  let L1ERC721: MockContract<Contract>
  let L2StandardERC721Factory: Contract
  let baseURI: string
  let chainId: number

  beforeEach(async () => {
    ;[signer] = await ethers.getSigners()

    // deploy an ERC721 contract on L1
    Factory__L1ERC721 = await smock.mock(
      '@openzeppelin/contracts/token/ERC721/ERC721.sol:ERC721'
    )
    L1ERC721 = await Factory__L1ERC721.deploy('L1ERC721', 'ERC')

    L2StandardERC721Factory = await (
      await ethers.getContractFactory('L2StandardERC721Factory')
    ).deploy(L2_ERC721_BRIDGE_ADDRESS)

    chainId = await signer.getChainId()
    baseURI = ''.concat(
      'ethereum:',
      L1ERC721.address.toLowerCase(),
      '@',
      chainId.toString(),
      '/tokenURI?uint256='
    )
  })

  it('should be deployed with the correct constructor argument', async () => {
    expect(await L2StandardERC721Factory.l2ERC721Bridge()).to.equal(
      L2_ERC721_BRIDGE_ADDRESS
    )
  })

  it('should be able to create a standard ERC721 contract', async () => {
    const tx = await L2StandardERC721Factory.createStandardL2ERC721(
      L1ERC721.address,
      'L2ERC721',
      'ERC'
    )
    const receipt = await tx.wait()

    // Get the StandardL2ERC721Created event
    const erc721CreatedEvent = receipt.events[0]

    // Expect there to be an event emitted for the standard token creation
    expect(erc721CreatedEvent.event).to.be.eq('StandardL2ERC721Created')

    // Get the L2 ERC721 address from the emitted event and check it was created correctly
    const l2ERC721Address = erc721CreatedEvent.args._l2Token
    const L2StandardERC721 = new Contract(
      l2ERC721Address,
      (await ethers.getContractFactory('L2StandardERC721')).interface,
      signer
    )

    expect(await L2StandardERC721.l2Bridge()).to.equal(L2_ERC721_BRIDGE_ADDRESS)
    expect(await L2StandardERC721.l1Token()).to.equal(L1ERC721.address)
    expect(await L2StandardERC721.name()).to.equal('L2ERC721')
    expect(await L2StandardERC721.symbol()).to.equal('ERC')
    expect(await L2StandardERC721.baseTokenURI()).to.equal(baseURI)

    expect(
      await L2StandardERC721Factory.isStandardERC721(L2StandardERC721.address)
    ).to.equal(true)
    expect(
      await L2StandardERC721Factory.standardERC721Mapping(L1ERC721.address)
    ).to.equal(l2ERC721Address)
  })

  it('should not be able to create a standard token with a 0 address for l1 token', async () => {
    await expect(
      L2StandardERC721Factory.createStandardL2ERC721(
        ethers.constants.AddressZero,
        'L2ERC721',
        'ERC'
      )
    ).to.be.revertedWith('Must provide L1 token address')
  })

  it('should not be able create two l2 standard tokens with the same l1 token', async () => {
    // The first call will not revert
    await L2StandardERC721Factory.createStandardL2ERC721(
      L1ERC721.address,
      'L2ERC721',
      'ERC'
    )

    await expect(
      L2StandardERC721Factory.createStandardL2ERC721(
        L1ERC721.address,
        'L2ERC721',
        'ERC'
      )
    ).to.be.revertedWith('L2 Standard Token already exists for this L1 Token')
  })
})
