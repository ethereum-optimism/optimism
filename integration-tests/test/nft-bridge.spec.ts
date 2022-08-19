/* Imports: External */
import { Contract, ContractFactory, utils, Wallet } from 'ethers'
import { ethers } from 'hardhat'
import { getChainId } from '@eth-optimism/core-utils'
import { predeploys } from '@eth-optimism/contracts'
import { MessageLike } from '@eth-optimism/sdk'
import Artifact__TestERC721 from '@eth-optimism/contracts-periphery/artifacts/contracts/testing/helpers/TestERC721.sol/TestERC721.json'
import Artifact__L1ERC721Bridge from '@eth-optimism/contracts-periphery/artifacts/contracts/L1/L1ERC721Bridge.sol/L1ERC721Bridge.json'
import Artifact__L2ERC721Bridge from '@eth-optimism/contracts-periphery/artifacts/contracts/L2/L2ERC721Bridge.sol/L2ERC721Bridge.json'
import Artifact__OptimismMintableERC721Factory from '@eth-optimism/contracts-periphery/artifacts/contracts/universal/op-erc721/OptimismMintableERC721Factory.sol/OptimismMintableERC721Factory.json'
import Artifact__OptimismMintableERC721 from '@eth-optimism/contracts-periphery/artifacts/contracts/universal/op-erc721/OptimismMintableERC721.sol/OptimismMintableERC721.json'

/* Imports: Internal */
import { expect } from './shared/setup'
import { OptimismEnv } from './shared/env'
import { withdrawalTest } from './shared/utils'

