/* Imports */
import { ethers } from 'hardhat'
import { Signer, ContractFactory, Contract, constants } from 'ethers'
import { smock, FakeContract, MockContract } from '@defi-wonderland/smock'
import ICrossDomainMessenger from '@eth-optimism/contracts/artifacts/contracts/libraries/bridge/ICrossDomainMessenger.sol/ICrossDomainMessenger.json'

import { expect } from '../../../setup'
import {
  NON_NULL_BYTES32,
  NON_ZERO_ADDRESS,
} from '../../../../../contracts/test/helpers'

const ERR_ALREADY_INITIALIZED = 'Initializable: contract is already initialized'
const ERR_INVALID_MESSENGER = 'OVM_XCHAIN: messenger contract unauthenticated'
const ERR_INVALID_X_DOMAIN_MSG_SENDER =
  'OVM_XCHAIN: wrong sender of cross-domain message'
const DUMMY_L1BRIDGE_ADDRESS: string =
  '0x1234123412341234123412341234123412341234'
const DUMMY_L1ERC721_ADDRESS: string =
  '0x2234223412342234223422342234223422342234'
const ERR_INVALID_WITHDRAWAL: string =
  'Withdrawal is not being initiated by NFT owner'
const ALICE_INITIAL_BALANCE: number = 10
const TOKEN_ID: number = 10

