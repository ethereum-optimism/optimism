import { expect } from '../../../setup'
import { deployContractCode } from '../../../helpers/utils'

/* External Imports */
import { ethers } from 'hardhat'
import { Contract, ContractFactory, Signer, BigNumber } from 'ethers'
import { smockit, MockContract } from '@eth-optimism/smock'

/* Internal Imports */
import {
  makeAddressManager,
  NON_ZERO_ADDRESS,
  NON_NULL_BYTES32,
  GasMeasurement,
} from '../../../helpers'

import { OVM_TX_GAS_LIMIT, RUN_OVM_TEST_GAS } from '../../../helpers/constants'

const DUMMY_GASMETERCONFIG = {
  minTransactionGasLimit: 0,
  maxTransactionGasLimit: NON_NULL_BYTES32,
  maxGasPerQueuePerEpoch: NON_NULL_BYTES32,
  secondsPerEpoch: NON_NULL_BYTES32,
}

const DUMMY_GLOBALCONTEXT = {
  ovmCHAINID: 420,
}

const QUEUE_ORIGIN = {
  SEQUENCER_QUEUE: 0,
  L1TOL2_QUEUE: 1,
}

const DUMMY_TRANSACTION = {
  timestamp: 111111111111,
  blockNumber: 20,
  l1QueueOrigin: QUEUE_ORIGIN.SEQUENCER_QUEUE,
  l1TxOrigin: NON_ZERO_ADDRESS,
  entrypoint: NON_ZERO_ADDRESS, // update this below
  gasLimit: 10_000_000,
  data: 0,
}

