import { expect } from '../../setup'

/* External Imports */
import * as path from 'path'
import { ethers } from '@nomiclabs/buidler'
import { getLogger, TestUtils } from '@eth-optimism/core-utils'
import * as solc from '@eth-optimism/solc-transpiler'
import { Contract, ContractFactory, Signer } from 'ethers'
import { keccak256 } from 'ethers/utils'

/* Internal Imports */
import {
  makeAccountStorageProofTest,
  makeAccountStorageUpdateTest,
  AccountStorageProofTest,
  AccountStorageUpdateTest,
  compile,
  DEFAULT_OPCODE_WHITELIST_MASK,
  GAS_LIMIT
} from '../../test-helpers'

/* Logging */
const log = getLogger('state-transitioner', true)

const DUMMY_ACCOUNT_ADDRESSES = [
  '0x548855F6073c3430285c61Ed0ABf62F12084aA41',
  '0xD80e66Cbc34F06d24a0a4fDdD6f2aDB41ac1517D',
  '0x069889F3DC507DdA244d19b5f24caDCDd2a735c2',
  '0x808E5eCe9a8EA2cdce515764139Ee24bEF7098b4',
]

const CORRECT_ACCOUNT_STATE = {
  nonce: 0,
  balance: 0,
  storageRoot: null,
  codeHash: null,
}

const EMPTY_ACCOUNT_STATE = {
  nonce: 0,
  balance: 0,
  storageRoot: null,
  codeHash: null,
}

const DUMMY_ACCOUNT_STORAGE = [
  {
    key: keccak256('0x123'),
    val: keccak256('0x456'),
  },
  {
    key: keccak256('0x123123'),
    val: keccak256('0x456456'),
  },
  {
    key: keccak256('0x123123123'),
    val: keccak256('0x456456456'),
  },
]

const DUMMY_STATE_TRIE = {
  [DUMMY_ACCOUNT_ADDRESSES[0]]: {
    state: EMPTY_ACCOUNT_STATE,
    storage: DUMMY_ACCOUNT_STORAGE,
  },
  [DUMMY_ACCOUNT_ADDRESSES[1]]: {
    state: EMPTY_ACCOUNT_STATE,
    storage: DUMMY_ACCOUNT_STORAGE,
  },
  [DUMMY_ACCOUNT_ADDRESSES[2]]: {
    state: EMPTY_ACCOUNT_STATE,
    storage: DUMMY_ACCOUNT_STORAGE,
  }
}

const makeStateTrie = (account: string, state: any, storage: any[]): any => {
  return {
    [account]: {
      state,
      storage,
    },
    ...DUMMY_STATE_TRIE
  }
}

const getCodeHash = async (provider: any, address: string): Promise<string> => {
  return keccak256(await provider.getCode(address))
}