describe('L2ERC721Bridge', () => {
  let alice: Signer
  let aliceAddress: string
  let bob: Signer
  let bobsAddress: string
  let l2MessengerImpersonator: Signer
  let Factory__L1ERC721Bridge: ContractFactory

  before(async () => {
    // Create a special signer which will enable us to send messages from the L2Messenger contract
    ;[l2MessengerImpersonator, alice, bob] = await ethers.getSigners()
    aliceAddress = await alice.getAddress()
    bobsAddress = await bob.getAddress()
    Factory__L1ERC721Bridge = await ethers.getContractFactory('L1ERC721Bridge')
  })

  let L2ERC721Bridge: Contract
  let L2ERC721: Contract
  let Fake__L2CrossDomainMessenger: FakeContract
  beforeEach(async () => {
    // Get a new fake L2 messenger
    Fake__L2CrossDomainMessenger = await smock.fake<Contract>(
      new ethers.utils.Interface(ICrossDomainMessenger.abi),
      // This allows us to use an ethers override {from: Fake__L2CrossDomainMessenger.address} to mock calls
      { address: await l2MessengerImpersonator.getAddress() }
    )

    // Deploy the contract under test
    L2ERC721Bridge = await (
      await ethers.getContractFactory('L2ERC721Bridge')
    ).deploy(Fake__L2CrossDomainMessenger.address, DUMMY_L1BRIDGE_ADDRESS)

    // Deploy an L2 ERC721
    L2ERC721 = await (
      await ethers.getContractFactory('OptimismMintableERC721')
    ).deploy(
      L2ERC721Bridge.address,
      DUMMY_L1ERC721_ADDRESS,
      'L2Token',
      'L2T',
      { gasLimit: 4_000_000 } // Necessary to avoid an out-of-gas error
    )
  })

  describe('initialize', () => {
    it('Should only be callable once', async () => {
      await expect(
        L2ERC721Bridge.initialize(
          Fake__L2CrossDomainMessenger.address,
          DUMMY_L1BRIDGE_ADDRESS
        )
      ).to.be.revertedWith(ERR_ALREADY_INITIALIZED)
    })
  })

  // test the transfer flow of moving a token from L1 to L2
  describe('finalizeBridgeERC721', () => {
    it('onlyFromCrossDomainAccount: should revert on calls from a non-crossDomainMessenger L2 account', async () => {
      await expect(
        L2ERC721Bridge.connect(alice).finalizeBridgeERC721(
          DUMMY_L1ERC721_ADDRESS,
          NON_ZERO_ADDRESS,
          NON_ZERO_ADDRESS,
          NON_ZERO_ADDRESS,
          TOKEN_ID,
          NON_NULL_BYTES32
        )
      ).to.be.revertedWith(ERR_INVALID_MESSENGER)
    })

    it('onlyFromCrossDomainAccount: should revert on calls from the right crossDomainMessenger, but wrong xDomainMessageSender (ie. not the L1ERC721Bridge)', async () => {
      Fake__L2CrossDomainMessenger.xDomainMessageSender.returns(
        NON_ZERO_ADDRESS
      )

      await expect(
        L2ERC721Bridge.connect(l2MessengerImpersonator).finalizeBridgeERC721(
          DUMMY_L1ERC721_ADDRESS,
          NON_ZERO_ADDRESS,
          NON_ZERO_ADDRESS,
          NON_ZERO_ADDRESS,
          TOKEN_ID,
          NON_NULL_BYTES32,
          {
            from: Fake__L2CrossDomainMessenger.address,
          }
        )
      ).to.be.revertedWith(ERR_INVALID_X_DOMAIN_MSG_SENDER)
    })

    it('should initialize a withdrawal if the L2 token is not compliant', async () => {
      // Deploy a non compliant ERC721
      const NonCompliantERC721 = await (
        await ethers.getContractFactory(
          '@openzeppelin/contracts/token/ERC721/ERC721.sol:ERC721'
        )
      ).deploy('L2Token', 'L2T')

      Fake__L2CrossDomainMessenger.xDomainMessageSender.returns(
        DUMMY_L1BRIDGE_ADDRESS
      )

      // A failed attempt to finalize the deposit causes an ERC721BridgeFailed event to be emitted.
      await expect(
        L2ERC721Bridge.connect(l2MessengerImpersonator).finalizeBridgeERC721(
          NonCompliantERC721.address,
          DUMMY_L1ERC721_ADDRESS,
          aliceAddress,
          bobsAddress,
          TOKEN_ID,
          NON_NULL_BYTES32,
          {
            from: Fake__L2CrossDomainMessenger.address,
          }
        )
      )
        .to.emit(L2ERC721Bridge, 'ERC721BridgeFailed')
        .withArgs(
          NonCompliantERC721.address,
          DUMMY_L1ERC721_ADDRESS,
          aliceAddress,
          bobsAddress,
          TOKEN_ID,
          NON_NULL_BYTES32
        )

      const withdrawalCallToMessenger =
        Fake__L2CrossDomainMessenger.sendMessage.getCall(0)

      expect(withdrawalCallToMessenger.args[0]).to.equal(DUMMY_L1BRIDGE_ADDRESS)
      expect(withdrawalCallToMessenger.args[1]).to.equal(
        Factory__L1ERC721Bridge.interface.encodeFunctionData(
          'finalizeBridgeERC721',
          [
            DUMMY_L1ERC721_ADDRESS,
            NonCompliantERC721.address,
            bobsAddress,
            aliceAddress,
            TOKEN_ID,
            NON_NULL_BYTES32,
          ]
        )
      )
      expect(withdrawalCallToMessenger.args[2]).to.equal(0)
    })

    it('should credit funds to the depositor', async () => {
      Fake__L2CrossDomainMessenger.xDomainMessageSender.returns(
        DUMMY_L1BRIDGE_ADDRESS
      )

      // Assert that nobody owns the L2 token initially
      await expect(L2ERC721.ownerOf(TOKEN_ID)).to.be.revertedWith(
        'ERC721: owner query for nonexistent token'
      )

      // Successfully finalizes the deposit.
      const expectedResult = expect(
        L2ERC721Bridge.connect(l2MessengerImpersonator).finalizeBridgeERC721(
          L2ERC721.address,
          DUMMY_L1ERC721_ADDRESS,
          aliceAddress,
          bobsAddress,
          TOKEN_ID,
          NON_NULL_BYTES32,
          {
            from: Fake__L2CrossDomainMessenger.address,
          }
        )
      )

      // Depositing causes an ERC721BridgeFinalized event to be emitted.
      await expectedResult.to
        .emit(L2ERC721Bridge, 'ERC721BridgeFinalized')
        .withArgs(
          L2ERC721.address,
          DUMMY_L1ERC721_ADDRESS,
          aliceAddress,
          bobsAddress,
          TOKEN_ID,
          NON_NULL_BYTES32
        )

      // Causes a Transfer event to be emitted from the L2 ERC721.
      await expectedResult.to
        .emit(L2ERC721, 'Transfer')
        .withArgs(constants.AddressZero, bobsAddress, TOKEN_ID)

      // Bob is now the owner of the L2 ERC721
      const tokenIdOwner = await L2ERC721.ownerOf(TOKEN_ID)
      tokenIdOwner.should.equal(bobsAddress)
    })
  })

  describe('withdrawals', () => {
    let Mock__L2Token: MockContract<Contract>

    beforeEach(async () => {
      Mock__L2Token = await (
        await smock.mock('OptimismMintableERC721')
      ).deploy(
        L2ERC721Bridge.address,
        DUMMY_L1ERC721_ADDRESS,
        'L2Token',
        'L2T',
        { gasLimit: 4_000_000 } // Necessary to avoid an out-of-gas error
      )

      await Mock__L2Token.setVariable('_owners', {
        [TOKEN_ID]: aliceAddress,
      })
      await Mock__L2Token.setVariable('_balances', {
        [aliceAddress]: ALICE_INITIAL_BALANCE,
      })
    })

    it('bridgeERC721() reverts when called by non-owner of nft', async () => {
      await expect(
        L2ERC721Bridge.connect(bob).bridgeERC721(
          Mock__L2Token.address,
          DUMMY_L1ERC721_ADDRESS,
          TOKEN_ID,
          0,
          NON_NULL_BYTES32
        )
      ).to.be.revertedWith(ERR_INVALID_WITHDRAWAL)
    })

    it('bridgeERC721() reverts if called by a contract', async () => {
      await expect(
        L2ERC721Bridge.connect(l2MessengerImpersonator).bridgeERC721(
          Mock__L2Token.address,
          DUMMY_L1ERC721_ADDRESS,
          TOKEN_ID,
          0,
          NON_NULL_BYTES32
        )
      ).to.be.revertedWith('L2ERC721Bridge: account is not externally owned')
    })

    it('bridgeERC721() burns and sends the correct withdrawal message', async () => {
      // Make sure that alice begins as the NFT owner
      expect(await Mock__L2Token.ownerOf(TOKEN_ID)).to.equal(aliceAddress)

      // Initiates a successful withdrawal.
      const expectedResult = expect(
        L2ERC721Bridge.connect(alice).bridgeERC721(
          Mock__L2Token.address,
          DUMMY_L1ERC721_ADDRESS,
          TOKEN_ID,
          0,
          NON_NULL_BYTES32
        )
      )

      // A successful withdrawal causes an ERC721BridgeInitiated event to be emitted from the L2 ERC721 Bridge.
      await expectedResult.to
        .emit(L2ERC721Bridge, 'ERC721BridgeInitiated')
        .withArgs(
          Mock__L2Token.address,
          DUMMY_L1ERC721_ADDRESS,
          aliceAddress,
          aliceAddress,
          TOKEN_ID,
          NON_NULL_BYTES32
        )

      // A withdrawal also causes a Transfer event to be emitted the L2 ERC721, signifying that the L2 token
      // has been burnt.
      await expectedResult.to
        .emit(Mock__L2Token, 'Transfer')
        .withArgs(aliceAddress, constants.AddressZero, TOKEN_ID)

      // Assert Alice's balance went down
      const aliceBalance = await Mock__L2Token.balanceOf(aliceAddress)
      expect(aliceBalance).to.equal(ALICE_INITIAL_BALANCE - 1)

      // Assert that the token isn't owned by anyone
      await expect(Mock__L2Token.ownerOf(TOKEN_ID)).to.be.revertedWith(
        'ERC721: owner query for nonexistent token'
      )

      const withdrawalCallToMessenger =
        Fake__L2CrossDomainMessenger.sendMessage.getCall(0)

      // Assert the correct cross-chain call was sent:
      // Message should be sent to the L1ERC721Bridge on L1
      expect(withdrawalCallToMessenger.args[0]).to.equal(DUMMY_L1BRIDGE_ADDRESS)
      // Message data should be a call telling the L1ERC721Bridge to finalize the withdrawal
      expect(withdrawalCallToMessenger.args[1]).to.equal(
        Factory__L1ERC721Bridge.interface.encodeFunctionData(
          'finalizeBridgeERC721',
          [
            DUMMY_L1ERC721_ADDRESS,
            Mock__L2Token.address,
            aliceAddress,
            aliceAddress,
            TOKEN_ID,
            NON_NULL_BYTES32,
          ]
        )
      )
      // gaslimit should be correct
      expect(withdrawalCallToMessenger.args[2]).to.equal(0)
    })

    it('bridgeERC721To() reverts when called by non-owner of nft', async () => {
      await expect(
        L2ERC721Bridge.connect(bob).bridgeERC721To(
          Mock__L2Token.address,
          DUMMY_L1ERC721_ADDRESS,
          bobsAddress,
          TOKEN_ID,
          0,
          NON_NULL_BYTES32
        )
      ).to.be.revertedWith(ERR_INVALID_WITHDRAWAL)
    })

    it('bridgeERC721To() burns and sends the correct withdrawal message', async () => {
      // Make sure that alice begins as the NFT owner
      expect(await Mock__L2Token.ownerOf(TOKEN_ID)).to.equal(aliceAddress)

      // Initiates a successful withdrawal.
      const expectedResult = expect(
        L2ERC721Bridge.connect(alice).bridgeERC721To(
          Mock__L2Token.address,
          DUMMY_L1ERC721_ADDRESS,
          bobsAddress,
          TOKEN_ID,
          0,
          NON_NULL_BYTES32
        )
      )

      // A successful withdrawal causes an ERC721BridgeInitiated event to be emitted from the L2 ERC721 Bridge.
      await expectedResult.to
        .emit(L2ERC721Bridge, 'ERC721BridgeInitiated')
        .withArgs(
          Mock__L2Token.address,
          DUMMY_L1ERC721_ADDRESS,
          aliceAddress,
          bobsAddress,
          TOKEN_ID,
          NON_NULL_BYTES32
        )

      // A withdrawal also causes a Transfer event to be emitted the L2 ERC721, signifying that the L2 token
      // has been burnt.
      await expectedResult.to
        .emit(Mock__L2Token, 'Transfer')
        .withArgs(aliceAddress, constants.AddressZero, TOKEN_ID)

      // Assert Alice's balance went down
      const aliceBalance = await Mock__L2Token.balanceOf(aliceAddress)
      expect(aliceBalance).to.equal(ALICE_INITIAL_BALANCE - 1)

      // Assert that the token isn't owned by anyone
      await expect(Mock__L2Token.ownerOf(TOKEN_ID)).to.be.revertedWith(
        'ERC721: owner query for nonexistent token'
      )

      const withdrawalCallToMessenger =
        Fake__L2CrossDomainMessenger.sendMessage.getCall(0)

      // Assert the correct cross-chain call was sent.
      // Message should be sent to the L1ERC721Bridge on L1
      expect(withdrawalCallToMessenger.args[0]).to.equal(DUMMY_L1BRIDGE_ADDRESS)
      // The message data should be a call telling the L1ERC721Bridge to finalize the withdrawal
      expect(withdrawalCallToMessenger.args[1]).to.equal(
        Factory__L1ERC721Bridge.interface.encodeFunctionData(
          'finalizeBridgeERC721',
          [
            DUMMY_L1ERC721_ADDRESS,
            Mock__L2Token.address,
            aliceAddress,
            bobsAddress,
            TOKEN_ID,
            NON_NULL_BYTES32,
          ]
        )
      )
      // gas value is ignored and set to 0.
      expect(withdrawalCallToMessenger.args[2]).to.equal(0)
    })
  })
})
