import { expect } from '../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'

/* Internal Imports */
import { deployAllContracts } from '../../src'
import {
  RollupDeployConfig,
  factoryToContractName,
} from '../../src/deployment/types'
import { Signer } from 'ethers'
import { DEFAULT_FORCE_INCLUSION_PERIOD_SECONDS } from '../test-helpers'

describe('Contract Deployment', () => {
  let wallet: Signer
  let sequencer: Signer
  let l1ToL2TransactionPasser: Signer
  before(async () => {
    ;[wallet, sequencer, l1ToL2TransactionPasser] = await ethers.getSigners()
  })

  describe('deployAllContracts', () => {
    it('should deploy contracts in a default configuration', async () => {
      const config: RollupDeployConfig = {
        signer: wallet,
        rollupOptions: {
          forceInclusionPeriodSeconds: DEFAULT_FORCE_INCLUSION_PERIOD_SECONDS,
          ownerAddress: await wallet.getAddress(),
          sequencerAddress: await sequencer.getAddress(),
          gasMeterConfig: {
            ovmTxFlatGasFee: 1000,
            ovmTxMaxGas: 1000,
            gasRateLimitEpochLength: 1000,
            maxSequencedGasPerEpoch: 1000,
            maxQueuedGasPerEpoch: 1000,
          },
        },
      }

      const resolver = await deployAllContracts(config)

      let contractDeployed: boolean
      Object.values(factoryToContractName).forEach((contractName) => {
        contractDeployed = !!resolver.contracts[contractName]
        contractDeployed.should.equal(
          true,
          `Contract ${contractName} was not deployed!`
        )
      })
    })
  })
})
