/* Imports: External */
import { Contract, ContractFactory, utils, Wallet } from 'ethers'
import { ethers } from 'hardhat'
import { predeploys } from '@eth-optimism/contracts'
import Artifact__TestERC721 from '@eth-optimism/contracts-periphery/artifacts/contracts/testing/helpers/TestERC721.sol/TestERC721.json'
import Artifact__L1ERC721Bridge from '@eth-optimism/contracts-periphery/artifacts/contracts/L1/messaging/L1ERC721Bridge.sol/L1ERC721Bridge.json'
import Artifact__L2ERC721Bridge from '@eth-optimism/contracts-periphery/artifacts/contracts/L2/messaging/L2ERC721Bridge.sol/L2ERC721Bridge.json'
import Artifact__L2StandardERC721Factory from '@eth-optimism/contracts-periphery/artifacts/contracts/L2/messaging/L2StandardERC721Factory.sol/L2StandardERC721Factory.json'
import Artifact__L2StandardERC721 from '@eth-optimism/contracts-periphery/artifacts/contracts/standards/L2StandardERC721.sol/L2StandardERC721.json'

/* Imports: Internal */
import { expect } from './shared/setup'
import { OptimismEnv } from './shared/env'
import { withdrawalTest } from './shared/utils'

const TOKEN_ID: number = 1
const FINALIZATION_GAS: number = 1_200_000
const NON_NULL_BYTES: string = '0x1111'

