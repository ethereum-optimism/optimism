import hre from 'hardhat'
import '@nomiclabs/hardhat-ethers'
import { Contract, utils } from 'ethers'
import { toRpcHexString } from '@eth-optimism/core-utils'
import Artifact__L2OutputOracle from '@eth-optimism/contracts-bedrock/forge-artifacts/L2OutputOracle.sol/L2OutputOracle.json'
import Artifact__Proxy from '@eth-optimism/contracts-bedrock/forge-artifacts/Proxy.sol/Proxy.json'
import { SignerWithAddress } from '@nomiclabs/hardhat-ethers/signers'

import { expect } from './setup'
import {
  findOutputForIndex,
  findFirstUnfinalizedOutputIndex,
} from '../../src/fault-mon'

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
  let Proxy: Contract
  beforeEach(async () => {
    const Factory__Proxy = new hre.ethers.ContractFactory(
      Artifact__Proxy.abi,
      Artifact__Proxy.bytecode.object,
      signer
    )

    Proxy = await Factory__Proxy.deploy(signer.address)

    const Factory__L2OutputOracle = new hre.ethers.ContractFactory(
      Artifact__L2OutputOracle.abi,
      Artifact__L2OutputOracle.bytecode.object,
      signer
    )

    const L2OutputOracleImplementation = await Factory__L2OutputOracle.deploy(
      deployConfig.l2OutputOracleSubmissionInterval,
      deployConfig.l2BlockTime,
      deployConfig.l2OutputOracleStartingBlockNumber,
      deployConfig.l2OutputOracleStartingTimestamp,
      deployConfig.l2OutputOracleProposer,
      deployConfig.l2OutputOracleChallenger,
      deployConfig.finalizationPeriodSeconds
    )

    await Proxy.upgradeToAndCall(
      L2OutputOracleImplementation.address,
      L2OutputOracleImplementation.interface.encodeFunctionData('initialize', [
        deployConfig.l2OutputOracleStartingBlockNumber,
        deployConfig.l2OutputOracleStartingTimestamp,
      ])
    )

    L2OutputOracle = new hre.ethers.Contract(
      Proxy.address,
      Artifact__L2OutputOracle.abi,
      signer
    )
  })

  describe('findOutputForIndex', () => {
    describe('when the output exists once', () => {
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

      it('should return the output', async () => {
        const output = await findOutputForIndex(L2OutputOracle, 0)

        expect(output.l2OutputIndex).to.equal(0)
      })
    })

    describe('when the output does not exist', () => {
      it('should throw an error', async () => {
        await expect(
          findOutputForIndex(L2OutputOracle, 0)
        ).to.eventually.be.rejectedWith('unable to find output for index')
      })
    })
  })

  describe('findFirstUnfinalizedIndex', () => {
    describe('when the chain is more then FPW seconds old', () => {
      beforeEach(async () => {
        const latestBlock = await hre.ethers.provider.getBlock('latest')
        const params = {
          _l2BlockNumber:
            deployConfig.l2OutputOracleStartingBlockNumber +
            deployConfig.l2OutputOracleSubmissionInterval,
          _l1BlockHash: latestBlock.hash,
          _l1BlockNumber: latestBlock.number,
        }
        await L2OutputOracle.proposeL2Output(
          utils.formatBytes32String('outputRoot1'),
          params._l2BlockNumber,
          params._l1BlockHash,
          params._l1BlockNumber
        )

        // Simulate FPW passing
        await hre.ethers.provider.send('evm_increaseTime', [
          toRpcHexString(deployConfig.finalizationPeriodSeconds * 2),
        ])

        await L2OutputOracle.proposeL2Output(
          utils.formatBytes32String('outputRoot2'),
          params._l2BlockNumber + deployConfig.l2OutputOracleSubmissionInterval,
          params._l1BlockHash,
          params._l1BlockNumber
        )
        await L2OutputOracle.proposeL2Output(
          utils.formatBytes32String('outputRoot3'),
          params._l2BlockNumber +
            deployConfig.l2OutputOracleSubmissionInterval * 2,
          params._l1BlockHash,
          params._l1BlockNumber
        )
      })

      it('should find the first batch older than the FPW', async () => {
        const first = await findFirstUnfinalizedOutputIndex(
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
        const first = await findFirstUnfinalizedOutputIndex(
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
        const first = await findFirstUnfinalizedOutputIndex(
          L2OutputOracle,
          deployConfig.finalizationPeriodSeconds
        )

        expect(first).to.equal(undefined)
      })
    })
  })
})
