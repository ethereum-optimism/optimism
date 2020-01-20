import '../setup'

/* External Imports */
import { getLogger } from '@pigi/core-utils'
import { newInMemoryDB, SparseMerkleTreeImpl } from '@pigi/core-db'
import { createMockProvider, deployContract, getWallets } from 'ethereum-waffle'

const log = getLogger('state-manager', true)

/* Contract Imports */
import * as ExecutionManager from '../../build/contracts/ExecutionManager.json'
import * as ContractAddressGenerator from '../../build/contracts/ContractAddressGenerator.json'
import * as RLPEncode from '../../build/contracts/RLPEncode.json'
import { Contract, ContractFactory, Wallet, utils } from 'ethers'

/* Begin tests */
describe('ExecutionManager', () => {
  const provider = createMockProvider()
  const [wallet1, wallet2] = getWallets(provider)
  let executionManager
  let contractAddressGenerator
  let rlpEncode

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
   * Test hello world!
   */
  describe('Hello World!', async () => {
    it('Hello World!', async () => {
      log.info('Hello World!')
    })
  })
})
