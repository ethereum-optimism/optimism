import { expect } from '../../../../setup'

/* External Imports */
import { ethers } from 'hardhat'
import { Signer, ContractFactory, Contract, constants } from 'ethers'
import {
  smockit,
  MockContract,
  smoddit,
  ModifiableContract,
} from '@eth-optimism/smock'

/* Internal Imports */
import { NON_NULL_BYTES32, NON_ZERO_ADDRESS } from '../../../../helpers'

const ERR_INVALID_MESSENGER = 'OVM_XCHAIN: messenger contract unauthenticated'
const ERR_INVALID_X_DOMAIN_MSG_SENDER =
  'OVM_XCHAIN: wrong sender of cross-domain message'
const MOCK_L1GATEWAY_ADDRESS: string =
  '0x1234123412341234123412341234123412341234'

describe('OVM_L2DepositedERC20', () => {
  let alice: Signer
  let bob: Signer
  let Factory__OVM_L1ERC20Gateway: ContractFactory
  before(async () => {
    ;[alice, bob] = await ethers.getSigners()
    Factory__OVM_L1ERC20Gateway = await ethers.getContractFactory(
      'OVM_L1ERC20Gateway'
    )
  })

  let OVM_L2DepositedERC20: Contract
  let Mock__OVM_L2CrossDomainMessenger: MockContract
  let finalizeWithdrawalGasLimit: number
  beforeEach(async () => {
    // Create a special signer which will enable us to send messages from the L2Messenger contract
    let l2MessengerImpersonator: Signer
    ;[l2MessengerImpersonator] = await ethers.getSigners()

    // Get a new mock L2 messenger
    Mock__OVM_L2CrossDomainMessenger = await smockit(
      await ethers.getContractFactory('OVM_L2CrossDomainMessenger'),
      // This allows us to use an ethers override {from: Mock__OVM_L2CrossDomainMessenger.address} to mock calls
      { address: await l2MessengerImpersonator.getAddress() }
    )

    // Deploy the contract under test
    OVM_L2DepositedERC20 = await (
      await ethers.getContractFactory('OVM_L2DepositedERC20')
    ).deploy(Mock__OVM_L2CrossDomainMessenger.address, 'ovmWETH', 'oWETH')

    // initialize the L2 Gateway with the L1G ateway addrss
    await OVM_L2DepositedERC20.init(MOCK_L1GATEWAY_ADDRESS)

    finalizeWithdrawalGasLimit = await OVM_L2DepositedERC20.getFinalizeWithdrawalL1Gas()
  })

  // test the transfer flow of moving a token from L2 to L1
  describe('finalizeDeposit', () => {
    it('onlyFromCrossDomainAccount: should revert on calls from a non-crossDomainMessenger L2 account', async () => {
      // Deploy new gateway, initialize with random messenger
      OVM_L2DepositedERC20 = await (
        await ethers.getContractFactory('OVM_L2DepositedERC20')
      ).deploy(NON_ZERO_ADDRESS, 'ovmWETH', 'oWETH')
      await OVM_L2DepositedERC20.init(NON_ZERO_ADDRESS)

      await expect(
        OVM_L2DepositedERC20.finalizeDeposit(
          constants.AddressZero,
          constants.AddressZero,
          0,
          NON_NULL_BYTES32
        )
      ).to.be.revertedWith(ERR_INVALID_MESSENGER)
    })

    it('onlyFromCrossDomainAccount: should revert on calls from the right crossDomainMessenger, but wrong xDomainMessageSender (ie. not the L1ERC20Gateway)', async () => {
      Mock__OVM_L2CrossDomainMessenger.smocked.xDomainMessageSender.will.return.with(
        NON_ZERO_ADDRESS
      )

      await expect(
        OVM_L2DepositedERC20.finalizeDeposit(
          constants.AddressZero,
          constants.AddressZero,
          0,
          NON_NULL_BYTES32,
          {
            from: Mock__OVM_L2CrossDomainMessenger.address,
          }
        )
      ).to.be.revertedWith(ERR_INVALID_X_DOMAIN_MSG_SENDER)
    })

    it('should credit funds to the depositor', async () => {
      const depositAmount = 100
      Mock__OVM_L2CrossDomainMessenger.smocked.xDomainMessageSender.will.return.with(
        () => MOCK_L1GATEWAY_ADDRESS
      )

      await OVM_L2DepositedERC20.finalizeDeposit(
        NON_ZERO_ADDRESS,
        await alice.getAddress(),
        depositAmount,
        NON_NULL_BYTES32,
        { from: Mock__OVM_L2CrossDomainMessenger.address }
      )

      const aliceBalance = await OVM_L2DepositedERC20.balanceOf(
        await alice.getAddress()
      )
      aliceBalance.should.equal(depositAmount)
    })
  })

  describe('withdrawals', () => {
    const INITIAL_TOTAL_SUPPLY = 100_000
    const ALICE_INITIAL_BALANCE = 50_000
    const withdrawAmount = 1_000
    let SmoddedL2Gateway: ModifiableContract
    beforeEach(async () => {
      // Deploy a smodded gateway so we can give some balances to withdraw
      SmoddedL2Gateway = await (
        await smoddit('OVM_L2DepositedERC20', alice)
      ).deploy(Mock__OVM_L2CrossDomainMessenger.address, 'ovmWETH', 'oWETH')
      await SmoddedL2Gateway.init(MOCK_L1GATEWAY_ADDRESS)

      // Populate the initial state with a total supply and some money in alice's balance
      const aliceAddress = await alice.getAddress()
      SmoddedL2Gateway.smodify.put({
        totalSupply: INITIAL_TOTAL_SUPPLY,
        balanceOf: {
          [aliceAddress]: ALICE_INITIAL_BALANCE,
        },
      })
    })

    it('withdraw() burns and sends the correct withdrawal message', async () => {
      await SmoddedL2Gateway.withdraw(withdrawAmount, 0, NON_NULL_BYTES32)
      const withdrawalCallToMessenger =
        Mock__OVM_L2CrossDomainMessenger.smocked.sendMessage.calls[0]

      // Assert Alice's balance went down
      const aliceBalance = await SmoddedL2Gateway.balanceOf(
        await alice.getAddress()
      )
      expect(aliceBalance).to.deep.equal(
        ethers.BigNumber.from(ALICE_INITIAL_BALANCE - withdrawAmount)
      )

      // Assert totalSupply went down
      const newTotalSupply = await SmoddedL2Gateway.totalSupply()
      expect(newTotalSupply).to.deep.equal(
        ethers.BigNumber.from(INITIAL_TOTAL_SUPPLY - withdrawAmount)
      )

      // Assert the correct cross-chain call was sent:
      // Message should be sent to the L1ERC20Gateway on L1
      expect(withdrawalCallToMessenger._target).to.equal(MOCK_L1GATEWAY_ADDRESS)
      // Message data should be a call telling the L1ERC20Gateway to finalize the withdrawal
      expect(withdrawalCallToMessenger._message).to.equal(
        await Factory__OVM_L1ERC20Gateway.interface.encodeFunctionData(
          'finalizeWithdrawal',
          [
            await alice.getAddress(),
            await alice.getAddress(),
            withdrawAmount,
            NON_NULL_BYTES32,
          ]
        )
      )
      // Hardcoded gaslimit should be correct
      expect(withdrawalCallToMessenger._gasLimit).to.equal(
        finalizeWithdrawalGasLimit
      )
    })

    it('withdraw() uses the user provided gas limit if it is larger than the default value ', async () => {
      const customGasLimit = 10_000_000
      await SmoddedL2Gateway.withdraw(
        withdrawAmount,
        customGasLimit,
        NON_NULL_BYTES32,
      )
      const withdrawalCallToMessenger =
        Mock__OVM_L2CrossDomainMessenger.smocked.sendMessage.calls[0]
      expect(withdrawalCallToMessenger._gasLimit).to.equal(customGasLimit)
    })

    it('withdrawTo() burns and sends the correct withdrawal message', async () => {
      await SmoddedL2Gateway.withdrawTo(
        await bob.getAddress(),
        withdrawAmount,
        0,
        NON_NULL_BYTES32,
      )
      const withdrawalCallToMessenger =
        Mock__OVM_L2CrossDomainMessenger.smocked.sendMessage.calls[0]

      // Assert Alice's balance went down
      const aliceBalance = await SmoddedL2Gateway.balanceOf(
        await alice.getAddress()
      )
      expect(aliceBalance).to.deep.equal(
        ethers.BigNumber.from(ALICE_INITIAL_BALANCE - withdrawAmount)
      )

      // Assert totalSupply went down
      const newTotalSupply = await SmoddedL2Gateway.totalSupply()
      expect(newTotalSupply).to.deep.equal(
        ethers.BigNumber.from(INITIAL_TOTAL_SUPPLY - withdrawAmount)
      )

      // Assert the correct cross-chain call was sent.
      // Message should be sent to the L1ERC20Gateway on L1
      expect(withdrawalCallToMessenger._target).to.equal(MOCK_L1GATEWAY_ADDRESS)
      // The message data should be a call telling the L1ERC20Gateway to finalize the withdrawal
      expect(withdrawalCallToMessenger._message).to.equal(
        await Factory__OVM_L1ERC20Gateway.interface.encodeFunctionData(
          'finalizeWithdrawal',
          [
            await alice.getAddress(),
            await bob.getAddress(),
            withdrawAmount,
            NON_NULL_BYTES32,
          ]
        )
      )
      // Hardcoded gaslimit should be correct
      expect(withdrawalCallToMessenger._gasLimit).to.equal(
        finalizeWithdrawalGasLimit
      )
    })

    it('withdrawTo() uses the user provided gas limit if it is larger than the default value ', async () => {
      const customGasLimit = 10_000_000
      await SmoddedL2Gateway.withdrawTo(
        await bob.getAddress(),
        withdrawAmount,
        customGasLimit,
        NON_NULL_BYTES32
      )
      const withdrawalCallToMessenger =
        Mock__OVM_L2CrossDomainMessenger.smocked.sendMessage.calls[0]
      expect(withdrawalCallToMessenger._gasLimit).to.equal(customGasLimit)
    })
  })

  // low priority todos: see question in contract
  describe.skip('Initialization logic', () => {
    it('should not allow calls to onlyInitialized functions', async () => {
      // TODO
    })

    it('should only allow initialization once and emits initialized event', async () => {
      // TODO
    })
  })
})
