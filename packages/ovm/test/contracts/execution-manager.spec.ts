import '../setup'

/* External Imports */
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'
import { getLogger, add0x } from '@pigi/core-utils'
import { newInMemoryDB, SparseMerkleTreeImpl } from '@pigi/core-db'

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import * as SimpleStorage from '../../build/contracts/SimpleStorage.json'
import * as ContractAddressGenerator from '../../build/contracts/ContractAddressGenerator.json'
import * as RLPEncode from '../../build/contracts/RLPEncode.json'
import { Contract, ContractFactory, Wallet, utils } from 'ethers'

const log = getLogger('execution-manager', true)

/*********
 * TESTS *
 *********/

describe('ExecutionManager', () => {
  const provider = createMockProvider()
  const [wallet1, wallet2] = getWallets(provider)
  let executionManager
  let contractAddressGenerator
  let rlpEncode
  // Useful constants
  const ONE_FILLED_BYTES_32 = '0x' + '11'.repeat(32)
  const TWO_FILLED_BYTES_32 = '0x' + '22'.repeat(32)

  /* Link libraries before tests */
  before(async () => {
    rlpEncode = await deployContract(wallet1, RLPEncode, [], {
      gasLimit: 6700000,
    })
    contractAddressGenerator = await deployContract(
      wallet1,
      ContractAddressGenerator,
      [rlpEncode.address],
      {
        gasLimit: 6700000,
      }
    )
  })

  /* Deploy contracts before each test */
  beforeEach(async () => {
    // Deploy the execution manager
    executionManager = await deployContract(
      wallet1,
      ExecutionManager,
      [
        '0x' + '00'.repeat(20),
        contractAddressGenerator.address,
        '0x' + '00'.repeat(20),
      ],
      {
        gasLimit: 6700000,
      }
    )
  })

  /*
   * Test CREATE opcode
   */
  describe('ovmCREATE', async () => {
    it('does not throw when passed bytecode', async () => {
      const deployTx = new ContractFactory(
        SimpleStorage.abi,
        SimpleStorage.bytecode
      ).getDeployTransaction(executionManager.address)
      // Call CREATE
      const tx1 = await executionManager.ovmCREATE(deployTx.data)
      // Get the reciept
      const reciept1 = await provider.getTransactionReceipt(tx1.hash)
      // Verify the log data exists
      reciept1.logs[0].should.have.property('data')
      // TODO: Check the actual output for the expected data
    })
  })

  /*
   * Test SSTORE opcode
   */
  describe('ovmSSTORE', async () => {
    it('successfully stores without throwing', async () => {
      await executionManager.ovmSSTORE(ONE_FILLED_BYTES_32, TWO_FILLED_BYTES_32)
    })
  })

  /*
   * Test SLOAD opcode
   */
  describe('ovmSLOAD', async () => {
    it('loads a value immediately after it is stored', async () => {
      await executionManager.ovmSSTORE(ONE_FILLED_BYTES_32, TWO_FILLED_BYTES_32)
      const two = await executionManager.ovmSLOAD(ONE_FILLED_BYTES_32)
      // It should load the value which we just set
      two.should.equal(TWO_FILLED_BYTES_32)
    })
  })
})