describe('ERC721 Bridge', () => {
  let env: OptimismEnv
  before(async () => {
    env = await OptimismEnv.new()
  })

  let aliceWalletL1: Wallet
  let aliceWalletL2: Wallet
  let aliceAddress: string
  let bobWalletL1: Wallet
  let bobWalletL2: Wallet
  let bobAddress: string
  before(async () => {
    const alice = Wallet.createRandom()
    aliceWalletL1 = alice.connect(env.l1Wallet.provider)
    aliceWalletL2 = alice.connect(env.l2Wallet.provider)
    aliceAddress = aliceWalletL1.address

    const tx = await env.l2Wallet.sendTransaction({
      to: aliceAddress,
      value: utils.parseEther('0.01'),
    })
    await tx.wait()

    bobWalletL1 = env.l1Wallet
    bobWalletL2 = env.l2Wallet
    bobAddress = env.l1Wallet.address
  })

  let Factory__L1ERC721: ContractFactory
  let Factory__L1ERC721Bridge: ContractFactory
  let Factory__L2ERC721Bridge: ContractFactory
  let Factory__L2StandardERC721Factory: ContractFactory
  before(async () => {
    Factory__L1ERC721 = await ethers.getContractFactory(
      Artifact__TestERC721.abi,
      Artifact__TestERC721.bytecode,
      bobWalletL1
    )
    Factory__L1ERC721Bridge = await ethers.getContractFactory(
      Artifact__L1ERC721Bridge.abi,
      Artifact__L1ERC721Bridge.bytecode,
      bobWalletL1
    )
    Factory__L2ERC721Bridge = await ethers.getContractFactory(
      Artifact__L2ERC721Bridge.abi,
      Artifact__L2ERC721Bridge.bytecode,
      bobWalletL2
    )
    Factory__L2StandardERC721Factory = await ethers.getContractFactory(
      Artifact__L2StandardERC721Factory.abi,
      Artifact__L2StandardERC721Factory.bytecode,
      bobWalletL2
    )
  })

  let L1ERC721: Contract
  let L1ERC721Bridge: Contract
  let L2ERC721Bridge: Contract
  let L2StandardERC721Factory: Contract
  let L2StandardERC721: Contract
  beforeEach(async () => {
    L1ERC721 = await Factory__L1ERC721.deploy()
    await L1ERC721.deployed()

    L1ERC721Bridge = await Factory__L1ERC721Bridge.deploy()
    await L1ERC721Bridge.deployed()

    L2ERC721Bridge = await Factory__L2ERC721Bridge.deploy(
      predeploys.L2CrossDomainMessenger,
      L1ERC721Bridge.address
    )
    await L2ERC721Bridge.deployed()

    L2StandardERC721Factory = await Factory__L2StandardERC721Factory.deploy(
      L2ERC721Bridge.address
    )
    await L2StandardERC721Factory.deployed()

    // Create a L2 Standard ERC721 with the Standard ERC721 Factory
    const tx = await L2StandardERC721Factory.createStandardL2ERC721(
      L1ERC721.address,
      'L2ERC721',
      'L2'
    )
    await tx.wait()

    // Retrieve the deployed L2 Standard ERC721
    const L2StandardERC721Address =
      await L2StandardERC721Factory.standardERC721Mapping(L1ERC721.address)
    L2StandardERC721 = await ethers.getContractAt(
      Artifact__L2StandardERC721.abi,
      L2StandardERC721Address
    )
    await L2StandardERC721.deployed()

    // Initialize the L1 ERC721 Bridge
    const tx1 = await L1ERC721Bridge.initialize(
      env.messenger.contracts.l1.L1CrossDomainMessenger.address,
      L2ERC721Bridge.address
    )
    await tx1.wait()

    // Mint an L1 ERC721 to Bob on L1
    const tx2 = await L1ERC721.mint(bobAddress, TOKEN_ID)
    await tx2.wait()

    // Approve the L1 Bridge to operate the NFT
    const tx3 = await L1ERC721.approve(L1ERC721Bridge.address, TOKEN_ID)
    await tx3.wait()
  })

  it('depositERC721', async () => {
    await env.messenger.waitForMessageReceipt(
      await L1ERC721Bridge.depositERC721(
        L1ERC721.address,
        L2StandardERC721.address,
        TOKEN_ID,
        FINALIZATION_GAS,
        NON_NULL_BYTES
      )
    )

    // The L1 Bridge now owns the L1 NFT
    expect(await L1ERC721.ownerOf(TOKEN_ID)).to.equal(L1ERC721Bridge.address)

    // Bob owns the NFT on L2
    expect(await L2StandardERC721.ownerOf(TOKEN_ID)).to.equal(bobAddress)
  })

  it('depositERC721To', async () => {
    await env.messenger.waitForMessageReceipt(
      await L1ERC721Bridge.depositERC721To(
        L1ERC721.address,
        L2StandardERC721.address,
        aliceAddress,
        TOKEN_ID,
        FINALIZATION_GAS,
        NON_NULL_BYTES
      )
    )

    // The L1 Bridge now owns the L1 NFT
    expect(await L1ERC721.ownerOf(TOKEN_ID)).to.equal(L1ERC721Bridge.address)

    // Alice owns the NFT on L2
    expect(await L2StandardERC721.ownerOf(TOKEN_ID)).to.equal(aliceAddress)
  })

  withdrawalTest('withdraw', async () => {
    // Deposit an NFT into L2 so that there's something to withdraw
    await env.messenger.waitForMessageReceipt(
      await L1ERC721Bridge.depositERC721(
        L1ERC721.address,
        L2StandardERC721.address,
        TOKEN_ID,
        FINALIZATION_GAS,
        NON_NULL_BYTES
      )
    )

    // First, check that the L1 Bridge now owns the L1 NFT
    expect(await L1ERC721.ownerOf(TOKEN_ID)).to.equal(L1ERC721Bridge.address)

    // Also check that Bob owns the NFT on L2 initially
    expect(await L2StandardERC721.ownerOf(TOKEN_ID)).to.equal(bobAddress)

    const tx = await L2ERC721Bridge.withdraw(
      L2StandardERC721.address,
      TOKEN_ID,
      0,
      NON_NULL_BYTES
    )
    await tx.wait()
    await env.relayXDomainMessages(tx)

    // L1 NFT has been sent back to Bob
    expect(await L1ERC721.ownerOf(TOKEN_ID)).to.equal(bobAddress)

    // L2 NFT is burned
    await expect(L2StandardERC721.ownerOf(TOKEN_ID)).to.be.reverted
  })

  withdrawalTest('withdrawTo', async () => {
    // Deposit an NFT into L2 so that there's something to withdraw
    await env.messenger.waitForMessageReceipt(
      await L1ERC721Bridge.depositERC721(
        L1ERC721.address,
        L2StandardERC721.address,
        TOKEN_ID,
        FINALIZATION_GAS,
        NON_NULL_BYTES
      )
    )

    // First, check that the L1 Bridge now owns the L1 NFT
    expect(await L1ERC721.ownerOf(TOKEN_ID)).to.equal(L1ERC721Bridge.address)

    // Also check that Bob owns the NFT on L2 initially
    expect(await L2StandardERC721.ownerOf(TOKEN_ID)).to.equal(bobAddress)

    const tx = await L2ERC721Bridge.withdrawTo(
      L2StandardERC721.address,
      aliceAddress,
      TOKEN_ID,
      0,
      NON_NULL_BYTES
    )
    await tx.wait()
    await env.relayXDomainMessages(tx)

    // L1 NFT has been sent to Alice
    expect(await L1ERC721.ownerOf(TOKEN_ID)).to.equal(aliceAddress)

    // L2 NFT is burned
    await expect(L2StandardERC721.ownerOf(TOKEN_ID)).to.be.reverted
  })

  withdrawalTest(
    'should not allow an arbitrary L2 NFT to be withdrawn in exchange for a legitimate L1 NFT',
    async () => {
      // First, deposit the legitimate L1 NFT.
      await env.messenger.waitForMessageReceipt(
        await L1ERC721Bridge.depositERC721(
          L1ERC721.address,
          L2StandardERC721.address,
          TOKEN_ID,
          FINALIZATION_GAS,
          NON_NULL_BYTES
        )
      )
      // Check that the L1 Bridge owns the L1 NFT initially
      expect(await L1ERC721.ownerOf(TOKEN_ID)).to.equal(L1ERC721Bridge.address)

      // Deploy a fake L2 ERC721, which:
      // - Returns the address of the legitimate L1 token from its l1Token() getter.
      // - Allows the L2 bridge to call its burn() function.
      const FakeL2StandardERC721 = await (
        await ethers.getContractFactory('FakeL2StandardERC721', bobWalletL2)
      ).deploy(L1ERC721.address, L2ERC721Bridge.address)
      await FakeL2StandardERC721.deployed()

      // Use the fake contract to mint Alice an NFT with the same token ID
      const tx = await FakeL2StandardERC721.mint(aliceAddress, TOKEN_ID)
      await tx.wait()

      // Check that Alice owns the NFT from the fake ERC721 contract
      expect(await FakeL2StandardERC721.ownerOf(TOKEN_ID)).to.equal(
        aliceAddress
      )

      // Alice withdraws the NFT from the fake contract to L1, hoping to receive the legitimate L1 NFT.
      const withdrawalTx = await L2ERC721Bridge.connect(aliceWalletL2).withdraw(
        FakeL2StandardERC721.address,
        TOKEN_ID,
        0,
        NON_NULL_BYTES
      )
      await withdrawalTx.wait()
      await env.relayXDomainMessages(withdrawalTx)

      // The legitimate NFT on L1 is still held in the bridge.
      expect(await L1ERC721.ownerOf(TOKEN_ID)).to.equal(L1ERC721Bridge.address)
    }
  )
})
