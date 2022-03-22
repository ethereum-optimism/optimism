import { ethers } from 'hardhat'
import { Signer, ContractFactory, Contract, constants } from 'ethers'
import { Interface } from 'ethers/lib/utils'
import { smock, FakeContract, MockContract, MockContractFactory } from '@defi-wonderland/smock'
import { getAddress } from 'ethers/lib/utils';
import { randomHex } from 'web3-utils';
import { getContractInterface, predeploys } from '../../../../src'

import { expect } from '../../../setup'

describe('L2NativeERC20Bridge', () => {
  let alice: Signer
  let aliceAddress: string
  let bob: Signer
  let bobsAddress: string
  let l2MessengerImpersonator: Signer
  let IL1NativeERC20Bridge: Interface
  const INITIAL_TOTAL_SUPPLY = 100_000
  const ALICE_INITIAL_BALANCE = 50_000

  const DUMMY_L1BRIDGE_ADDRESS: string = getAddress(randomHex(20))
  const DUMMY_L1_ERC20_ADDRESS: string = getAddress(randomHex(20))

  before(async () => {
    // Create a special signer which will enable us to send messages from the L2Messenger contract
    ;[alice, bob, l2MessengerImpersonator] = await ethers.getSigners()
    aliceAddress = await alice.getAddress()
    bobsAddress = await bob.getAddress()

    // get an L1NativeER20Bridge Interface
    IL1NativeERC20Bridge = getContractInterface('IL1NativeERC20Bridge')
  })

  let Mock_L2CrossDomainMessenger: FakeContract
  let L2NativeERC20Bridge: Contract
  let Mock_Factory_ERC20: MockContractFactory<ContractFactory>
  let L2NativeERC20: MockContract<Contract>

  beforeEach(async () => {
   Mock_L2CrossDomainMessenger = await smock.fake<Contract>(
      'L2CrossDomainMessenger',
      { address: await l2MessengerImpersonator.getAddress() }
    )

    // Deploy the contract under test
    L2NativeERC20Bridge = await (
      await ethers.getContractFactory('L2NativeERC20Bridge')
    ).deploy(Mock_L2CrossDomainMessenger.address, DUMMY_L1BRIDGE_ADDRESS)

    // Deploy an L2 native ERC20
    Mock_Factory_ERC20 = await smock.mock(
      '@openzeppelin/contracts/token/ERC20/ERC20.sol:ERC20'
    )

    L2NativeERC20 = await Mock_Factory_ERC20.deploy('L2 Native Token', 'L2NT')

    await L2NativeERC20.setVariable('_totalSupply', INITIAL_TOTAL_SUPPLY)
    await L2NativeERC20.setVariable('_balances', {
      [aliceAddress]: ALICE_INITIAL_BALANCE,
    })
  })

  describe('ERC20 deposits', () => {
    const depositAmount = 1_000

    beforeEach(async () => {
      await L2NativeERC20.connect(alice).approve(
        L2NativeERC20Bridge.address,
        depositAmount
      )
    })

    it('depositERC20() escrows the deposit amount and sends the correct deposit message', async () => {
      await L2NativeERC20Bridge.depositERC20(
        L2NativeERC20.address,
        DUMMY_L1_ERC20_ADDRESS,
        depositAmount,
        0,
        ethers.constants.HashZero)

      // Check the sender's balance had decreased and the bridge balance increased by the deposit amount
      const depositerBalance = await L2NativeERC20.balanceOf(aliceAddress)
      expect(depositerBalance).to.equal(ALICE_INITIAL_BALANCE - depositAmount)

      const bridgeBalance = await L2NativeERC20.balanceOf(L2NativeERC20Bridge.address)
      expect(bridgeBalance).to.equal(depositAmount)

      const depositCallToMessenger =
      Mock_L2CrossDomainMessenger.sendMessage.getCall(0)
      // Check the message was sent to the L1 bridge
      expect(depositCallToMessenger.args[0]).to.equal(DUMMY_L1BRIDGE_ADDRESS)

      // Check the correct message was sent accross the layers
      expect(depositCallToMessenger.args[1]).to.equal(
        IL1NativeERC20Bridge.encodeFunctionData('finalizeDeposit', [
          L2NativeERC20.address,
          DUMMY_L1_ERC20_ADDRESS,
          aliceAddress,
          aliceAddress,
          depositAmount,
          ethers.constants.HashZero,
        ])
      )
      expect(depositCallToMessenger.args[2]).to.equal(0)
    })

    it('depositERC20To() escrows the deposit amount and sends the correct deposit message', async () => {
      await L2NativeERC20Bridge.depositERC20To(L2NativeERC20.address, DUMMY_L1_ERC20_ADDRESS, bobsAddress, depositAmount, 0, ethers.constants.HashZero)

      // Check the sender's balance had decreased and the bridge balance increased by the deposit amount
      const depositerBalance = await L2NativeERC20.balanceOf(aliceAddress)
      expect(depositerBalance).to.equal(ALICE_INITIAL_BALANCE - depositAmount)

      const bridgeBalance = await L2NativeERC20.balanceOf(L2NativeERC20Bridge.address)
      expect(bridgeBalance).to.equal(depositAmount)

      const depositCallToMessenger =
      Mock_L2CrossDomainMessenger.sendMessage.getCall(0)
      // Check the message was sent to the L1 bridge
      expect(depositCallToMessenger.args[0]).to.equal(DUMMY_L1BRIDGE_ADDRESS)

      // Check the correct message was sent accross the layers
      expect(depositCallToMessenger.args[1]).to.equal(
        IL1NativeERC20Bridge.encodeFunctionData('finalizeDeposit', [
          L2NativeERC20.address,
          DUMMY_L1_ERC20_ADDRESS,
          aliceAddress,
          bobsAddress,
          depositAmount,
          ethers.constants.HashZero,
        ])
      )
      expect(depositCallToMessenger.args[2]).to.equal(0)
    })

    it('cannot deposit from a contract account', async () => {
      expect(
        L2NativeERC20Bridge.depositERC20(
          L2NativeERC20.address,
          DUMMY_L1_ERC20_ADDRESS,
          depositAmount,
          0,
          ethers.constants.HashZero
      )).to.be.revertedWith('Account not EOA')
    })
  })
})