const TOKEN_ID: number = 1
const FINALIZATION_GAS: number = 600_000
const NON_NULL_BYTES: string = '0x1111'
const DUMMY_L1ERC721_ADDRESS: string = ethers.utils.getAddress(
  '0x' + 'acdc'.repeat(10)
)

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
  let Factory__OptimismMintableERC721Factory: ContractFactory
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
    Factory__OptimismMintableERC721Factory = await ethers.getContractFactory(
      Artifact__OptimismMintableERC721Factory.abi,
      Artifact__OptimismMintableERC721Factory.bytecode,
      bobWalletL2
    )
  })

  let L1ERC721: Contract
  let L1ERC721Bridge: Contract
  let L2ERC721Bridge: Contract
  let OptimismMintableERC721Factory: Contract
  let OptimismMintableERC721: Contract
  beforeEach(async () => {
    L1ERC721 = await Factory__L1ERC721.deploy()
    await L1ERC721.deployed()

    L1ERC721Bridge = await Factory__L1ERC721Bridge.deploy(
      env.messenger.contracts.l1.L1CrossDomainMessenger.address,
      ethers.utils.getContractAddress({
        from: await Factory__L2ERC721Bridge.signer.getAddress(),
        nonce: await Factory__L2ERC721Bridge.signer.getTransactionCount(),
      })
    )
    await L1ERC721Bridge.deployed()

    L2ERC721Bridge = await Factory__L2ERC721Bridge.deploy(
      predeploys.L2CrossDomainMessenger,
      L1ERC721Bridge.address
    )
    await L2ERC721Bridge.deployed()

    OptimismMintableERC721Factory =
      await Factory__OptimismMintableERC721Factory.deploy(
        L2ERC721Bridge.address,
        await getChainId(env.l1Wallet.provider)
      )
    await OptimismMintableERC721Factory.deployed()

    expect(await L1ERC721Bridge.otherBridge()).to.equal(L2ERC721Bridge.address)
    expect(await L2ERC721Bridge.otherBridge()).to.equal(L1ERC721Bridge.address)

    expect(await OptimismMintableERC721Factory.bridge()).to.equal(
      L2ERC721Bridge.address
    )

    // Create a L2 Standard ERC721 with the Standard ERC721 Factory
    const tx =
      await OptimismMintableERC721Factory.createStandardOptimismMintableERC721(
        L1ERC721.address,
        'L2ERC721',
        'L2'
      )
    const receipt = await tx.wait()

    // Get the OptimismMintableERC721Created event
    const erc721CreatedEvent = receipt.events[0]
    expect(erc721CreatedEvent.event).to.be.eq('OptimismMintableERC721Created')

    OptimismMintableERC721 = await ethers.getContractAt(
      Artifact__OptimismMintableERC721.abi,
      erc721CreatedEvent.args.localToken
    )

    // Mint an L1 ERC721 to Bob on L1
    const tx2 = await L1ERC721.mint(bobAddress, TOKEN_ID)
    await tx2.wait()

    // Approve the L1 Bridge to operate the NFT
    const tx3 = await L1ERC721.approve(L1ERC721Bridge.address, TOKEN_ID)
    await tx3.wait()
  })

  it('bridgeERC721', async () => {
    await env.messenger.waitForMessageReceipt(
      await L1ERC721Bridge.bridgeERC721(
        L1ERC721.address,
        OptimismMintableERC721.address,
        TOKEN_ID,
        FINALIZATION_GAS,
        NON_NULL_BYTES
      )
    )

    // The L1 Bridge now owns the L1 NFT
    expect(await L1ERC721.ownerOf(TOKEN_ID)).to.equal(L1ERC721Bridge.address)

    // Bob owns the NFT on L2
    expect(await OptimismMintableERC721.ownerOf(TOKEN_ID)).to.equal(bobAddress)
  })

  it('bridgeERC721To', async () => {
    await env.messenger.waitForMessageReceipt(
      await L1ERC721Bridge.bridgeERC721To(
        L1ERC721.address,
        OptimismMintableERC721.address,
        aliceAddress,
        TOKEN_ID,
        FINALIZATION_GAS,
        NON_NULL_BYTES
      )
    )

    // The L1 Bridge now owns the L1 NFT
    expect(await L1ERC721.ownerOf(TOKEN_ID)).to.equal(L1ERC721Bridge.address)

    // Alice owns the NFT on L2
    expect(await OptimismMintableERC721.ownerOf(TOKEN_ID)).to.equal(
      aliceAddress
    )
  })

  withdrawalTest('bridgeERC721', async () => {
    // Deposit an NFT into L2 so that there's something to withdraw
    await env.messenger.waitForMessageReceipt(
      await L1ERC721Bridge.bridgeERC721(
        L1ERC721.address,
        OptimismMintableERC721.address,
        TOKEN_ID,
        FINALIZATION_GAS,
        NON_NULL_BYTES
      )
    )

    // First, check that the L1 Bridge now owns the L1 NFT
    expect(await L1ERC721.ownerOf(TOKEN_ID)).to.equal(L1ERC721Bridge.address)

    // Also check that Bob owns the NFT on L2 initially
    expect(await OptimismMintableERC721.ownerOf(TOKEN_ID)).to.equal(bobAddress)

    const tx = await L2ERC721Bridge.bridgeERC721(
      OptimismMintableERC721.address,
      L1ERC721.address,
      TOKEN_ID,
      0,
      NON_NULL_BYTES
    )
    await tx.wait()
    await env.relayXDomainMessages(tx)

    // L1 NFT has been sent back to Bob
    expect(await L1ERC721.ownerOf(TOKEN_ID)).to.equal(bobAddress)

    // L2 NFT is burned
    await expect(OptimismMintableERC721.ownerOf(TOKEN_ID)).to.be.reverted
  })

  withdrawalTest('bridgeERC721To', async () => {
    // Deposit an NFT into L2 so that there's something to withdraw
    await env.messenger.waitForMessageReceipt(
      await L1ERC721Bridge.bridgeERC721(
        L1ERC721.address,
        OptimismMintableERC721.address,
        TOKEN_ID,
        FINALIZATION_GAS,
        NON_NULL_BYTES
      )
    )

    // First, check that the L1 Bridge now owns the L1 NFT
    expect(await L1ERC721.ownerOf(TOKEN_ID)).to.equal(L1ERC721Bridge.address)

    // Also check that Bob owns the NFT on L2 initially
    expect(await OptimismMintableERC721.ownerOf(TOKEN_ID)).to.equal(bobAddress)

    const tx = await L2ERC721Bridge.bridgeERC721To(
      OptimismMintableERC721.address,
      L1ERC721.address,
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
    await expect(OptimismMintableERC721.ownerOf(TOKEN_ID)).to.be.reverted
  })

  withdrawalTest(
    'should not allow an arbitrary L2 NFT to be withdrawn in exchange for a legitimate L1 NFT',
    async () => {
      // First, deposit the legitimate L1 NFT.
      await env.messenger.waitForMessageReceipt(
        await L1ERC721Bridge.bridgeERC721(
          L1ERC721.address,
          OptimismMintableERC721.address,
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
      const FakeOptimismMintableERC721 = await (
        await ethers.getContractFactory(
          'FakeOptimismMintableERC721',
          bobWalletL2
        )
      ).deploy(L1ERC721.address, L2ERC721Bridge.address)
      await FakeOptimismMintableERC721.deployed()

      // Use the fake contract to mint Alice an NFT with the same token ID
      const tx = await FakeOptimismMintableERC721.mint(aliceAddress, TOKEN_ID)
      await tx.wait()

      // Check that Alice owns the NFT from the fake ERC721 contract
      expect(await FakeOptimismMintableERC721.ownerOf(TOKEN_ID)).to.equal(
        aliceAddress
      )

      // Alice withdraws the NFT from the fake contract to L1, hoping to receive the legitimate L1 NFT.
      const withdrawalTx = await L2ERC721Bridge.connect(
        aliceWalletL2
      ).bridgeERC721(
        FakeOptimismMintableERC721.address,
        L1ERC721.address,
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

  withdrawalTest(
    'should refund an L2 NFT that fails to be finalized on l1',
    async () => {
      // Deploy an L2 native NFT, which:
      // - Mimics the interface of an OptimismMintableERC721.
      // - Allows anyone to mint tokens.
      // - Has a `remoteToken` state variable that returns the address of a non-existent L1 ERC721.
      //     This will cause the bridge to fail on L1, triggering a refund on L2.
      const L2NativeNFT = await (
        await ethers.getContractFactory(
          'FakeOptimismMintableERC721',
          aliceWalletL2
        )
      ).deploy(DUMMY_L1ERC721_ADDRESS, L2ERC721Bridge.address)
      await L2NativeNFT.deployed()

      // Alice mints an NFT from the L2 native ERC721 contract
      const tx = await L2NativeNFT.mint(aliceAddress, TOKEN_ID)
      await tx.wait()

      // Check that Alice owns the L2 NFT
      expect(await L2NativeNFT.ownerOf(TOKEN_ID)).to.equal(aliceAddress)

      // Alice bridges her L2 native NFT to L1, which burns the L2 NFT.
      const withdrawalTx = await L2ERC721Bridge.connect(
        aliceWalletL2
      ).bridgeERC721(
        L2NativeNFT.address,
        DUMMY_L1ERC721_ADDRESS,
        TOKEN_ID,
        FINALIZATION_GAS,
        NON_NULL_BYTES
      )
      await withdrawalTx.wait()

      // Check that the token was burnt on L2 (pre-refund).
      await expect(L2NativeNFT.ownerOf(TOKEN_ID)).to.be.revertedWith(
        'ERC721: owner query for nonexistent token'
      )

      // Relay the cross-domain transaction to L1, which initiates an L1 -> L2 message to refund
      // Alice her L2 NFT.
      await env.relayXDomainMessages(withdrawalTx)

      // Wait for the L1 -> L2 message to finalize on L2
      const txPair = await env.waitForXDomainTransaction(withdrawalTx)
      await env.messenger.waitForMessageReceipt(txPair.remoteTx as MessageLike)

      // Check that the L2 NFT has been refunded to Alice.
      expect(await L2NativeNFT.ownerOf(TOKEN_ID)).to.equal(aliceAddress)
    }
  )
})
