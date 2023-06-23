import hre from 'hardhat'
import { Contract, utils } from 'ethers'
import { toRpcHexString } from '@eth-optimism/core-utils'
import { getContractFactory } from '@eth-optimism/contracts-bedrock'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'

import { expect } from './setup'
import {
  findEventForStateBatch,
  findFirstUnfinalizedStateBatchIndex,
} from '../src'

describe('helpers', () => {
  const deployConfig = {
    l2OutputOracleSubmissionInterval: 6,
    l2BlockTime: 2,
    l2OutputOracleStartingBlockNumber: 0,
    l2OutputOracleStartingTimestamp: 0,
    l2OutputOracleProposer: '0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266',
    l2OutputOracleChallenger: '0x6925B8704Ff96DEe942623d6FB5e946EF5884b63',
    // Can be any non-zero value, 1000 is fine.
    finalizationPeriodSeconds: 1000,
  }

  let signer: SignerWithAddress
  before(async () => {
    ;[signer] = await hre.ethers.getSigners()
  })

  let L2OutputOracle: Contract
  beforeEach(async () => {
    L2OutputOracle = await getContractFactory('L2OutputOracle', signer).deploy(
      deployConfig.l2OutputOracleSubmissionInterval,
      deployConfig.l2BlockTime,
      deployConfig.l2OutputOracleStartingBlockNumber,
      deployConfig.l2OutputOracleStartingTimestamp,
      deployConfig.l2OutputOracleProposer,
      deployConfig.l2OutputOracleChallenger,
      deployConfig.finalizationPeriodSeconds
    )
  })

  describe('findEventForStateBatch', () => {
    describe('when the event exists once', () => {
      beforeEach(async () => {
        const latestBlock = await hre.ethers.provider.getBlock('latest')
        const params = {
          _outputRoot: utils.formatBytes32String('testhash'),
          _l2BlockNumber:
            deployConfig.l2OutputOracleStartingBlockNumber +
            deployConfig.l2OutputOracleSubmissionInterval,
          _l1BlockHash: latestBlock.hash,
          _l1BlockNumber: latestBlock.number,
        }
        await L2OutputOracle.proposeL2Output(
          params._outputRoot,
          params._l2BlockNumber,
          params._l1BlockHash,
          params._l1BlockNumber
        )
      })

      it('should return the event', async () => {
        const event = await findEventForStateBatch(L2OutputOracle, 0)

        expect(event.args.l2OutputIndex).to.equal(0)
      })
    })

    describe('when the event does not exist', () => {
      it('should throw an error', async () => {
        await expect(
          findEventForStateBatch(L2OutputOracle, 0)
        ).to.eventually.be.rejectedWith('unable to find event for batch')
      })
    })
  })

  describe('findFirstUnfinalizedIndex', () => {
    describe('when the chain is more then FPW seconds old', () => {
      beforeEach(async () => {
        const latestBlock = await hre.ethers.provider.getBlock('latest')
        const params = {
          _outputRoot: utils.formatBytes32String('testhash'),
          _l2BlockNumber:
            deployConfig.l2OutputOracleStartingBlockNumber +
            deployConfig.l2OutputOracleSubmissionInterval,
          _l1BlockHash: latestBlock.hash,
          _l1BlockNumber: latestBlock.number,
        }
        await L2OutputOracle.proposeL2Output(
          params._outputRoot,
          params._l2BlockNumber,
          params._l1BlockHash,
          params._l1BlockNumber
        )

        // Simulate FPW passing
        await hre.ethers.provider.send('evm_increaseTime', [
          toRpcHexString(deployConfig.finalizationPeriodSeconds * 2),
        ])

        await L2OutputOracle.proposeL2Output(
          params._outputRoot,
          params._l2BlockNumber + deployConfig.l2OutputOracleSubmissionInterval,
          params._l1BlockHash,
          params._l1BlockNumber
        )
        await L2OutputOracle.proposeL2Output(
          params._outputRoot,
          params._l2BlockNumber +
            deployConfig.l2OutputOracleSubmissionInterval * 2,
          params._l1BlockHash,
          params._l1BlockNumber
        )
      })

      it('should find the first batch older than the FPW', async () => {
        const first = await findFirstUnfinalizedStateBatchIndex(
          L2OutputOracle,
          deployConfig.finalizationPeriodSeconds
        )

        expect(first).to.equal(1)
      })
    })

    describe('when the chain is less than FPW seconds old', () => {
      beforeEach(async () => {
        const latestBlock = await hre.ethers.provider.getBlock('latest')
        const params = {
          _outputRoot: utils.formatBytes32String('testhash'),
          _l2BlockNumber:
            deployConfig.l2OutputOracleStartingBlockNumber +
            deployConfig.l2OutputOracleSubmissionInterval,
          _l1BlockHash: latestBlock.hash,
          _l1BlockNumber: latestBlock.number,
        }
        await L2OutputOracle.proposeL2Output(
          params._outputRoot,
          params._l2BlockNumber,
          params._l1BlockHash,
          params._l1BlockNumber
        )
        await L2OutputOracle.proposeL2Output(
          params._outputRoot,
          params._l2BlockNumber + deployConfig.l2OutputOracleSubmissionInterval,
          params._l1BlockHash,
          params._l1BlockNumber
        )
        await L2OutputOracle.proposeL2Output(
          params._outputRoot,
          params._l2BlockNumber +
            deployConfig.l2OutputOracleSubmissionInterval * 2,
          params._l1BlockHash,
          params._l1BlockNumber
        )
      })

      it('should return zero', async () => {
        const first = await findFirstUnfinalizedStateBatchIndex(
          L2OutputOracle,
          deployConfig.finalizationPeriodSeconds
        )

        expect(first).to.equal(0)
      })
    })

    describe('when no batches submitted for the entire FPW', () => {
      beforeEach(async () => {
        const latestBlock = await hre.ethers.provider.getBlock('latest')
        const params = {
          _outputRoot: utils.formatBytes32String('testhash'),
          _l2BlockNumber:
            deployConfig.l2OutputOracleStartingBlockNumber +
            deployConfig.l2OutputOracleSubmissionInterval,
          _l1BlockHash: latestBlock.hash,
          _l1BlockNumber: latestBlock.number,
        }
        await L2OutputOracle.proposeL2Output(
          params._outputRoot,
          params._l2BlockNumber,
          params._l1BlockHash,
          params._l1BlockNumber
        )
        await L2OutputOracle.proposeL2Output(
          params._outputRoot,
          params._l2BlockNumber + deployConfig.l2OutputOracleSubmissionInterval,
          params._l1BlockHash,
          params._l1BlockNumber
        )
        await L2OutputOracle.proposeL2Output(
          params._outputRoot,
          params._l2BlockNumber +
            deployConfig.l2OutputOracleSubmissionInterval * 2,
          params._l1BlockHash,
          params._l1BlockNumber
        )

        // Simulate FPW passing and no new batches
        await hre.ethers.provider.send('evm_increaseTime', [
          toRpcHexString(deployConfig.finalizationPeriodSeconds * 2),
        ])

        // Mine a block to force timestamp to update
        await hre.ethers.provider.send('hardhat_mine', ['0x1'])
      })

      it('should return undefined', async () => {
        const first = await findFirstUnfinalizedStateBatchIndex(
          L2OutputOracle,
          deployConfig.finalizationPeriodSeconds
        )

        expect(first).to.equal(undefined)
      })
    })
  })
})
