import { expect } from '../../../../setup'

/* External Imports */
import { ethers } from 'hardhat'
import { Signer, ContractFactory, Contract } from 'ethers'
import { Interface } from 'ethers/lib/utils'
import {
  smockit,
  MockContract,
  smoddit,
  ModifiableContract,
} from '@eth-optimism/smock'

/* Internal Imports */
import { NON_NULL_BYTES32, NON_ZERO_ADDRESS } from '../../../../helpers'

import { getContractInterface } from '../../../../../src'

const ERR_INVALID_MESSENGER = 'OVM_XCHAIN: messenger contract unauthenticated'
const ERR_INVALID_X_DOMAIN_MSG_SENDER =
  'OVM_XCHAIN: wrong sender of cross-domain message'
const DUMMY_L1BRIDGE_ADDRESS: string =
  '0x1234123412341234123412341234123412341234'
const DUMMY_L1TOKEN_ADDRESS: string =
  '0x2234223412342234223422342234223422342234'

describe('OVM_L2StandardBridge', () => {
  let alice: Signer
  let aliceAddress: string
  let bob: Signer
  let bobsAddress: string
  let l2MessengerImpersonator: Signer
  let Factory__OVM_L1StandardBridge: ContractFactory
  let IL2ERC20Bridge: Interface
  const INITIAL_TOTAL_SUPPLY = 100_000
  const ALICE_INITIAL_BALANCE = 50_000
  before(async () => {
    // Create a special signer which will enable us to send messages from the L2Messenger contract
    ;[alice, bob, l2MessengerImpersonator] = await ethers.getSigners()
    aliceAddress = await alice.getAddress()
    bobsAddress = await bob.getAddress()
    Factory__OVM_L1StandardBridge = await ethers.getContractFactory(
      'OVM_L1StandardBridge'
    )

    // get an L2ER20Bridge Interface
    IL2ERC20Bridge = getContractInterface('iOVM_L2ERC20Bridge')
  })

  let OVM_L2StandardBridge: Contract
  let L2ERC20: Contract
  let Mock__OVM_L2CrossDomainMessenger: MockContract
  beforeEach(async () => {
    // Get a new mock L2 messenger
    Mock__OVM_L2CrossDomainMessenger = await smockit(
      await ethers.getContractFactory('OVM_L2CrossDomainMessenger'),
      // This allows us to use an ethers override {from: Mock__OVM_L2CrossDomainMessenger.address} to mock calls
      { address: await l2MessengerImpersonator.getAddress() }
    )

    // Deploy the contract under test
    OVM_L2StandardBridge = await (
      await ethers.getContractFactory('OVM_L2StandardBridge')
    ).deploy(Mock__OVM_L2CrossDomainMessenger.address, DUMMY_L1BRIDGE_ADDRESS)

    // Deploy an L2 ERC20
    L2ERC20 = await (
      await ethers.getContractFactory('L2StandardERC20', alice)
    ).deploy(
      OVM_L2StandardBridge.address,
      DUMMY_L1TOKEN_ADDRESS,
      'L2Token',
      'L2T'
    )
  })

  // test the transfer flow of moving a token from L2 to L1
  describe('finalizeDeposit', () => {
    it('onlyFromCrossDomainAccount: should revert on calls from a non-crossDomainMessenger L2 account', async () => {
      await expect(
        OVM_L2StandardBridge.finalizeDeposit(
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
      Mock__OVM_L2CrossDomainMessenger.smocked.xDomainMessageSender.will.return.with(
        NON_ZERO_ADDRESS
      )

      await expect(
        OVM_L2StandardBridge.connect(l2MessengerImpersonator).finalizeDeposit(
          DUMMY_L1TOKEN_ADDRESS,
          NON_ZERO_ADDRESS,
          NON_ZERO_ADDRESS,
          NON_ZERO_ADDRESS,
          0,
          NON_NULL_BYTES32,
          {
            from: Mock__OVM_L2CrossDomainMessenger.address,
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

      OVM_L2StandardBridge.connect(l2MessengerImpersonator).finalizeDeposit(
        DUMMY_L1TOKEN_ADDRESS,
        NON_ZERO_ADDRESS,
        NON_ZERO_ADDRESS,
        NON_ZERO_ADDRESS,
        0,
        NON_NULL_BYTES32,
        {
          from: Mock__OVM_L2CrossDomainMessenger.address,
        }
      )

      Mock__OVM_L2CrossDomainMessenger.smocked.xDomainMessageSender.will.return.with(
        () => DUMMY_L1BRIDGE_ADDRESS
      )

      await OVM_L2StandardBridge.connect(
        l2MessengerImpersonator
      ).finalizeDeposit(
        DUMMY_L1TOKEN_ADDRESS,
        NonCompliantERC20.address,
        aliceAddress,
        bobsAddress,
        100,
        NON_NULL_BYTES32,
        {
          from: Mock__OVM_L2CrossDomainMessenger.address,
        }
      )

      const withdrawalCallToMessenger =
        Mock__OVM_L2CrossDomainMessenger.smocked.sendMessage.calls[0]

      expect(withdrawalCallToMessenger._target).to.equal(DUMMY_L1BRIDGE_ADDRESS)
      expect(withdrawalCallToMessenger._message).to.equal(
        Factory__OVM_L1StandardBridge.interface.encodeFunctionData(
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

      Mock__OVM_L2CrossDomainMessenger.smocked.xDomainMessageSender.will.return.with(
        () => DUMMY_L1BRIDGE_ADDRESS
      )

      await OVM_L2StandardBridge.connect(
        l2MessengerImpersonator
      ).finalizeDeposit(
        DUMMY_L1TOKEN_ADDRESS,
        L2ERC20.address,
        aliceAddress,
        bobsAddress,
        depositAmount,
        NON_NULL_BYTES32,
        {
          from: Mock__OVM_L2CrossDomainMessenger.address,
        }
      )

      const bobsBalance = await L2ERC20.balanceOf(bobsAddress)
      bobsBalance.should.equal(depositAmount)
    })
  })

  describe('withdrawals', () => {
    const withdrawAmount = 1_000
    let SmoddedL2Token: ModifiableContract
    beforeEach(async () => {
      // Deploy a smodded gateway so we can give some balances to withdraw
      SmoddedL2Token = await (await smoddit('L2StandardERC20', alice)).deploy(
        OVM_L2StandardBridge.address,
        DUMMY_L1TOKEN_ADDRESS,
        'L2Token',
        'L2T'
      )

      // Populate the initial state with a total supply and some money in alice's balance
      SmoddedL2Token.smodify.put({
        _totalSupply: INITIAL_TOTAL_SUPPLY,
        _balances: {
          [aliceAddress]: ALICE_INITIAL_BALANCE,
        },
        l2Bridge: OVM_L2StandardBridge.address,
      })
    })

    it('withdraw() burns and sends the correct withdrawal message', async () => {
      await OVM_L2StandardBridge.withdraw(
        SmoddedL2Token.address,
        withdrawAmount,
        0,
        NON_NULL_BYTES32
      )
      const withdrawalCallToMessenger =
        Mock__OVM_L2CrossDomainMessenger.smocked.sendMessage.calls[0]

      // Assert Alice's balance went down
      const aliceBalance = await SmoddedL2Token.balanceOf(
        await alice.getAddress()
      )
      expect(aliceBalance).to.deep.equal(
        ethers.BigNumber.from(ALICE_INITIAL_BALANCE - withdrawAmount)
      )

      // Assert totalSupply went down
      const newTotalSupply = await SmoddedL2Token.totalSupply()
      expect(newTotalSupply).to.deep.equal(
        ethers.BigNumber.from(INITIAL_TOTAL_SUPPLY - withdrawAmount)
      )

      // Assert the correct cross-chain call was sent:
      // Message should be sent to the L1L1StandardBridge on L1
      expect(withdrawalCallToMessenger._target).to.equal(DUMMY_L1BRIDGE_ADDRESS)
      // Message data should be a call telling the L1L1StandardBridge to finalize the withdrawal
      expect(withdrawalCallToMessenger._message).to.equal(
        Factory__OVM_L1StandardBridge.interface.encodeFunctionData(
          'finalizeERC20Withdrawal',
          [
            DUMMY_L1TOKEN_ADDRESS,
            SmoddedL2Token.address,
            await alice.getAddress(),
            await alice.getAddress(),
            withdrawAmount,
            NON_NULL_BYTES32,
          ]
        )
      )
      // gaslimit should be correct
      expect(withdrawalCallToMessenger._gasLimit).to.equal(0)
    })

    it('withdrawTo() burns and sends the correct withdrawal message', async () => {
      await OVM_L2StandardBridge.withdrawTo(
        SmoddedL2Token.address,
        await bob.getAddress(),
        withdrawAmount,
        0,
        NON_NULL_BYTES32
      )
      const withdrawalCallToMessenger =
        Mock__OVM_L2CrossDomainMessenger.smocked.sendMessage.calls[0]

      // Assert Alice's balance went down
      const aliceBalance = await SmoddedL2Token.balanceOf(
        await alice.getAddress()
      )
      expect(aliceBalance).to.deep.equal(
        ethers.BigNumber.from(ALICE_INITIAL_BALANCE - withdrawAmount)
      )

      // Assert totalSupply went down
      const newTotalSupply = await SmoddedL2Token.totalSupply()
      expect(newTotalSupply).to.deep.equal(
        ethers.BigNumber.from(INITIAL_TOTAL_SUPPLY - withdrawAmount)
      )

      // Assert the correct cross-chain call was sent.
      // Message should be sent to the L1L1StandardBridge on L1
      expect(withdrawalCallToMessenger._target).to.equal(DUMMY_L1BRIDGE_ADDRESS)
      // The message data should be a call telling the L1L1StandardBridge to finalize the withdrawal
      expect(withdrawalCallToMessenger._message).to.equal(
        Factory__OVM_L1StandardBridge.interface.encodeFunctionData(
          'finalizeERC20Withdrawal',
          [
            DUMMY_L1TOKEN_ADDRESS,
            SmoddedL2Token.address,
            await alice.getAddress(),
            await bob.getAddress(),
            withdrawAmount,
            NON_NULL_BYTES32,
          ]
        )
      )
      // gas value is ignored and set to 0.
      expect(withdrawalCallToMessenger._gasLimit).to.equal(0)
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
