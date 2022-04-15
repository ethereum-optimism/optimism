import { ethers } from 'hardhat'
import { Contract } from 'ethers'
import { MockContract, smock } from '@defi-wonderland/smock'
import { expectApprox } from '@eth-optimism/core-utils'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'

import {
  deploy,
  L2_GAS_DISCOUNT_DIVISOR,
  ENQUEUE_GAS_COST,
  NON_ZERO_ADDRESS,
  NON_NULL_BYTES32,
} from '../../../helpers'
import { expect } from '../../../setup'

// Still have some duplication from CanonicalTransactionChain.spec.ts, but it's so minimal that
// this is probably cleaner for now. Particularly since we're planning to move all of this out into
// core-utils soon anyway.
const MAX_GAS_LIMIT = 8_000_000
const INITIAL_TOTAL_L1_SUPPLY = 5000
const FINALIZATION_GAS = 1_200_000

describe('[GAS BENCHMARK] Depositing via the standard bridge [ @skip-on-coverage ]', () => {
  let sequencer: SignerWithAddress
  let alice: SignerWithAddress
  before(async () => {
    ;[sequencer, alice] = await ethers.getSigners()
  })

  let AddressManager: Contract
  let CanonicalTransactionChain: Contract
  before(async () => {
    AddressManager = await deploy('Lib_AddressManager')

    CanonicalTransactionChain = await deploy('CanonicalTransactionChain', {
      args: [
        AddressManager.address,
        MAX_GAS_LIMIT,
        L2_GAS_DISCOUNT_DIVISOR,
        ENQUEUE_GAS_COST,
      ],
    })

    const batches = await deploy('ChainStorageContainer', {
      args: [AddressManager.address, 'CanonicalTransactionChain'],
    })

    await AddressManager.setAddress(
      'OVM_Sequencer',
      await sequencer.getAddress()
    )

    await AddressManager.setAddress(
      'ChainStorageContainer-CTC-batches',
      batches.address
    )

    await AddressManager.setAddress(
      'CanonicalTransactionChain',
      CanonicalTransactionChain.address
    )
  })

  let L1CrossDomainMessenger: Contract
  before(async () => {
    const xDomainMessengerImpl = await deploy('L1CrossDomainMessenger')

    await AddressManager.setAddress(
      'L1CrossDomainMessenger',
      xDomainMessengerImpl.address
    )

    const proxy = await deploy('Lib_ResolvedDelegateProxy', {
      args: [AddressManager.address, 'L1CrossDomainMessenger'],
    })

    L1CrossDomainMessenger = xDomainMessengerImpl.attach(proxy.address)

    await L1CrossDomainMessenger.initialize(AddressManager.address)
  })

  let L1ERC20: MockContract<Contract>
  let L1StandardBridge: Contract
  before('Deploy the bridge and setup the token', async () => {
    L1StandardBridge = await deploy('L1StandardBridge')
    await L1StandardBridge.initialize(
      L1CrossDomainMessenger.address,
      NON_ZERO_ADDRESS
    )

    L1ERC20 = await (await smock.mock('ERC20')).deploy('L1ERC20', 'ERC')
    await L1ERC20.setVariable('_totalSupply', INITIAL_TOTAL_L1_SUPPLY)
    await L1ERC20.setVariable('_balances', {
      [alice.address]: INITIAL_TOTAL_L1_SUPPLY,
    })
  })

  describe('[GAS BENCHMARK] L1 to L2 Deposit costs [ @skip-on-coverage ]', async () => {
    const depositAmount = 1_000

    before(async () => {
      // Load a transaction into the queue first to 'dirty' the buffer's length slot
      await CanonicalTransactionChain.enqueue(
        NON_ZERO_ADDRESS,
        FINALIZATION_GAS,
        '0x1234'
      )
    })

    it('cost to deposit ETH', async () => {
      // Alice calls deposit on the bridge and the L1 bridge calls transferFrom on the token.
      const res = await L1StandardBridge.connect(alice).depositETH(
        FINALIZATION_GAS,
        NON_NULL_BYTES32,
        {
          value: depositAmount,
        }
      )

      const receipt = await res.wait()
      const gasUsed = receipt.gasUsed.toNumber()
      console.log('    - Gas used:', gasUsed)

      expectApprox(gasUsed, 132_481, {
        absoluteUpperDeviation: 500,
        // Assert a lower bound of 1% reduction on gas cost. If your tests are breaking because your
        // contracts are too efficient, consider updating the target value!
        percentLowerDeviation: 1,
      })

      // Sanity check that the message was enqueued.
      expect(await CanonicalTransactionChain.getQueueLength()).to.equal(2)
    })

    it('cost to deposit an ERC20', async () => {
      await L1ERC20.connect(alice).approve(
        L1StandardBridge.address,
        depositAmount
      )

      // Alice calls deposit on the bridge and the L1 bridge calls transferFrom on the token.
      const res = await L1StandardBridge.connect(alice).depositERC20(
        L1ERC20.address,
        NON_ZERO_ADDRESS,
        depositAmount,
        FINALIZATION_GAS,
        NON_NULL_BYTES32
      )

      const receipt = await res.wait()
      const gasUsed = receipt.gasUsed.toNumber()
      console.log('    - Gas used:', gasUsed)

      expectApprox(gasUsed, 192_822, {
        absoluteUpperDeviation: 500,
        // Assert a lower bound of 1% reduction on gas cost. If your tests are breaking because your
        // contracts are too efficient, consider updating the target value!
        percentLowerDeviation: 1,
      })

      // Sanity check that the message was enqueued.
      expect(await CanonicalTransactionChain.getQueueLength()).to.equal(3)
    })
  })
})
