import { expect } from '../../../../setup'

/* External Imports */
import { ethers } from 'hardhat'
import { Signer, ContractFactory, Contract, constants } from 'ethers'
import { smockit, MockContract, smoddit } from '@eth-optimism/smock'

/* Internal Imports */
import { NON_NULL_BYTES32, NON_ZERO_ADDRESS } from '../../../../helpers'

const INITIAL_TOTAL_L1_SUPPLY = 3000

const ERR_INVALID_MESSENGER = 'OVM_XCHAIN: messenger contract unauthenticated'
const ERR_INVALID_X_DOMAIN_MSG_SENDER =
  'OVM_XCHAIN: wrong sender of cross-domain message'

describe('OVM_L1ERC20Gateway', () => {
  // init signers
  let alice: Signer
  let bob: Signer

  // we can just make up this string since it's on the "other" Layer
  let Mock__OVM_L2DepositedERC20: MockContract
  let Factory__L1ERC20: ContractFactory
  let L1ERC20: Contract
  before(async () => {
    ;[alice, bob] = await ethers.getSigners()

    Mock__OVM_L2DepositedERC20 = await smockit(
      await ethers.getContractFactory('OVM_L2DepositedERC20')
    )

    // deploy an ERC20 contract on L1
    Factory__L1ERC20 = await smoddit('UniswapV2ERC20')

    L1ERC20 = await Factory__L1ERC20.deploy('L1ERC20', 'ERC')

    const aliceAddress = await alice.getAddress()
    await L1ERC20.smodify.put({
      totalSupply: INITIAL_TOTAL_L1_SUPPLY,
      balanceOf: {
        [aliceAddress]: INITIAL_TOTAL_L1_SUPPLY,
      },
    })
  })

  let OVM_L1ERC20Gateway: Contract
  let Mock__OVM_L1CrossDomainMessenger: MockContract
  let finalizeDepositGasLimit: number
  beforeEach(async () => {
    // Create a special signer which will enable us to send messages from the L1Messenger contract
    let l1MessengerImpersonator: Signer
    ;[l1MessengerImpersonator, alice, bob] = await ethers.getSigners()
    // Get a new mock L1 messenger
    Mock__OVM_L1CrossDomainMessenger = await smockit(
      await ethers.getContractFactory('OVM_L1CrossDomainMessenger'),
      { address: await l1MessengerImpersonator.getAddress() } // This allows us to use an ethers override {from: Mock__OVM_L2CrossDomainMessenger.address} to mock calls
    )

    // Deploy the contract under test
    OVM_L1ERC20Gateway = await (
      await ethers.getContractFactory('OVM_L1ERC20Gateway')
    ).deploy(
      L1ERC20.address,
      Mock__OVM_L2DepositedERC20.address,
      Mock__OVM_L1CrossDomainMessenger.address
    )

    finalizeDepositGasLimit = await OVM_L1ERC20Gateway.getFinalizeDepositL2Gas()
  })

  describe('finalizeWithdrawal', () => {
    it('onlyFromCrossDomainAccount: should revert on calls from a non-crossDomainMessenger L1 account', async () => {
      // Deploy new gateway, initialize with random messenger
      OVM_L1ERC20Gateway = await (
        await ethers.getContractFactory('OVM_L1ERC20Gateway')
      ).deploy(
        L1ERC20.address,
        Mock__OVM_L2DepositedERC20.address,
        NON_ZERO_ADDRESS
      )

      await expect(
        OVM_L1ERC20Gateway.finalizeWithdrawal(
          constants.AddressZero,
          constants.AddressZero,
          1,
          NON_NULL_BYTES32
        )
      ).to.be.revertedWith(ERR_INVALID_MESSENGER)
    })

    it('onlyFromCrossDomainAccount: should revert on calls from the right crossDomainMessenger, but wrong xDomainMessageSender (ie. not the L2ERC20Gateway)', async () => {
      Mock__OVM_L1CrossDomainMessenger.smocked.xDomainMessageSender.will.return.with(
        () => NON_ZERO_ADDRESS
      )

      await expect(
        OVM_L1ERC20Gateway.finalizeWithdrawal(
          constants.AddressZero,
          constants.AddressZero,
          1,
          NON_NULL_BYTES32,
          {
            from: Mock__OVM_L1CrossDomainMessenger.address,
          }
        )
      ).to.be.revertedWith(ERR_INVALID_X_DOMAIN_MSG_SENDER)
    })

    it('should credit funds to the withdrawer and not use too much gas', async () => {
      // make sure no balance at start of test
      await expect(await L1ERC20.balanceOf(NON_ZERO_ADDRESS)).to.be.equal(0)

      const withdrawalAmount = 100
      Mock__OVM_L1CrossDomainMessenger.smocked.xDomainMessageSender.will.return.with(
        () => Mock__OVM_L2DepositedERC20.address
      )

      await L1ERC20.transfer(OVM_L1ERC20Gateway.address, withdrawalAmount)

      const res = await OVM_L1ERC20Gateway.finalizeWithdrawal(
        NON_ZERO_ADDRESS,
        NON_ZERO_ADDRESS,
        withdrawalAmount,
        NON_NULL_BYTES32,
        { from: Mock__OVM_L1CrossDomainMessenger.address }
      )

      await expect(await L1ERC20.balanceOf(NON_ZERO_ADDRESS)).to.be.equal(
        withdrawalAmount
      )

      const gasUsed = (
        await OVM_L1ERC20Gateway.provider.getTransactionReceipt(res.hash)
      ).gasUsed

      const OVM_L2DepositedERC20 = await (
        await ethers.getContractFactory('OVM_L2DepositedERC20')
      ).deploy(constants.AddressZero, '', '')
      const defaultFinalizeWithdrawalGas = await OVM_L2DepositedERC20.getFinalizeWithdrawalL1Gas()
      await expect(gasUsed.gt((defaultFinalizeWithdrawalGas * 11) / 10))
    })

    it.skip('finalizeWithdrawalAndCall(): should should credit funds to the withdrawer, and forward from and data', async () => {
      // TODO: implement this functionality in a future update
      expect.fail()
    })
  })

  describe('deposits', () => {
    const INITIAL_DEPOSITER_BALANCE = 100_000
    let depositer: string
    const depositAmount = 1_000

    beforeEach(async () => {
      // Deploy the L1 ERC20 token, Alice will receive the full initialSupply
      L1ERC20 = await Factory__L1ERC20.deploy('L1ERC20', 'ERC')

      // get a new mock L1 messenger
      Mock__OVM_L1CrossDomainMessenger = await smockit(
        await ethers.getContractFactory('OVM_L1CrossDomainMessenger')
      )

      // Deploy the contract under test:
      OVM_L1ERC20Gateway = await (
        await ethers.getContractFactory('OVM_L1ERC20Gateway')
      ).deploy(
        L1ERC20.address,
        Mock__OVM_L2DepositedERC20.address,
        Mock__OVM_L1CrossDomainMessenger.address
      )

      // the Signer sets approve for the L1 Gateway
      await L1ERC20.approve(OVM_L1ERC20Gateway.address, depositAmount)
      depositer = await L1ERC20.signer.getAddress()

      await L1ERC20.smodify.put({
        balanceOf: {
          [depositer]: INITIAL_DEPOSITER_BALANCE,
        },
      })
    })

    it('deposit() escrows the deposit amount and sends the correct deposit message', async () => {
      // alice calls deposit on the gateway and the L1 gateway calls transferFrom on the token
      await OVM_L1ERC20Gateway.deposit(depositAmount, NON_NULL_BYTES32)
      const depositCallToMessenger =
        Mock__OVM_L1CrossDomainMessenger.smocked.sendMessage.calls[0]

      const depositerBalance = await L1ERC20.balanceOf(depositer)
      expect(depositerBalance).to.equal(
        INITIAL_DEPOSITER_BALANCE - depositAmount
      )

      // gateway's balance is increased
      const gatewayBalance = await L1ERC20.balanceOf(OVM_L1ERC20Gateway.address)
      expect(gatewayBalance).to.equal(depositAmount)

      // Check the correct cross-chain call was sent:
      // Message should be sent to the L2ERC20Gateway on L2
      expect(depositCallToMessenger._target).to.equal(
        Mock__OVM_L2DepositedERC20.address
      )
      // Message data should be a call telling the L2ERC20Gateway to finalize the deposit

      // the L1 gateway sends the correct message to the L1 messenger
      expect(depositCallToMessenger._message).to.equal(
        await Mock__OVM_L2DepositedERC20.interface.encodeFunctionData(
          'finalizeDeposit',
          [depositer, depositer, depositAmount, NON_NULL_BYTES32]
        )
      )
      expect(depositCallToMessenger._gasLimit).to.equal(finalizeDepositGasLimit)
    })

    it('depositTo() escrows the deposit amount and sends the correct deposit message', async () => {
      // depositor calls deposit on the gateway and the L1 gateway calls transferFrom on the token
      const bobsAddress = await bob.getAddress()
      await OVM_L1ERC20Gateway.depositTo(
        bobsAddress,
        depositAmount,
        NON_NULL_BYTES32
      )
      const depositCallToMessenger =
        Mock__OVM_L1CrossDomainMessenger.smocked.sendMessage.calls[0]

      const depositerBalance = await L1ERC20.balanceOf(depositer)
      expect(depositerBalance).to.equal(
        INITIAL_DEPOSITER_BALANCE - depositAmount
      )

      // gateway's balance is increased
      const gatewayBalance = await L1ERC20.balanceOf(OVM_L1ERC20Gateway.address)
      expect(gatewayBalance).to.equal(depositAmount)

      // Check the correct cross-chain call was sent:
      // Message should be sent to the L2ERC20Gateway on L2
      expect(depositCallToMessenger._target).to.equal(
        Mock__OVM_L2DepositedERC20.address
      )
      // Message data should be a call telling the L2ERC20Gateway to finalize the deposit

      // the L1 gateway sends the correct message to the L1 messenger
      expect(depositCallToMessenger._message).to.equal(
        await Mock__OVM_L2DepositedERC20.interface.encodeFunctionData(
          'finalizeDeposit',
          [depositer, bobsAddress, depositAmount, NON_NULL_BYTES32]
        )
      )
      expect(depositCallToMessenger._gasLimit).to.equal(finalizeDepositGasLimit)
    })
  })
})
