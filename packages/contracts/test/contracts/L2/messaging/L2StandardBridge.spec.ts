import { ethers } from 'hardhat'
import { Contract } from 'ethers'
import { smock, FakeContract, MockContract } from '@defi-wonderland/smock'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'

import { expect } from '../../../setup'
import { deploy, NON_NULL_BYTES32, NON_ZERO_ADDRESS } from '../../../helpers'
import { getContractInterface, predeploys } from '../../../../src'

// TODO: Maybe we should consider automatically generating these and exporting them?
const ERROR_STRINGS = {
  INVALID_MESSENGER: 'OVM_XCHAIN: messenger contract unauthenticated',
  INVALID_X_DOMAIN_MSG_SENDER:
    'OVM_XCHAIN: wrong sender of cross-domain message',
}

const DUMMY_L1_ERC20_ADDRESS = '0xaBBAABbaaBbAABbaABbAABbAABbaAbbaaBbaaBBa'
const DUMMY_L1_BRIDGE_ADDRESS = '0xACDCacDcACdCaCDcacdcacdCaCdcACdCAcDcaCdc'

describe('L2StandardBridge', () => {
  const INITIAL_TOTAL_SUPPLY = 100_000
  const ALICE_INITIAL_BALANCE = 50_000

  let alice: SignerWithAddress
  let bob: SignerWithAddress
  let l2MessengerImpersonator: SignerWithAddress
  before(async () => {
    // Create a special signer which will enable us to send messages from the L2Messenger contract
    ;[alice, bob, l2MessengerImpersonator] = await ethers.getSigners()
  })

  let L2StandardBridge: Contract
  let L2ERC20: Contract
  let Fake__L2CrossDomainMessenger: FakeContract
  beforeEach(async () => {
    // Get a new mock L2 messenger
    Fake__L2CrossDomainMessenger = await smock.fake<Contract>(
      'L2CrossDomainMessenger',
      // This allows us to use an ethers override {from: Mock__L2CrossDomainMessenger.address} to mock calls
      { address: await l2MessengerImpersonator.getAddress() }
    )

    // Deploy the contract under test
    L2StandardBridge = await deploy('L2StandardBridge', {
      args: [Fake__L2CrossDomainMessenger.address, DUMMY_L1_BRIDGE_ADDRESS],
    })

    // Deploy an L2 ERC20
    L2ERC20 = await deploy('L2StandardERC20', {
      args: [
        L2StandardBridge.address,
        DUMMY_L1_ERC20_ADDRESS,
        'L2Token',
        'L2T',
      ],
    })
  })

  // test the transfer flow of moving a token from L2 to L1
  describe('finalizeDeposit', () => {
    it('onlyFromCrossDomainAccount: should revert on calls from a non-crossDomainMessenger L2 account', async () => {
      await expect(
        L2StandardBridge.finalizeDeposit(
          DUMMY_L1_ERC20_ADDRESS,
          NON_ZERO_ADDRESS,
          NON_ZERO_ADDRESS,
          NON_ZERO_ADDRESS,
          0,
          NON_NULL_BYTES32
        )
      ).to.be.revertedWith(ERROR_STRINGS.INVALID_MESSENGER)
    })

    it('onlyFromCrossDomainAccount: should revert on calls from the right crossDomainMessenger, but wrong xDomainMessageSender (ie. not the L1L1StandardBridge)', async () => {
      Fake__L2CrossDomainMessenger.xDomainMessageSender.returns(
        NON_ZERO_ADDRESS
      )

      await expect(
        L2StandardBridge.connect(l2MessengerImpersonator).finalizeDeposit(
          DUMMY_L1_ERC20_ADDRESS,
          NON_ZERO_ADDRESS,
          NON_ZERO_ADDRESS,
          NON_ZERO_ADDRESS,
          0,
          NON_NULL_BYTES32,
          {
            from: Fake__L2CrossDomainMessenger.address,
          }
        )
      ).to.be.revertedWith(ERROR_STRINGS.INVALID_X_DOMAIN_MSG_SENDER)
    })

    it('should initialize a withdrawal if the L2 token is not compliant', async () => {
      // Deploy a non compliant ERC20
      const NonCompliantERC20 = await deploy('ERC20', {
        args: ['L2Token', 'L2T'],
      })

      L2StandardBridge.connect(l2MessengerImpersonator).finalizeDeposit(
        DUMMY_L1_ERC20_ADDRESS,
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
        DUMMY_L1_BRIDGE_ADDRESS
      )

      await L2StandardBridge.connect(l2MessengerImpersonator).finalizeDeposit(
        DUMMY_L1_ERC20_ADDRESS,
        NonCompliantERC20.address,
        alice.address,
        bob.address,
        100,
        NON_NULL_BYTES32,
        {
          from: Fake__L2CrossDomainMessenger.address,
        }
      )

      expect(
        Fake__L2CrossDomainMessenger.sendMessage.getCall(1).args
      ).to.deep.equal([
        DUMMY_L1_BRIDGE_ADDRESS,
        getContractInterface('L1StandardBridge').encodeFunctionData(
          'finalizeERC20Withdrawal',
          [
            DUMMY_L1_ERC20_ADDRESS,
            NonCompliantERC20.address,
            bob.address,
            alice.address,
            100,
            NON_NULL_BYTES32,
          ]
        ),
        0,
      ])
    })

    it('should credit funds to the depositor', async () => {
      const depositAmount = 100

      Fake__L2CrossDomainMessenger.xDomainMessageSender.returns(
        DUMMY_L1_BRIDGE_ADDRESS
      )

      await L2StandardBridge.connect(l2MessengerImpersonator).finalizeDeposit(
        DUMMY_L1_ERC20_ADDRESS,
        L2ERC20.address,
        alice.address,
        bob.address,
        depositAmount,
        NON_NULL_BYTES32,
        {
          from: Fake__L2CrossDomainMessenger.address,
        }
      )

      expect(await L2ERC20.balanceOf(bob.address)).to.equal(depositAmount)
    })
  })

  describe('withdrawals', () => {
    const withdrawAmount = 1_000

    let Fake__OVM_ETH: FakeContract<Contract>
    before(async () => {
      Fake__OVM_ETH = await smock.fake('OVM_ETH', {
        address: predeploys.OVM_ETH,
      })
    })

    let Mock__L2Token: MockContract<Contract>
    beforeEach(async () => {
      // Deploy a smodded gateway so we can give some balances to withdraw
      Mock__L2Token = await (
        await smock.mock('L2StandardERC20')
      ).deploy(
        L2StandardBridge.address,
        DUMMY_L1_ERC20_ADDRESS,
        'L2Token',
        'L2T'
      )

      await Mock__L2Token.setVariable('_totalSupply', INITIAL_TOTAL_SUPPLY)
      await Mock__L2Token.setVariable('_balances', {
        [alice.address]: ALICE_INITIAL_BALANCE,
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

      expect(
        Fake__L2CrossDomainMessenger.sendMessage.getCall(0).args
      ).to.deep.equal([
        DUMMY_L1_BRIDGE_ADDRESS,
        getContractInterface('L1StandardBridge').encodeFunctionData(
          'finalizeETHWithdrawal',
          [alice.address, alice.address, 0, NON_NULL_BYTES32]
        ),
        0,
      ])
    })

    it('withdraw() burns and sends the correct withdrawal message', async () => {
      await L2StandardBridge.withdraw(
        Mock__L2Token.address,
        withdrawAmount,
        0,
        NON_NULL_BYTES32
      )

      expect(
        Fake__L2CrossDomainMessenger.sendMessage.getCall(0).args
      ).to.deep.equal([
        DUMMY_L1_BRIDGE_ADDRESS,
        getContractInterface('L1StandardBridge').encodeFunctionData(
          'finalizeERC20Withdrawal',
          [
            DUMMY_L1_ERC20_ADDRESS,
            Mock__L2Token.address,
            alice.address,
            alice.address,
            withdrawAmount,
            NON_NULL_BYTES32,
          ]
        ),
        0,
      ])

      // Assert Alice's balance went down
      expect(await Mock__L2Token.balanceOf(alice.address)).to.deep.equal(
        ethers.BigNumber.from(ALICE_INITIAL_BALANCE - withdrawAmount)
      )

      // Assert totalSupply went down
      expect(await Mock__L2Token.totalSupply()).to.deep.equal(
        ethers.BigNumber.from(INITIAL_TOTAL_SUPPLY - withdrawAmount)
      )
    })

    it('withdrawTo() burns and sends the correct withdrawal message', async () => {
      await L2StandardBridge.withdrawTo(
        Mock__L2Token.address,
        bob.address,
        withdrawAmount,
        0,
        NON_NULL_BYTES32
      )

      expect(
        Fake__L2CrossDomainMessenger.sendMessage.getCall(0).args
      ).to.deep.equal([
        DUMMY_L1_BRIDGE_ADDRESS,
        getContractInterface('L1StandardBridge').encodeFunctionData(
          'finalizeERC20Withdrawal',
          [
            DUMMY_L1_ERC20_ADDRESS,
            Mock__L2Token.address,
            alice.address,
            bob.address,
            withdrawAmount,
            NON_NULL_BYTES32,
          ]
        ),
        0,
      ])

      // Assert Alice's balance went down
      expect(await Mock__L2Token.balanceOf(alice.address)).to.deep.equal(
        ethers.BigNumber.from(ALICE_INITIAL_BALANCE - withdrawAmount)
      )

      // Assert totalSupply went down
      expect(await Mock__L2Token.totalSupply()).to.deep.equal(
        ethers.BigNumber.from(INITIAL_TOTAL_SUPPLY - withdrawAmount)
      )
    })
  })

  describe('standard erc20', () => {
    it('should not allow anyone but the L2 bridge to mint and burn', async () => {
      expect(
        L2ERC20.connect(alice).mint(alice.address, 100)
      ).to.be.revertedWith('Only L2 Bridge can mint and burn')

      expect(
        L2ERC20.connect(alice).burn(alice.address, 100)
      ).to.be.revertedWith('Only L2 Bridge can mint and burn')
    })

    it('should return the correct interface support', async () => {
      // ERC165
      expect(await L2ERC20.supportsInterface(0x01ffc9a7)).to.be.true

      // L2StandardERC20
      expect(await L2ERC20.supportsInterface(0x1d1d8b63)).to.be.true

      expect(await L2ERC20.supportsInterface(0xffffffff)).to.be.false
    })
  })
})
