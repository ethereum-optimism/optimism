import { ethers } from 'hardhat'
import { Signer, Contract, constants } from 'ethers'
import { smock, FakeContract, MockContract } from '@defi-wonderland/smock'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'

import { expect } from '../../../setup'
import { NON_NULL_BYTES32, NON_ZERO_ADDRESS, deploy } from '../../../helpers'
import { getContractInterface, predeploys } from '../../../../src'

// TODO: Maybe we should consider automatically generating these and exporting them?
const ERROR_STRINGS = {
  INVALID_MESSENGER: 'OVM_XCHAIN: messenger contract unauthenticated',
  INVALID_X_DOMAIN_MSG_SENDER:
    'OVM_XCHAIN: wrong sender of cross-domain message',
  ALREADY_INITIALIZED: 'Contract has already been initialized.',
}

const DUMMY_L2_ERC20_ADDRESS = '0xaBBAABbaaBbAABbaABbAABbAABbaAbbaaBbaaBBa'
const DUMMY_L2_BRIDGE_ADDRESS = '0xACDCacDcACdCaCDcacdcacdCaCdcACdCAcDcaCdc'
const INITIAL_TOTAL_L1_SUPPLY = 5000
const FINALIZATION_GAS = 1_200_000

describe('L1StandardBridge', () => {
  let l1MessengerImpersonator: Signer
  let alice: SignerWithAddress
  let bob: SignerWithAddress
  before(async () => {
    ;[l1MessengerImpersonator, alice, bob] = await ethers.getSigners()
  })

  let L1ERC20: MockContract<Contract>
  let L1StandardBridge: Contract
  let Fake__L1CrossDomainMessenger: FakeContract
  beforeEach(async () => {
    // Get a new mock L1 messenger
    Fake__L1CrossDomainMessenger = await smock.fake<Contract>(
      'L1CrossDomainMessenger',
      { address: await l1MessengerImpersonator.getAddress() } // This allows us to use an ethers override {from: Mock__L2CrossDomainMessenger.address} to mock calls
    )

    // Deploy the contract under test
    L1StandardBridge = await deploy('L1StandardBridge')
    await L1StandardBridge.initialize(
      Fake__L1CrossDomainMessenger.address,
      DUMMY_L2_BRIDGE_ADDRESS
    )

    L1ERC20 = await (
      await smock.mock('@openzeppelin/contracts/token/ERC20/ERC20.sol:ERC20')
    ).deploy('L1ERC20', 'ERC')
    await L1ERC20.setVariable('_totalSupply', INITIAL_TOTAL_L1_SUPPLY)
    await L1ERC20.setVariable('_balances', {
      [alice.address]: INITIAL_TOTAL_L1_SUPPLY,
    })
  })

  describe('initialize', () => {
    it('Should only be callable once', async () => {
      await expect(
        L1StandardBridge.initialize(
          ethers.constants.AddressZero,
          DUMMY_L2_BRIDGE_ADDRESS
        )
      ).to.be.revertedWith(ERROR_STRINGS.ALREADY_INITIALIZED)
    })
  })

  describe('receive', () => {
    it('should send an amount of ETH to the callers balance on L2', async () => {
      await expect(
        alice.sendTransaction({
          to: L1StandardBridge.address,
          data: '0x',
        })
      ).to.not.be.reverted
    })
  })

  describe('ETH deposits', () => {
    const depositAmount = 1_000

    it('depositETH() escrows the deposit amount and sends the correct deposit message', async () => {
      const initialBalance = await alice.getBalance()

      // alice calls deposit on the bridge and the L1 bridge calls transferFrom on the token
      const res = await L1StandardBridge.connect(alice).depositETH(
        FINALIZATION_GAS,
        NON_NULL_BYTES32,
        {
          value: depositAmount,
        }
      )

      expect(
        Fake__L1CrossDomainMessenger.sendMessage.getCall(0).args
      ).to.deep.equal([
        DUMMY_L2_BRIDGE_ADDRESS,
        getContractInterface('IL2ERC20Bridge').encodeFunctionData(
          'finalizeDeposit',
          [
            constants.AddressZero,
            predeploys.OVM_ETH,
            alice.address,
            alice.address,
            depositAmount,
            NON_NULL_BYTES32,
          ]
        ),
        FINALIZATION_GAS,
      ])

      const receipt = await res.wait()
      const depositerFeePaid = receipt.cumulativeGasUsed.mul(
        receipt.effectiveGasPrice
      )

      expect(await alice.getBalance()).to.equal(
        initialBalance.sub(depositAmount).sub(depositerFeePaid)
      )

      expect(
        await ethers.provider.getBalance(L1StandardBridge.address)
      ).to.equal(depositAmount)
    })

    it('depositETHTo() escrows the deposit amount and sends the correct deposit message', async () => {
      const initialBalance = await alice.getBalance()

      const res = await L1StandardBridge.connect(alice).depositETHTo(
        bob.address,
        FINALIZATION_GAS,
        NON_NULL_BYTES32,
        {
          value: depositAmount,
        }
      )

      expect(
        Fake__L1CrossDomainMessenger.sendMessage.getCall(0).args
      ).to.deep.equal([
        DUMMY_L2_BRIDGE_ADDRESS,
        getContractInterface('IL2ERC20Bridge').encodeFunctionData(
          'finalizeDeposit',
          [
            constants.AddressZero,
            predeploys.OVM_ETH,
            alice.address,
            bob.address,
            depositAmount,
            NON_NULL_BYTES32,
          ]
        ),
        FINALIZATION_GAS,
      ])

      const receipt = await res.wait()
      const depositerFeePaid = receipt.cumulativeGasUsed.mul(
        receipt.effectiveGasPrice
      )

      expect(await alice.getBalance()).to.equal(
        initialBalance.sub(depositAmount).sub(depositerFeePaid)
      )

      expect(
        await ethers.provider.getBalance(L1StandardBridge.address)
      ).to.equal(depositAmount)
    })

    it('cannot depositETH from a contract account', async () => {
      expect(
        L1StandardBridge.depositETH(FINALIZATION_GAS, NON_NULL_BYTES32, {
          value: depositAmount,
        })
      ).to.be.revertedWith('Account not EOA')
    })
  })

  describe('ETH withdrawals', () => {
    it('onlyFromCrossDomainAccount: should revert on calls from a non-crossDomainMessenger L1 account', async () => {
      await expect(
        L1StandardBridge.connect(alice).finalizeETHWithdrawal(
          constants.AddressZero,
          constants.AddressZero,
          1,
          NON_NULL_BYTES32
        )
      ).to.be.revertedWith(ERROR_STRINGS.INVALID_MESSENGER)
    })

    it('onlyFromCrossDomainAccount: should revert on calls from the right crossDomainMessenger, but wrong xDomainMessageSender (ie. not the L2ETHToken)', async () => {
      Fake__L1CrossDomainMessenger.xDomainMessageSender.returns(
        '0x' + '22'.repeat(20)
      )

      await expect(
        L1StandardBridge.finalizeETHWithdrawal(
          constants.AddressZero,
          constants.AddressZero,
          1,
          NON_NULL_BYTES32,
          {
            from: Fake__L1CrossDomainMessenger.address,
          }
        )
      ).to.be.revertedWith(ERROR_STRINGS.INVALID_X_DOMAIN_MSG_SENDER)
    })

    it('should revert in nothing to withdraw', async () => {
      expect(await ethers.provider.getBalance(NON_ZERO_ADDRESS)).to.be.equal(0)

      Fake__L1CrossDomainMessenger.xDomainMessageSender.returns(
        DUMMY_L2_BRIDGE_ADDRESS
      )

      await expect(
        L1StandardBridge.finalizeETHWithdrawal(
          NON_ZERO_ADDRESS,
          NON_ZERO_ADDRESS,
          100,
          NON_NULL_BYTES32,
          {
            from: Fake__L1CrossDomainMessenger.address,
          }
        )
      ).to.be.revertedWith(
        'TransferHelper::safeTransferETH: ETH transfer failed'
      )
    })

    it('should credit funds to the withdrawer and not use too much gas', async () => {
      expect(await ethers.provider.getBalance(NON_ZERO_ADDRESS)).to.be.equal(0)

      const withdrawalAmount = 100
      Fake__L1CrossDomainMessenger.xDomainMessageSender.returns(
        DUMMY_L2_BRIDGE_ADDRESS
      )

      await L1StandardBridge.connect(alice).depositETH(
        FINALIZATION_GAS,
        NON_NULL_BYTES32,
        {
          value: ethers.utils.parseEther('1.0'),
        }
      )

      await L1StandardBridge.finalizeETHWithdrawal(
        NON_ZERO_ADDRESS,
        NON_ZERO_ADDRESS,
        withdrawalAmount,
        NON_NULL_BYTES32,
        {
          from: Fake__L1CrossDomainMessenger.address,
        }
      )

      expect(await ethers.provider.getBalance(NON_ZERO_ADDRESS)).to.be.equal(
        withdrawalAmount
      )
    })
  })

  describe('ERC20 deposits', () => {
    const depositAmount = 1_000

    beforeEach(async () => {
      await L1ERC20.connect(alice).approve(
        L1StandardBridge.address,
        depositAmount
      )
    })

    it('depositERC20() escrows the deposit amount and sends the correct deposit message', async () => {
      await L1StandardBridge.connect(alice).depositERC20(
        L1ERC20.address,
        DUMMY_L2_ERC20_ADDRESS,
        depositAmount,
        FINALIZATION_GAS,
        NON_NULL_BYTES32
      )

      expect(
        Fake__L1CrossDomainMessenger.sendMessage.getCall(0).args
      ).to.deep.equal([
        DUMMY_L2_BRIDGE_ADDRESS,
        getContractInterface('IL2ERC20Bridge').encodeFunctionData(
          'finalizeDeposit',
          [
            L1ERC20.address,
            DUMMY_L2_ERC20_ADDRESS,
            alice.address,
            alice.address,
            depositAmount,
            NON_NULL_BYTES32,
          ]
        ),
        FINALIZATION_GAS,
      ])

      expect(await L1ERC20.balanceOf(alice.address)).to.equal(
        INITIAL_TOTAL_L1_SUPPLY - depositAmount
      )

      expect(await L1ERC20.balanceOf(L1StandardBridge.address)).to.equal(
        depositAmount
      )
    })

    it('depositERC20To() escrows the deposit amount and sends the correct deposit message', async () => {
      await L1StandardBridge.connect(alice).depositERC20To(
        L1ERC20.address,
        DUMMY_L2_ERC20_ADDRESS,
        bob.address,
        depositAmount,
        FINALIZATION_GAS,
        NON_NULL_BYTES32
      )

      expect(
        Fake__L1CrossDomainMessenger.sendMessage.getCall(0).args
      ).to.deep.equal([
        DUMMY_L2_BRIDGE_ADDRESS,
        getContractInterface('IL2ERC20Bridge').encodeFunctionData(
          'finalizeDeposit',
          [
            L1ERC20.address,
            DUMMY_L2_ERC20_ADDRESS,
            alice.address,
            bob.address,
            depositAmount,
            NON_NULL_BYTES32,
          ]
        ),
        FINALIZATION_GAS,
      ])

      expect(await L1ERC20.balanceOf(alice.address)).to.equal(
        INITIAL_TOTAL_L1_SUPPLY - depositAmount
      )

      expect(await L1ERC20.balanceOf(L1StandardBridge.address)).to.equal(
        depositAmount
      )
    })

    it('cannot depositERC20 from a contract account', async () => {
      expect(
        L1StandardBridge.depositERC20(
          L1ERC20.address,
          DUMMY_L2_ERC20_ADDRESS,
          depositAmount,
          FINALIZATION_GAS,
          NON_NULL_BYTES32
        )
      ).to.be.revertedWith('Account not EOA')
    })

    describe('Handling ERC20.transferFrom() failures that revert ', () => {
      let Fake__L1ERC20: FakeContract
      before(async () => {
        Fake__L1ERC20 = await smock.fake<Contract>('ERC20')
        Fake__L1ERC20.transferFrom.reverts()
      })

      it('depositERC20(): will revert if ERC20.transferFrom() reverts', async () => {
        await expect(
          L1StandardBridge.connect(alice).depositERC20(
            Fake__L1ERC20.address,
            DUMMY_L2_ERC20_ADDRESS,
            depositAmount,
            FINALIZATION_GAS,
            NON_NULL_BYTES32
          )
        ).to.be.revertedWith('SafeERC20: low-level call failed')
      })

      it('depositERC20To(): will revert if ERC20.transferFrom() reverts', async () => {
        await expect(
          L1StandardBridge.connect(alice).depositERC20To(
            Fake__L1ERC20.address,
            DUMMY_L2_ERC20_ADDRESS,
            bob.address,
            depositAmount,
            FINALIZATION_GAS,
            NON_NULL_BYTES32
          )
        ).to.be.revertedWith('SafeERC20: low-level call failed')
      })

      it('depositERC20To(): will revert if the L1 ERC20 has no code or is zero address', async () => {
        await expect(
          L1StandardBridge.connect(alice).depositERC20To(
            ethers.constants.AddressZero,
            DUMMY_L2_ERC20_ADDRESS,
            bob.address,
            depositAmount,
            FINALIZATION_GAS,
            NON_NULL_BYTES32
          )
        ).to.be.revertedWith('Address: call to non-contract')
      })
    })

    describe('Handling ERC20.transferFrom failures that return false', () => {
      let Fake__L1ERC20: FakeContract
      before(async () => {
        Fake__L1ERC20 = await smock.fake('ERC20')
        Fake__L1ERC20.transferFrom.returns(false)
      })

      it('deposit(): will revert if ERC20.transferFrom() returns false', async () => {
        await expect(
          L1StandardBridge.connect(alice).depositERC20(
            Fake__L1ERC20.address,
            DUMMY_L2_ERC20_ADDRESS,
            depositAmount,
            FINALIZATION_GAS,
            NON_NULL_BYTES32
          )
        ).to.be.revertedWith('SafeERC20: ERC20 operation did not succeed')
      })

      it('depositTo(): will revert if ERC20.transferFrom() returns false', async () => {
        await expect(
          L1StandardBridge.depositERC20To(
            Fake__L1ERC20.address,
            DUMMY_L2_ERC20_ADDRESS,
            bob.address,
            depositAmount,
            FINALIZATION_GAS,
            NON_NULL_BYTES32
          )
        ).to.be.revertedWith('SafeERC20: ERC20 operation did not succeed')
      })
    })
  })

  describe('ERC20 withdrawals', () => {
    it('onlyFromCrossDomainAccount: should revert on calls from a non-crossDomainMessenger L1 account', async () => {
      await expect(
        L1StandardBridge.connect(alice).finalizeERC20Withdrawal(
          L1ERC20.address,
          DUMMY_L2_ERC20_ADDRESS,
          constants.AddressZero,
          constants.AddressZero,
          1,
          NON_NULL_BYTES32
        )
      ).to.be.revertedWith(ERROR_STRINGS.INVALID_MESSENGER)
    })

    it('onlyFromCrossDomainAccount: should revert on calls from the right crossDomainMessenger, but wrong xDomainMessageSender (ie. not the L2DepositedERC20)', async () => {
      Fake__L1CrossDomainMessenger.xDomainMessageSender.returns(
        NON_ZERO_ADDRESS
      )

      await expect(
        L1StandardBridge.finalizeERC20Withdrawal(
          L1ERC20.address,
          DUMMY_L2_ERC20_ADDRESS,
          constants.AddressZero,
          constants.AddressZero,
          1,
          NON_NULL_BYTES32,
          {
            from: Fake__L1CrossDomainMessenger.address,
          }
        )
      ).to.be.revertedWith(ERROR_STRINGS.INVALID_X_DOMAIN_MSG_SENDER)
    })

    it('should credit funds to the withdrawer and not use too much gas', async () => {
      // First Alice will 'donate' some tokens so that there's a balance to be withdrawn
      const withdrawalAmount = 10
      await L1ERC20.connect(alice).approve(
        L1StandardBridge.address,
        withdrawalAmount
      )

      await L1StandardBridge.connect(alice).depositERC20(
        L1ERC20.address,
        DUMMY_L2_ERC20_ADDRESS,
        withdrawalAmount,
        FINALIZATION_GAS,
        NON_NULL_BYTES32
      )

      expect(await L1ERC20.balanceOf(L1StandardBridge.address)).to.be.equal(
        withdrawalAmount
      )

      // make sure no balance at start of test
      expect(await L1ERC20.balanceOf(NON_ZERO_ADDRESS)).to.be.equal(0)

      Fake__L1CrossDomainMessenger.xDomainMessageSender.returns(
        DUMMY_L2_BRIDGE_ADDRESS
      )

      await L1StandardBridge.finalizeERC20Withdrawal(
        L1ERC20.address,
        DUMMY_L2_ERC20_ADDRESS,
        NON_ZERO_ADDRESS,
        NON_ZERO_ADDRESS,
        withdrawalAmount,
        NON_NULL_BYTES32,
        { from: Fake__L1CrossDomainMessenger.address }
      )

      expect(await L1ERC20.balanceOf(NON_ZERO_ADDRESS)).to.be.equal(
        withdrawalAmount
      )
    })
  })

  describe('donateETH', () => {
    it('it should just call the function', async () => {
      await expect(L1StandardBridge.donateETH()).to.not.be.reverted
    })

    it('should send ETH to the contract account', async () => {
      await expect(
        L1StandardBridge.donateETH({
          value: 100,
        })
      ).to.not.be.reverted
    })
  })
})
