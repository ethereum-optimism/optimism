/* External Imports */
import { ethers } from 'hardhat'
import { Signer, ContractFactory, Contract } from 'ethers'
import { smock, FakeContract, MockContract } from '@defi-wonderland/smock'

/* Internal Imports */
import { expect } from '../../../setup'
import { NON_NULL_BYTES32, NON_ZERO_ADDRESS } from '../../../helpers'
import { getContractInterface } from '../../../../src'

const ERR_INVALID_MESSENGER = 'OVM_XCHAIN: messenger contract unauthenticated'
const ERR_INVALID_X_DOMAIN_MSG_SENDER =
  'OVM_XCHAIN: wrong sender of cross-domain message'
const DUMMY_L1BRIDGE_ADDRESS: string =
  '0x1234123412341234123412341234123412341234'
const DUMMY_L1TOKEN_ADDRESS: string =
  '0x2234223412342234223422342234223422342234'
const OVM_ETH_ADDRESS: string = '0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000'

describe('L2StandardBridge', () => {
  let alice: Signer
  let aliceAddress: string
  let bob: Signer
  let bobsAddress: string
  let l2MessengerImpersonator: Signer
  let Factory__L1StandardBridge: ContractFactory
  const INITIAL_TOTAL_SUPPLY = 100_000
  const ALICE_INITIAL_BALANCE = 50_000
  before(async () => {
    // Create a special signer which will enable us to send messages from the L2Messenger contract
    ;[alice, bob, l2MessengerImpersonator] = await ethers.getSigners()
    aliceAddress = await alice.getAddress()
    bobsAddress = await bob.getAddress()
    Factory__L1StandardBridge = await ethers.getContractFactory(
      'L1StandardBridge'
    )

    // get an L2ER20Bridge Interface
    getContractInterface('IL2ERC20Bridge')
  })

  let L2StandardBridge: Contract
  let L2ERC20: Contract
  let Fake__L2CrossDomainMessenger: FakeContract
  beforeEach(async () => {
    // Get a new mock L2 messenger
    Fake__L2CrossDomainMessenger = await smock.fake<Contract>(
      await ethers.getContractFactory('L2CrossDomainMessenger'),
      // This allows us to use an ethers override {from: Mock__L2CrossDomainMessenger.address} to mock calls
      { address: await l2MessengerImpersonator.getAddress() }
    )

    // Deploy the contract under test
    L2StandardBridge = await (
      await ethers.getContractFactory('L2StandardBridge')
    ).deploy(Fake__L2CrossDomainMessenger.address, DUMMY_L1BRIDGE_ADDRESS)

    // Deploy an L2 ERC20
    L2ERC20 = await (
      await ethers.getContractFactory('L2StandardERC20', alice)
    ).deploy(L2StandardBridge.address, DUMMY_L1TOKEN_ADDRESS, 'L2Token', 'L2T')
  })

  // test the transfer flow of moving a token from L2 to L1
  describe('finalizeDeposit', () => {
    it('onlyFromCrossDomainAccount: should revert on calls from a non-crossDomainMessenger L2 account', async () => {
      await expect(
        L2StandardBridge.finalizeDeposit(
          DUMMY_L1TOKEN_ADDRESS,
          NON_ZERO_ADDRESS,
          NON_ZERO_ADDRESS,
          NON_ZERO_ADDRESS,
          0,
          NON_NULL_BYTES32
        )
      ).to.be.revertedWith(ERR_INVALID_MESSENGER)
    })

    it('onlyFromCrossDomainAccount: should revert on calls from the right crossDomainMessenger, but wrong xDomainMessageSender (ie. not the L1L1StandardBridge)', async () => {
      Fake__L2CrossDomainMessenger.xDomainMessageSender.returns(
        NON_ZERO_ADDRESS
      )

      await expect(
        L2StandardBridge.connect(l2MessengerImpersonator).finalizeDeposit(
          DUMMY_L1TOKEN_ADDRESS,
          NON_ZERO_ADDRESS,
          NON_ZERO_ADDRESS,
          NON_ZERO_ADDRESS,
          0,
          NON_NULL_BYTES32,
          {
            from: Fake__L2CrossDomainMessenger.address,
          }
        )
      ).to.be.revertedWith(ERR_INVALID_X_DOMAIN_MSG_SENDER)
    })

    it('should initialize a withdrawal if the L2 token is not compliant', async () => {
      // Deploy a non compliant ERC20
      const NonCompliantERC20 = await (
        await ethers.getContractFactory(
          '@openzeppelin/contracts/token/ERC20/ERC20.sol:ERC20'
        )
      ).deploy('L2Token', 'L2T')

      L2StandardBridge.connect(l2MessengerImpersonator).finalizeDeposit(
        DUMMY_L1TOKEN_ADDRESS,
        NON_ZERO_ADDRESS,
        NON_ZERO_ADDRESS,
        NON_ZERO_ADDRESS,
        0,
        NON_NULL_BYTES32,
        {
          from: Fake__L2CrossDomainMessenger.address,
        }
      )

      Fake__L2CrossDomainMessenger.xDomainMessageSender.returns(
        () => DUMMY_L1BRIDGE_ADDRESS
      )

      await L2StandardBridge.connect(l2MessengerImpersonator).finalizeDeposit(
        DUMMY_L1TOKEN_ADDRESS,
        NonCompliantERC20.address,
        aliceAddress,
        bobsAddress,
        100,
        NON_NULL_BYTES32,
        {
          from: Fake__L2CrossDomainMessenger.address,
        }
      )

      const withdrawalCallToMessenger =
        Fake__L2CrossDomainMessenger.sendMessage.getCall(1)

      expect(withdrawalCallToMessenger.args[0]).to.equal(DUMMY_L1BRIDGE_ADDRESS)
      expect(withdrawalCallToMessenger.args[1]).to.equal(
        Factory__L1StandardBridge.interface.encodeFunctionData(
          'finalizeERC20Withdrawal',
          [
            DUMMY_L1TOKEN_ADDRESS,
            NonCompliantERC20.address,
            bobsAddress,
            aliceAddress,
            100,
            NON_NULL_BYTES32,
          ]
        )
      )
    })

    it('should credit funds to the depositor', async () => {
      const depositAmount = 100

      Fake__L2CrossDomainMessenger.xDomainMessageSender.returns(
        () => DUMMY_L1BRIDGE_ADDRESS
      )

      await L2StandardBridge.connect(l2MessengerImpersonator).finalizeDeposit(
        DUMMY_L1TOKEN_ADDRESS,
        L2ERC20.address,
        aliceAddress,
        bobsAddress,
        depositAmount,
        NON_NULL_BYTES32,
        {
          from: Fake__L2CrossDomainMessenger.address,
        }
      )

      const bobsBalance = await L2ERC20.balanceOf(bobsAddress)
      bobsBalance.should.equal(depositAmount)
    })
  })

  describe('withdrawals', () => {
    const withdrawAmount = 1_000
    let Mock__L2Token: MockContract<Contract>

    let Fake__OVM_ETH

    before(async () => {
      Fake__OVM_ETH = await smock.fake('OVM_ETH', {
        address: OVM_ETH_ADDRESS,
      })
    })

    beforeEach(async () => {
      // Deploy a smodded gateway so we can give some balances to withdraw
      Mock__L2Token = await (
        await smock.mock('L2StandardERC20')
      ).deploy(
        L2StandardBridge.address,
        DUMMY_L1TOKEN_ADDRESS,
        'L2Token',
        'L2T'
      )

      await Mock__L2Token.setVariable('_totalSupply', INITIAL_TOTAL_SUPPLY)
      await Mock__L2Token.setVariable('_balances', {
        [aliceAddress]: ALICE_INITIAL_BALANCE,
      })
      await Mock__L2Token.setVariable('l2Bridge', L2StandardBridge.address)
    })

    it('withdraw() withdraws and sends the correct withdrawal message for OVM_ETH', async () => {
      await L2StandardBridge.withdraw(
        Fake__OVM_ETH.address,
        0,
        0,
        NON_NULL_BYTES32
      )

      const withdrawalCallToMessenger =
        Fake__L2CrossDomainMessenger.sendMessage.getCall(0)

      // Assert the correct cross-chain call was sent:
      // Message should be sent to the L1L1StandardBridge on L1
      expect(withdrawalCallToMessenger.args[0]).to.equal(DUMMY_L1BRIDGE_ADDRESS)

      // Message data should be a call telling the L1StandardBridge to finalize the withdrawal
      expect(withdrawalCallToMessenger.args[1]).to.equal(
        Factory__L1StandardBridge.interface.encodeFunctionData(
          'finalizeETHWithdrawal',
          [
            await alice.getAddress(),
            await alice.getAddress(),
            0,
            NON_NULL_BYTES32,
          ]
        )
      )
    })

    it('withdraw() burns and sends the correct withdrawal message', async () => {
      await L2StandardBridge.withdraw(
        Mock__L2Token.address,
        withdrawAmount,
        0,
        NON_NULL_BYTES32
      )
      const withdrawalCallToMessenger =
        Fake__L2CrossDomainMessenger.sendMessage.getCall(0)

      // Assert Alice's balance went down
      const aliceBalance = await Mock__L2Token.balanceOf(
        await alice.getAddress()
      )
      expect(aliceBalance).to.deep.equal(
        ethers.BigNumber.from(ALICE_INITIAL_BALANCE - withdrawAmount)
      )

      // Assert totalSupply went down
      const newTotalSupply = await Mock__L2Token.totalSupply()
      expect(newTotalSupply).to.deep.equal(
        ethers.BigNumber.from(INITIAL_TOTAL_SUPPLY - withdrawAmount)
      )

      // Assert the correct cross-chain call was sent:
      // Message should be sent to the L1L1StandardBridge on L1
      expect(withdrawalCallToMessenger.args[0]).to.equal(DUMMY_L1BRIDGE_ADDRESS)
      // Message data should be a call telling the L1L1StandardBridge to finalize the withdrawal
      expect(withdrawalCallToMessenger.args[1]).to.equal(
        Factory__L1StandardBridge.interface.encodeFunctionData(
          'finalizeERC20Withdrawal',
          [
            DUMMY_L1TOKEN_ADDRESS,
            Mock__L2Token.address,
            await alice.getAddress(),
            await alice.getAddress(),
            withdrawAmount,
            NON_NULL_BYTES32,
          ]
        )
      )
      // gaslimit should be correct
      expect(withdrawalCallToMessenger.args[2]).to.equal(0)
    })

    it('withdrawTo() burns and sends the correct withdrawal message', async () => {
      await L2StandardBridge.withdrawTo(
        Mock__L2Token.address,
        await bob.getAddress(),
        withdrawAmount,
        0,
        NON_NULL_BYTES32
      )
      const withdrawalCallToMessenger =
        Fake__L2CrossDomainMessenger.sendMessage.getCall(0)

      // Assert Alice's balance went down
      const aliceBalance = await Mock__L2Token.balanceOf(
        await alice.getAddress()
      )
      expect(aliceBalance).to.deep.equal(
        ethers.BigNumber.from(ALICE_INITIAL_BALANCE - withdrawAmount)
      )

      // Assert totalSupply went down
      const newTotalSupply = await Mock__L2Token.totalSupply()
      expect(newTotalSupply).to.deep.equal(
        ethers.BigNumber.from(INITIAL_TOTAL_SUPPLY - withdrawAmount)
      )

      // Assert the correct cross-chain call was sent.
      // Message should be sent to the L1L1StandardBridge on L1
      expect(withdrawalCallToMessenger.args[0]).to.equal(DUMMY_L1BRIDGE_ADDRESS)
      // The message data should be a call telling the L1L1StandardBridge to finalize the withdrawal
      expect(withdrawalCallToMessenger.args[1]).to.equal(
        Factory__L1StandardBridge.interface.encodeFunctionData(
          'finalizeERC20Withdrawal',
          [
            DUMMY_L1TOKEN_ADDRESS,
            Mock__L2Token.address,
            await alice.getAddress(),
            await bob.getAddress(),
            withdrawAmount,
            NON_NULL_BYTES32,
          ]
        )
      )
      // gas value is ignored and set to 0.
      expect(withdrawalCallToMessenger.args[2]).to.equal(0)
    })
  })

  describe('standard erc20', () => {
    it('should not allow anyone but the L2 bridge to mint and burn', async () => {
      expect(L2ERC20.connect(alice).mint(aliceAddress, 100)).to.be.revertedWith(
        'Only L2 Bridge can mint and burn'
      )
      expect(L2ERC20.connect(alice).burn(aliceAddress, 100)).to.be.revertedWith(
        'Only L2 Bridge can mint and burn'
      )
    })

    it('should return the correct interface support', async () => {
      const supportsERC165 = await L2ERC20.supportsInterface(0x01ffc9a7)
      expect(supportsERC165).to.be.true

      const supportsL2TokenInterface = await L2ERC20.supportsInterface(
        0x1d1d8b63
      )
      expect(supportsL2TokenInterface).to.be.true

      const badSupports = await L2ERC20.supportsInterface(0xffffffff)
      expect(badSupports).to.be.false
    })
  })
})