/* Begin tests */
describe('StateTransitioner', () => {
  let wallet: Signer
  before(async () => {
    ;[wallet] = await ethers.getSigners()
  })

  let ExecutionManager: ContractFactory
  let StateTransitioner: ContractFactory
  let StateManager: ContractFactory
  let executionManager: Contract
  let SimpleStorageJson: any
  let SimpleStorage: ContractFactory
  let simpleStorage: Contract
  before(async () => {
    ExecutionManager = await ethers.getContractFactory('ExecutionManager')
    StateTransitioner = await ethers.getContractFactory('StateTransitioner')
    StateManager = await ethers.getContractFactory('PartialStateManager')

    executionManager = await ExecutionManager.deploy(
      DEFAULT_OPCODE_WHITELIST_MASK,
      '0x' + '00'.repeat(20),
      GAS_LIMIT,
      true
    )

    SimpleStorageJson = compile(solc, path.resolve(__dirname, '../../../contracts/test-helpers/SimpleStorage.sol'), {
      executionManagerAddress: executionManager.address
    }).contracts['SimpleStorage.sol'].SimpleStorage
    SimpleStorage = new ethers.ContractFactory(SimpleStorageJson.abi, SimpleStorageJson.evm.bytecode.object, wallet)
    simpleStorage = await SimpleStorage.deploy()
  })

  let test: AccountStorageProofTest
  before(async () => {
    test = await makeAccountStorageProofTest(
      makeStateTrie(
        simpleStorage.address,
        {
          nonce: 0,
          balance: 0,
          storageRoot: null,
          codeHash: await getCodeHash(ethers.provider, simpleStorage.address)
        },
        DUMMY_ACCOUNT_STORAGE
      ),
      simpleStorage.address,
      DUMMY_ACCOUNT_STORAGE[0].key
    )
  })

  let stateTransitioner: Contract
  let stateManager: Contract
  beforeEach(async () => {
    stateTransitioner = await StateTransitioner.deploy(
      10,
      test.stateTrieRoot,
      executionManager.address
    )
    stateManager = StateManager.attach(await stateTransitioner.stateManager())
  })

  describe('Initialization', async () => {
    it('sets the fraud verifier address to the deployer', async () => {
      const fraudVerifierAddress = await stateTransitioner.fraudVerifier()
      fraudVerifierAddress.should.equal(await wallet.getAddress())
    })
  })

  describe('Pre-Execution', async () => {
    describe('proveContractInclusion(...)', async () => {
      it('should correctly prove inclusion of a valid contract', async () => {
        await stateTransitioner.proveContractInclusion(
          simpleStorage.address,
          simpleStorage.address,
          0,
          test.stateTrieWitness
        )

        expect(await stateManager.isVerifiedContract(
          simpleStorage.address
        )).to.equal(true)
      })

      it('should correctly reject inclusion of a contract with an invalid nonce', async () => {
        try {
          await stateTransitioner.proveContractInclusion(
            simpleStorage.address,
            simpleStorage.address,
            123, // Wrong nonce.
            test.stateTrieWitness
          )
        } catch (e) {
          expect(e.toString()).to.contain('Invalid account state provided.')
        }

        expect(await stateManager.isVerifiedContract(
          simpleStorage.address
        )).to.equal(false)
      })
    })

    describe('proveStorageSlotInclusion(...)', async () => {
      it('should correctly prove inclusion of a valid storage slot', async () => {
        await stateTransitioner.proveContractInclusion(
          simpleStorage.address,
          simpleStorage.address,
          0,
          test.stateTrieWitness
        )

        await stateTransitioner.proveStorageSlotInclusion(
          simpleStorage.address,
          DUMMY_ACCOUNT_STORAGE[0].key,
          DUMMY_ACCOUNT_STORAGE[0].val,
          test.stateTrieWitness,
          test.storageTrieWitness
        )

        expect(await stateManager.isVerifiedStorage(
          simpleStorage.address,
          DUMMY_ACCOUNT_STORAGE[0].key
        )).to.equal(true)
      })

      it('should correctly reject inclusion of an invalid storage slot', async () => {
        await stateTransitioner.proveContractInclusion(
          simpleStorage.address,
          simpleStorage.address,
          0,
          test.stateTrieWitness
        )

        try {
          await stateTransitioner.proveStorageSlotInclusion(
            simpleStorage.address,
            DUMMY_ACCOUNT_STORAGE[0].key,
            DUMMY_ACCOUNT_STORAGE[1].val, // Different value.
            test.stateTrieWitness,
            test.storageTrieWitness
          )
        } catch (e) {
          expect(e.toString()).to.contain('Invalid account state provided.')
        }

        expect(await stateManager.isVerifiedStorage(
          simpleStorage.address,
          DUMMY_ACCOUNT_STORAGE[0].key
        )).to.equal(false)
      })
    })
  })

  describe('applyTransaction(...)', async () => {
    it('should succeed if no state is retrieved', async () => {
      await stateTransitioner.proveContractInclusion(
        simpleStorage.address,
        simpleStorage.address,
        0,
        test.stateTrieWitness
      )

      const calldata = SimpleStorage.interface.encodeFunctionData(
        'setStorage',
        [
          keccak256('0xabc'),
          keccak256('0xdef')
        ]
      )

      await stateTransitioner.setTransactionData({
        timestamp: 1,
        queueOrigin: 1,
        ovmEntrypoint: simpleStorage.address,
        callBytes: calldata,
        fromAddress: simpleStorage.address,
        l1MsgSenderAddress: await wallet.getAddress(),
        allowRevert: false
      })

      await stateTransitioner.applyTransaction()
      expect(await stateTransitioner.currentTransitionPhase()).to.equal(1)
    })

    it('should fail if attempting to access uninitialized state', async () => {
      await stateTransitioner.proveContractInclusion(
        simpleStorage.address,
        simpleStorage.address,
        0,
        test.stateTrieWitness
      )
      
      // Attempting a `getStorage` call to a key that hasn't been proven.
      const calldata = SimpleStorage.interface.encodeFunctionData(
        'getStorage',
        [
          keccak256('0xabc') 
        ]
      )

      await stateTransitioner.setTransactionData({
        timestamp: 1,
        queueOrigin: 1,
        ovmEntrypoint: simpleStorage.address,
        callBytes: calldata,
        fromAddress: simpleStorage.address,
        l1MsgSenderAddress: await wallet.getAddress(),
        allowRevert: false
      })

      await TestUtils.assertRevertsAsync(
        'Detected an invalid state access.',
        async () => {
          await stateTransitioner.applyTransaction()
        }
      )

      expect(await stateTransitioner.currentTransitionPhase()).to.equal(0)
    })

    it('should fail if attempting to access an uninitialized contract', async () => {
      // Haven't proven contract inclusion here.

      const calldata = SimpleStorage.interface.encodeFunctionData(
        'setStorage',
        [
          keccak256('0xabc'),
          keccak256('0xdef')
        ]
      )

      await stateTransitioner.setTransactionData({
        timestamp: 1,
        queueOrigin: 1,
        ovmEntrypoint: simpleStorage.address,
        callBytes: calldata,
        fromAddress: simpleStorage.address,
        l1MsgSenderAddress: await wallet.getAddress(),
        allowRevert: false
      })

      await TestUtils.assertRevertsAsync(
        'Detected an invalid state access.',
        async () => {
          await stateTransitioner.applyTransaction()
        }
      )

      expect(await stateTransitioner.currentTransitionPhase()).to.equal(0)
    })
  })

  describe('Post-Execution', async () => {
    describe('proveUpdatedStorageSlot(...)', async () => {

    })

    describe('proveUpdatedContract(...)', async () => {

    })

    describe('completeTransition(...)', async () => {

    })
  })
})
