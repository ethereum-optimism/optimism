/* Imports */
import { ethers } from 'hardhat'
import { Signer, ContractFactory, Contract, constants } from 'ethers'
import { Interface } from 'ethers/lib/utils'
import {
  smock,
  MockContractFactory,
  FakeContract,
  MockContract,
} from '@defi-wonderland/smock'
import ICrossDomainMessenger from '@eth-optimism/contracts/artifacts/contracts/libraries/bridge/ICrossDomainMessenger.sol/ICrossDomainMessenger.json'

import { NON_NULL_BYTES32, NON_ZERO_ADDRESS } from '../../helpers'
import { expect } from '../../setup'

const ERR_INVALID_X_DOMAIN_MESSAGE =
  'ERC721Bridge: function can only be called from the other bridge'
const DUMMY_L2_ERC721_ADDRESS = ethers.utils.getAddress(
  '0x' + 'abba'.repeat(10)
)
const DUMMY_L2_BRIDGE_ADDRESS = ethers.utils.getAddress(
  '0x' + 'acdc'.repeat(10)
)

const FINALIZATION_GAS = 600_000

describe('L1ERC721Bridge', () => {
  // init signers
  let l1MessengerImpersonator: Signer
  let alice: Signer
  let bob: Signer
  let bobsAddress
  let aliceAddress
  let tokenId
  let aliceInitialBalance

  // we can just make up this string since it's on the "other" Layer
  let Factory__L1ERC721: MockContractFactory<ContractFactory>
  let IL2ERC721Bridge: Interface
  before(async () => {
    ;[l1MessengerImpersonator, alice, bob] = await ethers.getSigners()

    // deploy an ERC721 contract on L1
    Factory__L1ERC721 = await smock.mock(
      '@openzeppelin/contracts/token/ERC721/ERC721.sol:ERC721'
    )

    // get an L2ERC721Bridge Interface
    IL2ERC721Bridge = (await ethers.getContractFactory('L2ERC721Bridge'))
      .interface

    aliceAddress = await alice.getAddress()
    bobsAddress = await bob.getAddress()
    aliceInitialBalance = 5
    tokenId = 10
  })

  let L1ERC721: MockContract<Contract>
  let L1ERC721Bridge: Contract
  let Fake__L1CrossDomainMessenger: FakeContract
  let Factory__L1ERC721Bridge: ContractFactory
  beforeEach(async () => {
    // Get a new mock L1 messenger
    Fake__L1CrossDomainMessenger = await smock.fake<Contract>(
      new ethers.utils.Interface(ICrossDomainMessenger.abi),
      { address: await l1MessengerImpersonator.getAddress() } // This allows us to use an ethers override {from: Fake__L1CrossDomainMessenger.address} to mock calls
    )

    // Deploy the contract under test
    Factory__L1ERC721Bridge = await ethers.getContractFactory('L1ERC721Bridge')
    L1ERC721Bridge = await Factory__L1ERC721Bridge.deploy(
      Fake__L1CrossDomainMessenger.address,
      DUMMY_L2_BRIDGE_ADDRESS
    )

    L1ERC721 = await Factory__L1ERC721.deploy('L1ERC721', 'ERC')

    await L1ERC721.setVariable('_owners', {
      [tokenId]: aliceAddress,
    })
    await L1ERC721.setVariable('_balances', {
      [aliceAddress]: aliceInitialBalance,
    })
  })

  describe('constructor', async () => {
    it('initializes correctly', async () => {
      it('reverts if cross domain messenger is address(0)', async () => {
        await expect(
          Factory__L1ERC721Bridge.deploy(
            constants.AddressZero,
            DUMMY_L2_BRIDGE_ADDRESS
          )
        ).to.be.revertedWith('ERC721Bridge: messenger cannot be address(0)')
      })

      it('reverts if other bridge is address(0)', async () => {
        await expect(
          Factory__L1ERC721Bridge.deploy(
            Fake__L1CrossDomainMessenger.address,
            constants.AddressZero
          )
        ).to.be.revertedWith('ERC721Bridge: other bridge cannot be address(0)')
      })

      expect(await L1ERC721Bridge.messenger()).equals(
        Fake__L1CrossDomainMessenger.address
      )
      expect(await L1ERC721Bridge.otherBridge()).equals(DUMMY_L2_BRIDGE_ADDRESS)
    })
  })

  describe('ERC721 deposits', () => {
    beforeEach(async () => {
      await L1ERC721.connect(alice).approve(L1ERC721Bridge.address, tokenId)
    })

    it('bridgeERC721() reverts if remote token is address(0)', async () => {
      await expect(
        L1ERC721Bridge.connect(alice).bridgeERC721(
          L1ERC721.address,
          constants.AddressZero,
          tokenId,
          FINALIZATION_GAS,
          NON_NULL_BYTES32
        )
      ).to.be.revertedWith('ERC721Bridge: remote token cannot be address(0)')
    })

    it('bridgeERC721() escrows the deposit and sends the correct deposit message', async () => {
      // alice calls deposit on the bridge and the L1 bridge calls transferFrom on the token.
      // emits an ERC721BridgeInitiated event with the correct arguments.
      await expect(
        L1ERC721Bridge.connect(alice).bridgeERC721(
          L1ERC721.address,
          DUMMY_L2_ERC721_ADDRESS,
          tokenId,
          FINALIZATION_GAS,
          NON_NULL_BYTES32
        )
      )
        .to.emit(L1ERC721Bridge, 'ERC721BridgeInitiated')
        .withArgs(
          L1ERC721.address,
          DUMMY_L2_ERC721_ADDRESS,
          aliceAddress,
          aliceAddress,
          tokenId,
          NON_NULL_BYTES32
        )

      const depositCallToMessenger =
        Fake__L1CrossDomainMessenger.sendMessage.getCall(0)

      // alice's balance decreases by 1
      const depositerBalance = await L1ERC721.balanceOf(aliceAddress)
      expect(depositerBalance).to.equal(aliceInitialBalance - 1)

      // bridge's balance increases by 1
      const bridgeBalance = await L1ERC721.balanceOf(L1ERC721Bridge.address)
      expect(bridgeBalance).to.equal(1)

      // Check the correct cross-chain call was sent:
      // Message should be sent to the L2 bridge
      expect(depositCallToMessenger.args[0]).to.equal(DUMMY_L2_BRIDGE_ADDRESS)
      // Message data should be a call telling the L2DepositedERC721 to finalize the deposit

      // the L1 bridge sends the correct message to the L1 messenger
      expect(depositCallToMessenger.args[1]).to.equal(
        IL2ERC721Bridge.encodeFunctionData('finalizeBridgeERC721', [
          DUMMY_L2_ERC721_ADDRESS,
          L1ERC721.address,
          aliceAddress,
          aliceAddress,
          tokenId,
          NON_NULL_BYTES32,
        ])
      )
      expect(depositCallToMessenger.args[2]).to.equal(FINALIZATION_GAS)

      // Updates the deposits mapping
      expect(
        await L1ERC721Bridge.deposits(
          L1ERC721.address,
          DUMMY_L2_ERC721_ADDRESS,
          tokenId
        )
      ).to.equal(true)
    })

    it('bridgeERC721To() reverts if NFT receiver is address(0)', async () => {
      await expect(
        L1ERC721Bridge.connect(alice).bridgeERC721To(
          L1ERC721.address,
          DUMMY_L2_ERC721_ADDRESS,
          constants.AddressZero,
          tokenId,
          FINALIZATION_GAS,
          NON_NULL_BYTES32
        )
      ).to.be.revertedWith('ERC721Bridge: nft recipient cannot be address(0)')
    })

    it('bridgeERC721To() escrows the deposited NFT and sends the correct deposit message', async () => {
      // depositor calls deposit on the bridge and the L1 bridge calls transferFrom on the token.
      // emits an ERC721BridgeInitiated event with the correct arguments.
      await expect(
        L1ERC721Bridge.connect(alice).bridgeERC721To(
          L1ERC721.address,
          DUMMY_L2_ERC721_ADDRESS,
          bobsAddress,
          tokenId,
          FINALIZATION_GAS,
          NON_NULL_BYTES32
        )
      )
        .to.emit(L1ERC721Bridge, 'ERC721BridgeInitiated')
        .withArgs(
          L1ERC721.address,
          DUMMY_L2_ERC721_ADDRESS,
          aliceAddress,
          bobsAddress,
          tokenId,
          NON_NULL_BYTES32
        )

      const depositCallToMessenger =
        Fake__L1CrossDomainMessenger.sendMessage.getCall(0)

      // alice's balance decreases by 1
      const depositerBalance = await L1ERC721.balanceOf(aliceAddress)
      expect(depositerBalance).to.equal(aliceInitialBalance - 1)

      // bridge's balance is increased
      const bridgeBalance = await L1ERC721.balanceOf(L1ERC721Bridge.address)
      expect(bridgeBalance).to.equal(1)

      // bridge is owner of tokenId
      const tokenIdOwner = await L1ERC721.ownerOf(tokenId)
      expect(tokenIdOwner).to.equal(L1ERC721Bridge.address)

      // Check the correct cross-chain call was sent:
      // Message should be sent to the L2DepositedERC721 on L2
      expect(depositCallToMessenger.args[0]).to.equal(DUMMY_L2_BRIDGE_ADDRESS)
      // Message data should be a call telling the L2DepositedERC721 to finalize the deposit

      // the L1 bridge sends the correct message to the L1 messenger
      expect(depositCallToMessenger.args[1]).to.equal(
        IL2ERC721Bridge.encodeFunctionData('finalizeBridgeERC721', [
          DUMMY_L2_ERC721_ADDRESS,
          L1ERC721.address,
          aliceAddress,
          bobsAddress,
          tokenId,
          NON_NULL_BYTES32,
        ])
      )
      expect(depositCallToMessenger.args[2]).to.equal(FINALIZATION_GAS)

      // Updates the deposits mapping
      expect(
        await L1ERC721Bridge.deposits(
          L1ERC721.address,
          DUMMY_L2_ERC721_ADDRESS,
          tokenId
        )
      ).to.equal(true)
    })

    it('cannot bridgeERC721 from a contract account', async () => {
      await expect(
        L1ERC721Bridge.bridgeERC721(
          L1ERC721.address,
          DUMMY_L2_ERC721_ADDRESS,
          tokenId,
          FINALIZATION_GAS,
          NON_NULL_BYTES32
        )
      ).to.be.revertedWith('ERC721Bridge: account is not externally owned')
    })

    describe('Handling ERC721.transferFrom() failures that revert', () => {
      it('bridgeERC721(): will revert if ERC721.transferFrom() reverts', async () => {
        await expect(
          L1ERC721Bridge.connect(bob).bridgeERC721To(
            L1ERC721.address,
            DUMMY_L2_ERC721_ADDRESS,
            bobsAddress,
            tokenId,
            FINALIZATION_GAS,
            NON_NULL_BYTES32
          )
        ).to.be.revertedWith('ERC721: transfer from incorrect owner')
      })

      it('bridgeERC721To(): will revert if ERC721.transferFrom() reverts', async () => {
        await expect(
          L1ERC721Bridge.connect(bob).bridgeERC721To(
            L1ERC721.address,
            DUMMY_L2_ERC721_ADDRESS,
            bobsAddress,
            tokenId,
            FINALIZATION_GAS,
            NON_NULL_BYTES32
          )
        ).to.be.revertedWith('ERC721: transfer from incorrect owner')
      })

      it('bridgeERC721To(): will revert if the L1 ERC721 is zero address', async () => {
        await expect(
          L1ERC721Bridge.connect(alice).bridgeERC721To(
            constants.AddressZero,
            DUMMY_L2_ERC721_ADDRESS,
            bobsAddress,
            tokenId,
            FINALIZATION_GAS,
            NON_NULL_BYTES32
          )
        ).to.be.revertedWith('function call to a non-contract account')
      })

      it('bridgeERC721To(): will revert if the L1 ERC721 has no code', async () => {
        await expect(
          L1ERC721Bridge.connect(alice).bridgeERC721To(
            bobsAddress,
            DUMMY_L2_ERC721_ADDRESS,
            bobsAddress,
            tokenId,
            FINALIZATION_GAS,
            NON_NULL_BYTES32
          )
        ).to.be.revertedWith('function call to a non-contract account')
      })
    })
  })

  describe('ERC721 withdrawals', () => {
    it('onlyFromCrossDomainAccount: should revert on calls from a non-crossDomainMessenger L1 account', async () => {
      await expect(
        L1ERC721Bridge.connect(alice).finalizeBridgeERC721(
          L1ERC721.address,
          DUMMY_L2_ERC721_ADDRESS,
          constants.AddressZero,
          constants.AddressZero,
          tokenId,
          NON_NULL_BYTES32
        )
      ).to.be.revertedWith(ERR_INVALID_X_DOMAIN_MESSAGE)
    })

    it('onlyFromCrossDomainAccount: should revert on calls from the right crossDomainMessenger, but wrong xDomainMessageSender (ie. not the L2DepositedERC721)', async () => {
      await expect(
        L1ERC721Bridge.finalizeBridgeERC721(
          L1ERC721.address,
          DUMMY_L2_ERC721_ADDRESS,
          constants.AddressZero,
          constants.AddressZero,
          tokenId,
          NON_NULL_BYTES32,
          {
            from: Fake__L1CrossDomainMessenger.address,
          }
        )
      ).to.be.revertedWith(ERR_INVALID_X_DOMAIN_MESSAGE)
    })

    describe('withdrawal attempts that pass the onlyFromCrossDomainAccount check', () => {
      beforeEach(async () => {
        // First Alice will send an NFT so that there's a balance to be withdrawn
        await L1ERC721.connect(alice).approve(L1ERC721Bridge.address, tokenId)

        await L1ERC721Bridge.connect(alice).bridgeERC721(
          L1ERC721.address,
          DUMMY_L2_ERC721_ADDRESS,
          tokenId,
          FINALIZATION_GAS,
          NON_NULL_BYTES32
        )

        // make sure bridge owns NFT
        expect(await L1ERC721.ownerOf(tokenId)).to.equal(L1ERC721Bridge.address)

        Fake__L1CrossDomainMessenger.xDomainMessageSender.returns(
          DUMMY_L2_BRIDGE_ADDRESS
        )
      })

      it('should credit funds to the withdrawer to finalize withdrawal', async () => {
        // finalizing the withdrawal emits an ERC721BridgeFinalized event with the correct arguments.
        await expect(
          L1ERC721Bridge.finalizeBridgeERC721(
            L1ERC721.address,
            DUMMY_L2_ERC721_ADDRESS,
            NON_ZERO_ADDRESS,
            NON_ZERO_ADDRESS,
            tokenId,
            NON_NULL_BYTES32,
            { from: Fake__L1CrossDomainMessenger.address }
          )
        )
          .to.emit(L1ERC721Bridge, 'ERC721BridgeFinalized')
          .withArgs(
            L1ERC721.address,
            DUMMY_L2_ERC721_ADDRESS,
            NON_ZERO_ADDRESS,
            NON_ZERO_ADDRESS,
            tokenId,
            NON_NULL_BYTES32
          )

        // NFT is transferred to new owner
        expect(await L1ERC721.ownerOf(tokenId)).to.equal(NON_ZERO_ADDRESS)

        // deposits state variable is updated
        expect(
          await L1ERC721Bridge.deposits(
            L1ERC721.address,
            DUMMY_L2_ERC721_ADDRESS,
            tokenId
          )
        ).to.equal(false)
      })
    })
  })
})
