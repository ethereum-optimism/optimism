import { should } from '../../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import {
  getLogger,
  hexStrToNumber,
  remove0x,
  add0x,
  TestUtils,
  getCurrentTime,
  NULL_ADDRESS,
  ZERO_ADDRESS,
} from '@eth-optimism/core-utils'
import { Contract, ContractFactory, Signer, BigNumber } from 'ethers'

/* Internal Imports */
import {
  OVM_METHOD_IDS,
  GAS_LIMIT,
  Address,
  manuallyDeployOvmContract,
  addressToBytes32Address,
  makeAddressResolver,
  deployAndRegister,
  AddressResolverMapping,
  executeTestTransaction,
} from '../../../test-helpers'

/* Logging */
const log = getLogger('execution-manager-context', true)

/* Tests */
describe('Execution Manager -- Context opcodes', () => {
  const provider = ethers.provider

  let wallet: Signer
  before(async () => {
    ;[wallet] = await ethers.getSigners()
  })

  let resolver: AddressResolverMapping
  before(async () => {
    resolver = await makeAddressResolver(wallet)
  })

  let ExecutionManager: ContractFactory
  let ContextContract: ContractFactory
  before(async () => {
    ExecutionManager = await ethers.getContractFactory('ExecutionManager')
    ContextContract = await ethers.getContractFactory('ContextContract')
  })

  let executionManager: Contract
  beforeEach(async () => {
    executionManager = await deployAndRegister(
      resolver.addressResolver,
      wallet,
      'ExecutionManager',
      {
        factory: ExecutionManager,
        params: [resolver.addressResolver.address, NULL_ADDRESS, GAS_LIMIT],
      }
    )
  })

  let contractAddress: Address
  let contract2Address: Address
  let contractAddress32: string
  let contract2Address32: string
  beforeEach(async () => {
    contractAddress = await manuallyDeployOvmContract(
      wallet,
      provider,
      executionManager,
      ContextContract,
      [executionManager.address]
    )

    log.debug(`Contract address: [${contractAddress}]`)

    // Deploy SimpleCopier with the ExecutionManager
    contract2Address = await manuallyDeployOvmContract(
      wallet,
      provider,
      executionManager,
      ContextContract,
      [executionManager.address]
    )

    log.debug(`Contract 2 address: [${contract2Address}]`)

    contractAddress32 = addressToBytes32Address(contractAddress)
    contract2Address32 = addressToBytes32Address(contract2Address)
  })

  describe('ovmCALLER', async () => {
    it('reverts when CALLER is not set', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await executeTestTransaction(
          executionManager,
          contractAddress,
          'ovmCALLER',
          []
        )
      })
    })

    it('properly retrieves CALLER when caller is set', async () => {
      const result = await executeTestTransaction(
        executionManager,
        contractAddress,
        'callThroughExecutionManager',
        [contract2Address32, OVM_METHOD_IDS.getCALLER]
      )
      log.debug(`CALLER result: ${result}`)

      should.exist(result, 'Result should exist!')
      result.should.equal(contractAddress32, 'Addresses do not match.')
    })
  })

  describe('ovmADDRESS', async () => {
    it('reverts when ADDRESS is not set', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await executeTestTransaction(
          executionManager,
          contractAddress,
          'ovmADDRESS',
          []
        )
      })
    })

    it('properly retrieves ADDRESS when address is set', async () => {
      const result = await executeTestTransaction(
        executionManager,
        contractAddress,
        'callThroughExecutionManager',
        [contract2Address32, OVM_METHOD_IDS.getADDRESS]
      )

      log.debug(`ADDRESS result: ${result}`)

      should.exist(result, 'Result should exist!')
      result.should.equal(contract2Address32, 'Addresses do not match.')
    })
  })

  describe('ovmTIMESTAMP', async () => {
    it('properly retrieves TIMESTAMP', async () => {
      const timestamp: number = getCurrentTime()
      const result = await executeTestTransaction(
        executionManager,
        contractAddress,
        'callThroughExecutionManager',
        [contract2Address32, OVM_METHOD_IDS.getTIMESTAMP]
      )

      log.debug(`TIMESTAMP result: ${result}`)

      should.exist(result, 'Result should exist!')
      hexStrToNumber(result).should.be.gte(
        timestamp,
        'Timestamps do not match.'
      )
    })
  })

  describe('ovmCHAINID', async () => {
    it('properly retrieves CHAINID', async () => {
      const chainId: number = 108
      const result = await executeTestTransaction(
        executionManager,
        contractAddress,
        'callThroughExecutionManager',
        [contract2Address32, OVM_METHOD_IDS.getCHAINID]
      )

      log.debug(`CHAINID result: ${result}`)

      should.exist(result, 'Result should exist!')
      hexStrToNumber(result).should.be.equal(chainId, 'ChainIds do not match.')
    })
  })

  describe('ovmGASLIMIT', async () => {
    it('properly retrieves GASLIMIT', async () => {
      const result = await executeTestTransaction(
        executionManager,
        contractAddress,
        'callThroughExecutionManager',
        [contract2Address32, OVM_METHOD_IDS.getGASLIMIT]
      )

      log.debug(`GASLIMIT result: ${result}`)

      should.exist(result, 'Result should exist!')
      hexStrToNumber(result).should.equal(GAS_LIMIT, 'Gas limits do not match.')
    })
  })

  describe('ovmQueueOrigin', async () => {
    it('gets Queue Origin when it is 0', async () => {
      const queueOrigin: string = '00'.repeat(32)
      const result = await executeTestTransaction(
        executionManager,
        contractAddress,
        'callThroughExecutionManager',
        [contract2Address32, OVM_METHOD_IDS.getQueueOrigin]
      )

      log.debug(`QUEUE ORIGIN result: ${result}`)

      should.exist(result, 'Result should exist!')
      remove0x(result).should.equal(queueOrigin, 'Queue origins do not match.')
    })

    it('properly retrieves Queue Origin when queue origin is set', async () => {
      const queueOrigin: string = '00'.repeat(30) + '1111'
      const result = await executeTestTransaction(
        executionManager,
        contractAddress,
        'callThroughExecutionManager',
        [contract2Address32, OVM_METHOD_IDS.getQueueOrigin],
        add0x(queueOrigin)
      )

      log.debug(`QUEUE ORIGIN result: ${result}`)

      should.exist(result, 'Result should exist!')
      remove0x(result).should.equal(queueOrigin, 'Queue origins do not match.')
    })
  })

  describe('ovmBlockGasLimit', async () => {
    it('should retrieve the block gas limit', async () => {
      const result = await executeTestTransaction(
        executionManager,
        contractAddress,
        'callThroughExecutionManager',
        [contract2Address32, OVM_METHOD_IDS.ovmBlockGasLimit]
      )

      const resultNum = BigNumber.from(result).toNumber()
      resultNum.should.equal(GAS_LIMIT, 'Block gas limit was incorrect')
    })
  })

  describe('isStaticContext', async () => {
    it('should be true when inside a static context', async () => {
      const result = await executeTestTransaction(
        executionManager,
        contractAddress,
        'staticCallThroughExecutionManager',
        [contract2Address32, OVM_METHOD_IDS.isStaticContext]
      )

      const resultNum = BigNumber.from(result).toNumber()
      resultNum.should.equal(1, 'Context is not static but should be')
    })

    it('should be false when not in a static context', async () => {
      const result = await executeTestTransaction(
        executionManager,
        contractAddress,
        'callThroughExecutionManager',
        [contract2Address32, OVM_METHOD_IDS.isStaticContext]
      )

      const resultNum = BigNumber.from(result).toNumber()
      resultNum.should.equal(0, 'Context is static but should not be')
    })
  })

  describe('ovmORIGIN', async () => {
    it('should give us the origin of the transaction', async () => {
      const origin = await wallet.getAddress()
      const result = await executeTestTransaction(
        executionManager,
        contractAddress,
        'callThroughExecutionManager',
        [contract2Address32, OVM_METHOD_IDS.ovmORIGIN],
        ZERO_ADDRESS,
        origin
      )

      result.should.equal(
        addressToBytes32Address(origin),
        'Returned origin is incorrect'
      )
    })

    it('should revert if the transaction has no origin', async () => {
      await TestUtils.assertThrowsAsync(async () => {
        await executeTestTransaction(
          executionManager,
          contractAddress,
          'callThroughExecutionManager',
          [contract2Address32, OVM_METHOD_IDS.ovmORIGIN],
          ZERO_ADDRESS,
          ZERO_ADDRESS
        )
      })
    })
  })
})
