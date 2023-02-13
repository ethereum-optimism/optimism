import hre from 'hardhat'
import { Contract } from 'ethers'
import { toRpcHexString } from '@eth-optimism/core-utils'
import {
  getContractFactory,
  getContractInterface,
} from '@eth-optimism/contracts'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'
import { smock, FakeContract } from '@defi-wonderland/smock'

import { expect } from './setup'
import {
  findEventForStateBatch,
  findFirstUnfinalizedStateBatchIndex,
  OutputOracle,
} from '../src'

describe('helpers', () => {
  // Can be any non-zero value, 1000 is fine.
  const challengeWindowSeconds = 1000

  let signer: SignerWithAddress
  before(async () => {
    ;[signer] = await hre.ethers.getSigners()
  })

  let FakeBondManager: FakeContract<Contract>
  let FakeCanonicalTransactionChain: FakeContract<Contract>
  let AddressManager: Contract
  let ChainStorageContainer: Contract
  let StateCommitmentChain: Contract
  let oracle: OutputOracle<any>
  beforeEach(async () => {
    // Set up fakes
    FakeBondManager = await smock.fake(getContractInterface('BondManager'))
    FakeCanonicalTransactionChain = await smock.fake(
      getContractInterface('CanonicalTransactionChain')
    )

    // Set up contracts
    AddressManager = await getContractFactory(
      'Lib_AddressManager',
      signer
    ).deploy()
    ChainStorageContainer = await getContractFactory(
      'ChainStorageContainer',
      signer
    ).deploy(AddressManager.address, 'StateCommitmentChain')
    StateCommitmentChain = await getContractFactory(
      'StateCommitmentChain',
      signer
    ).deploy(AddressManager.address, challengeWindowSeconds, 10000000)

    // Set addresses in manager
    await AddressManager.setAddress(
      'ChainStorageContainer-SCC-batches',
      ChainStorageContainer.address
    )
    await AddressManager.setAddress(
      'StateCommitmentChain',
      StateCommitmentChain.address
    )
    await AddressManager.setAddress(
      'CanonicalTransactionChain',
      FakeCanonicalTransactionChain.address
    )
    await AddressManager.setAddress('BondManager', FakeBondManager.address)

    // Set up mock returns
    FakeCanonicalTransactionChain.getTotalElements.returns(1000000000) // just needs to be large
    FakeBondManager.isCollateralized.returns(true)

    oracle = {
      contract: StateCommitmentChain,
      filter: StateCommitmentChain.filters.StateBatchAppended(),
      getTotalElements: async () => StateCommitmentChain.getTotalBatches(),
      getEventIndex: (args: any) => args._batchIndex,
    }
  })

  describe('findEventForStateBatch', () => {
    describe('when the event exists once', () => {
      beforeEach(async () => {
        await StateCommitmentChain.appendStateBatch(
          [hre.ethers.constants.HashZero],
          0
        )
      })

      it('should return the event', async () => {
        const event = await findEventForStateBatch(oracle, 0)

        expect(event.args._batchIndex).to.equal(0)
      })
    })

    describe('when the event does not exist', () => {
      it('should throw an error', async () => {
        await expect(
          findEventForStateBatch(oracle, 0)
        ).to.eventually.be.rejectedWith('unable to find event for batch')
      })
    })
  })

  describe('findFirstUnfinalizedIndex', () => {
    describe('when the chain is more then FPW seconds old', () => {
      beforeEach(async () => {
        await StateCommitmentChain.appendStateBatch(
          [hre.ethers.constants.HashZero],
          0
        )

        // Simulate FPW passing
        await hre.ethers.provider.send('evm_increaseTime', [
          toRpcHexString(challengeWindowSeconds * 2),
        ])

        await StateCommitmentChain.appendStateBatch(
          [hre.ethers.constants.HashZero],
          1
        )
        await StateCommitmentChain.appendStateBatch(
          [hre.ethers.constants.HashZero],
          2
        )
      })

      it('should find the first batch older than the FPW', async () => {
        const first = await findFirstUnfinalizedStateBatchIndex(
          oracle,
          challengeWindowSeconds
        )

        expect(first).to.equal(1)
      })
    })

    describe('when the chain is less than FPW seconds old', () => {
      beforeEach(async () => {
        await StateCommitmentChain.appendStateBatch(
          [hre.ethers.constants.HashZero],
          0
        )
        await StateCommitmentChain.appendStateBatch(
          [hre.ethers.constants.HashZero],
          1
        )
        await StateCommitmentChain.appendStateBatch(
          [hre.ethers.constants.HashZero],
          2
        )
      })

      it('should return zero', async () => {
        const first = await findFirstUnfinalizedStateBatchIndex(
          oracle,
          challengeWindowSeconds
        )

        expect(first).to.equal(0)
      })
    })

    describe('when no batches submitted for the entire FPW', () => {
      beforeEach(async () => {
        await StateCommitmentChain.appendStateBatch(
          [hre.ethers.constants.HashZero],
          0
        )
        await StateCommitmentChain.appendStateBatch(
          [hre.ethers.constants.HashZero],
          1
        )
        await StateCommitmentChain.appendStateBatch(
          [hre.ethers.constants.HashZero],
          2
        )

        // Simulate FPW passing and no new batches
        await hre.ethers.provider.send('evm_increaseTime', [
          toRpcHexString(challengeWindowSeconds * 2),
        ])

        // Mine a block to force timestamp to update
        await hre.ethers.provider.send('hardhat_mine', ['0x1'])
      })

      it('should return undefined', async () => {
        const first = await findFirstUnfinalizedStateBatchIndex(
          oracle,
          challengeWindowSeconds
        )

        expect(first).to.equal(undefined)
      })
    })
  })
})
