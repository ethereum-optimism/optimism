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

describe('OVM_L2TokenGateway', () => {
  let alice: Signer
  let bob: Signer
  let Factory__OVM_L1ERC20Gateway: ContractFactory
  before(async () => {
    ;[alice, bob] = await ethers.getSigners()
    Factory__OVM_L1ERC20Gateway = await ethers.getContractFactory(
      'OVM_L1ERC20Gateway'
    )
  })

  let OVM_L2TokenGateway: Contract
  let OVM_L2ERC20: Contract
  let Mock__OVM_L2CrossDomainMessenger: MockContract
  let finalizeInboundTransferGasLimit: number
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
    OVM_L2TokenGateway = await (
      await ethers.getContractFactory('OVM_L2TokenGateway')
    ).deploy(
      Mock__OVM_L2CrossDomainMessenger.address,
      constants.AddressZero,
      'ovmWETH',
      'oWETH'
    )

    // Get the address of the token contract created
    OVM_L2ERC20 = await ethers.getContractAt(
      'OVM_L2ERC20',
      await OVM_L2TokenGateway.token()
    )

    // initialize the L2 Gateway with the L1Gateway address
    await OVM_L2TokenGateway.init(MOCK_L1GATEWAY_ADDRESS)

    finalizeInboundTransferGasLimit = await OVM_L2TokenGateway.getFinalizationGas()
  })

  // test the transfer flow of moving a token from L2 to L1
  describe('finalizeDeposit', () => {
    it('onlyFromCrossDomainAccount: should revert on calls from a non-crossDomainMessenger L2 account', async () => {
      // Deploy new gateway, initialize with random messenger
      OVM_L2TokenGateway = await (
        await ethers.getContractFactory('OVM_L2TokenGateway')
      ).deploy(NON_ZERO_ADDRESS, constants.AddressZero, 'ovmWETH', 'oWETH')
      await OVM_L2TokenGateway.init(NON_ZERO_ADDRESS)

      await expect(
        OVM_L2TokenGateway.finalizeInboundTransfer(
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
        OVM_L2TokenGateway.finalizeInboundTransfer(
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

      await OVM_L2TokenGateway.finalizeInboundTransfer(
        NON_ZERO_ADDRESS,
        await alice.getAddress(),
        depositAmount,
        NON_NULL_BYTES32,
        { from: Mock__OVM_L2CrossDomainMessenger.address }
      )

      const aliceBalance = await OVM_L2ERC20.balanceOf(await alice.getAddress())
      aliceBalance.should.equal(depositAmount)
    })
  })

  describe('withdrawals', () => {
    const INITIAL_TOTAL_SUPPLY = 100_000
    const ALICE_INITIAL_BALANCE = 50_000
    const withdrawAmount = 1_000
    let SmoddedL2ERC20: ModifiableContract
    beforeEach(async () => {
      // Deploy a smodded ERC20 so we can give some balances to withdraw
      SmoddedL2ERC20 = await (await smoddit('OVM_L2ERC20', alice)).deploy(
        'ovmWETH',
        'oWETH'
      )

      // Deploy the gateway
      OVM_L2TokenGateway = await (
        await ethers.getContractFactory('OVM_L2TokenGateway')
      ).deploy(
        Mock__OVM_L2CrossDomainMessenger.address,
        SmoddedL2ERC20.address,
        '',
        ''
      )
      await OVM_L2TokenGateway.init(MOCK_L1GATEWAY_ADDRESS)

      // Setup the token with a total supply and some money in alice's balance,
      // and make it owned by the L2 gateway.
      const aliceAddress = await alice.getAddress()
      SmoddedL2ERC20.smodify.put({
        _totalSupply: INITIAL_TOTAL_SUPPLY,
        _balances: {
          [aliceAddress]: ALICE_INITIAL_BALANCE,
        },
        _owner: OVM_L2TokenGateway.address,
      })
    })

    it('outboundTransfer() burns and sends the correct withdrawal message', async () => {
      await OVM_L2TokenGateway.outboundTransfer(
        withdrawAmount,
        NON_NULL_BYTES32
      )
      const withdrawalCallToMessenger =
        Mock__OVM_L2CrossDomainMessenger.smocked.sendMessage.calls[0]

      // Assert Alice's balance went down
      const aliceBalance = await SmoddedL2ERC20.balanceOf(
        await alice.getAddress()
      )
      expect(aliceBalance).to.deep.equal(
        ethers.BigNumber.from(ALICE_INITIAL_BALANCE - withdrawAmount)
      )

      // Assert totalSupply went down
      const newTotalSupply = await SmoddedL2ERC20.totalSupply()
      expect(newTotalSupply).to.deep.equal(
        ethers.BigNumber.from(INITIAL_TOTAL_SUPPLY - withdrawAmount)
      )

      // Assert the correct cross-chain call was sent:
      // Message should be sent to the L1ERC20Gateway on L1
      expect(withdrawalCallToMessenger._target).to.equal(MOCK_L1GATEWAY_ADDRESS)
      // Message data should be a call telling the L1ERC20Gateway to finalize the withdrawal
      expect(withdrawalCallToMessenger._message).to.equal(
        await Factory__OVM_L1ERC20Gateway.interface.encodeFunctionData(
          'finalizeInboundTransfer',
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
        finalizeInboundTransferGasLimit
      )
    })

    it('outboundTransferTo() burns and sends the correct withdrawal message', async () => {
      await OVM_L2TokenGateway.outboundTransferTo(
        await bob.getAddress(),
        withdrawAmount,
        NON_NULL_BYTES32
      )
      const withdrawalCallToMessenger =
        Mock__OVM_L2CrossDomainMessenger.smocked.sendMessage.calls[0]

      // Assert Alice's balance went down
      const aliceBalance = await SmoddedL2ERC20.balanceOf(
        await alice.getAddress()
      )
      expect(aliceBalance).to.deep.equal(
        ethers.BigNumber.from(ALICE_INITIAL_BALANCE - withdrawAmount)
      )

      // Assert totalSupply went down
      const newTotalSupply = await SmoddedL2ERC20.totalSupply()
      expect(newTotalSupply).to.deep.equal(
        ethers.BigNumber.from(INITIAL_TOTAL_SUPPLY - withdrawAmount)
      )

      // Assert the correct cross-chain call was sent.
      // Message should be sent to the L1ERC20Gateway on L1
      expect(withdrawalCallToMessenger._target).to.equal(MOCK_L1GATEWAY_ADDRESS)
      // The message data should be a call telling the L1ERC20Gateway to finalize the withdrawal
      expect(withdrawalCallToMessenger._message).to.equal(
        await Factory__OVM_L1ERC20Gateway.interface.encodeFunctionData(
          'finalizeInboundTransfer',
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
        finalizeInboundTransferGasLimit
      )
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