describe('OVM_ExecutionManager gas consumption', () => {
  let wallet: Signer
  let Factory__OVM_StateManager: ContractFactory
  let Factory__OVM_ExecutionManager: ContractFactory
  let Factory__Helper_TestRunner: ContractFactory
  let MOCK__STATE_MANAGER: MockContract
  let AddressManager: Contract
  let targetContractAddress: string
  let gasMeasurement: GasMeasurement
  let OVM_StateManager: Contract
  let Helper_ExecutionManager: Contract
  let Helper_TestRunner: Contract
  before(async () => {
    ;[wallet] = await ethers.getSigners()

    Factory__OVM_StateManager = await ethers.getContractFactory(
      'OVM_StateManager'
    )

    Factory__OVM_ExecutionManager = await ethers.getContractFactory(
      'Helper_ExecutionManager'
    )

    Factory__Helper_TestRunner = await ethers.getContractFactory(
      'Helper_TestRunner'
    )

    // Deploy a simple contract that just returns successfully with no data
    targetContractAddress = await deployContractCode(
      '60206001f3',
      wallet,
      10_000_000
    )
    DUMMY_TRANSACTION.entrypoint = targetContractAddress

    AddressManager = await makeAddressManager()

    // deploy the state manager and mock it for the state transitioner
    MOCK__STATE_MANAGER = await smockit(
      await (await ethers.getContractFactory('OVM_StateManager')).deploy(
        NON_ZERO_ADDRESS
      )
    )

    // Setup the SM to satisfy all the checks executed during EM.run()
    MOCK__STATE_MANAGER.smocked.isAuthenticated.will.return.with(true)
    MOCK__STATE_MANAGER.smocked.getAccountEthAddress.will.return.with(
      targetContractAddress
    )
    MOCK__STATE_MANAGER.smocked.hasAccount.will.return.with(true)
    MOCK__STATE_MANAGER.smocked.testAndSetAccountLoaded.will.return.with(true)

    await AddressManager.setAddress(
      'OVM_StateManagerFactory',
      MOCK__STATE_MANAGER.address
    )

    gasMeasurement = new GasMeasurement()
    await gasMeasurement.init(wallet)

    Helper_TestRunner = await Factory__Helper_TestRunner.deploy()

    OVM_StateManager = (
      await Factory__OVM_StateManager.deploy(await wallet.getAddress())
    ).connect(wallet)

    await OVM_StateManager.connect(wallet).putAccount(
      Helper_TestRunner.address,
      {
        nonce: BigNumber.from(123),
        balance: BigNumber.from(456),
        storageRoot: NON_NULL_BYTES32,
        codeHash: NON_NULL_BYTES32,
        ethAddress: Helper_TestRunner.address,
      }
    )

    Helper_ExecutionManager = (
      await Factory__OVM_ExecutionManager.deploy(
        AddressManager.address,
        DUMMY_GASMETERCONFIG,
        DUMMY_GLOBALCONTEXT
      )
    ).connect(wallet)

    await OVM_StateManager.connect(wallet).setExecutionManager(
      Helper_ExecutionManager.address
    )
  })

  describe('Measure cost of a very simple contract', async () => {
    it('Gas cost of run', async () => {
      const gasCost = await gasMeasurement.getGasCost(
        Helper_ExecutionManager,
        'run',
        [DUMMY_TRANSACTION, MOCK__STATE_MANAGER.address]
      )
      console.log(`calculated gas cost of ${gasCost}`)

      const benchmark: number = 106_000
      expect(gasCost).to.be.lte(benchmark)
      expect(gasCost).to.be.gte(
        benchmark - 1_000,
        'Gas cost has significantly decreased, consider updating the benchmark to reflect the change'
      )
    })

    const dataVariants = [
      {
        inputData: '0x11',
        returnData:
          '0x000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000011100000000000000000000000000000000000000000000000000000000000000',
      },
      {
        inputData:
          '0x1111111111111111111111111111111111111111111111111111111111111111',
        returnData:
          '0x000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000201111111111111111111111111111111111111111111111111111111111111111',
      },
      {
        inputData:
          '0x11111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111',
        returnData:
          '0x0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000004011111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111',
      },
      {
        inputData:
          '0x111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111',
        returnData:
          '0x00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000060111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111',
      },
      {
        inputData:
          '0x1111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111',
        returnData:
          '0x000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000801111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111111',
      },
    ]

    dataVariants.forEach(async (dataVariant) => {
      it('Gas cost of ovmCALL', async () => {
        const ovmCallData = Helper_ExecutionManager.interface.encodeFunctionData(
          'returnData',
          [dataVariant.inputData]
        )

        const encodedStep = Helper_TestRunner.interface.encodeFunctionData(
          'runSingleTestStep',
          [
            {
              functionName: 'ovmCALL',
              functionData: ovmCallData,
              expectedReturnStatus: true,
              expectedReturnData: dataVariant.returnData,
              onlyValidateFlag: false,
            },
          ]
        )

        const tx = await Helper_ExecutionManager.ovmCALLHelper(
          OVM_TX_GAS_LIMIT,
          Helper_TestRunner.address,
          encodedStep,
          OVM_StateManager.address,
          { gasLimit: RUN_OVM_TEST_GAS }
        )

        const gasUsed = (
          await Helper_ExecutionManager.provider.getTransactionReceipt(tx.hash)
        ).gasUsed

        console.log(
          'inputData bytes size',
          ethers.utils.hexDataLength(dataVariant.inputData)
        )
        console.log(
          'returnData bytes size',
          ethers.utils.hexDataLength(dataVariant.returnData)
        )
        console.log('gasUsed', gasUsed.toString())
      })

      it('Gas cost of ovmDELEGATECALL', async () => {
        const ovmCallData = Helper_ExecutionManager.interface.encodeFunctionData(
          'returnData',
          [dataVariant.inputData]
        )

        const encodedStep = Helper_TestRunner.interface.encodeFunctionData(
          'runSingleTestStep',
          [
            {
              functionName: 'ovmDELEGATECALL',
              functionData: ovmCallData,
              expectedReturnStatus: true,
              expectedReturnData: dataVariant.returnData,
              onlyValidateFlag: false,
            },
          ]
        )

        const tx = await Helper_ExecutionManager.ovmDELEGATECALLHelper(
          OVM_TX_GAS_LIMIT,
          Helper_TestRunner.address,
          encodedStep,
          OVM_StateManager.address,
          { gasLimit: RUN_OVM_TEST_GAS }
        )

        const gasUsed = (
          await Helper_ExecutionManager.provider.getTransactionReceipt(tx.hash)
        ).gasUsed

        console.log(
          'inputData bytes size',
          ethers.utils.hexDataLength(dataVariant.inputData)
        )
        console.log(
          'returnData bytes size',
          ethers.utils.hexDataLength(dataVariant.returnData)
        )
        console.log('gasUsed', gasUsed.toString())
      })
    })
  })
})
