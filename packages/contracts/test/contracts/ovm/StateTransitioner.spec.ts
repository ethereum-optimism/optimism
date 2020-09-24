import { expect } from '../../setup'

/* External Imports */
import * as path from 'path'
import * as rlp from 'rlp'
import { ethers } from '@nomiclabs/buidler'
import {
  getLogger,
  TestUtils,
  remove0x,
  numberToHexString,
  hexStrToBuf,
  bufToHexString,
} from '@eth-optimism/core-utils'
import * as solc from '@eth-optimism/solc'
import { Contract, ContractFactory, Signer, BigNumber } from 'ethers'
import { keccak256 } from 'ethers/utils'
import { cloneDeep } from 'lodash'

/* Internal Imports */
import {
  makeAccountStorageProofTest,
  makeAccountStorageUpdateTest,
  AccountStorageProofTest,
  AccountStorageUpdateTest,
  StateTrieMap,
  StateTrieNode,
  TrieNode,
  compile,
  makeStateTrieUpdateTest,
  StateTrieUpdateTest,
  toHexString,
  makeAddressResolver,
  AddressResolverMapping,
  GAS_LIMIT,
  makeStateTrie,
} from '../../test-helpers'
import { BaseTrie } from 'merkle-patricia-tree'

/* Logging */
const log = getLogger('state-transitioner', true)

const NULL_ADDRESS = '0x' + '00'.repeat(20)
const DUMMY_ACCOUNT_ADDRESSES = [
  '0x548855F6073c3430285c61Ed0ABf62F12084aA41',
  '0xD80e66Cbc34F06d24a0a4fDdD6f2aDB41ac1517D',
  '0x069889F3DC507DdA244d19b5f24caDCDd2a735c2',
  '0x808E5eCe9a8EA2cdce515764139Ee24bEF7098b4',
]

// gas metering always causes some storage slots to be updated
const DEFAULT_TX_NUM_STORAGE_UPDATES: number = 0

const SUFFICIENT_APPLY_TRANSACTION_GAS = GAS_LIMIT * 2

interface OVMTransactionData {
  timestamp: number
  blockNumber: number
  queueOrigin: number
  ovmEntrypoint: string
  callBytes: string
  fromAddress: string
  l1MsgSenderAddress: string
  gasLimit: number
  allowRevert: boolean
}

const makeDummyTransaction = (calldata: string): OVMTransactionData => {
  return {
    timestamp: Math.floor(Date.now() / 1000),
    blockNumber: 0,
    queueOrigin: 0,
    ovmEntrypoint: NULL_ADDRESS,
    callBytes: calldata,
    fromAddress: NULL_ADDRESS,
    l1MsgSenderAddress: NULL_ADDRESS,
    gasLimit: GAS_LIMIT,
    allowRevert: false,
  }
}

const EMPTY_ACCOUNT_STATE = (): StateTrieNode => {
  return cloneDeep({
    nonce: 0,
    balance: 0,
    storageRoot: null,
    codeHash: null,
  })
}

const STATE_TRANSITIONER_PHASES = {
  PRE_EXECUTION: 0,
  POST_EXECUTION: 1,
  COMPLETE: 2,
}

const DUMMY_ACCOUNT_STORAGE = (): TrieNode[] => {
  return cloneDeep([
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
  ])
}

// OVM address where we handle some persistent state that is used directly by the EM.  There will never be code deployed here, we just use it to persist this chain-related metadata.
const METADATA_STORAGE_ADDRESS = NULL_ADDRESS
// Storage keys which the EM will directly use to persist the different pieces of metadata:
// Storage slot where we will store the cumulative sequencer tx gas spent
const CUMULATIVE_SEQUENCED_GAS_STORAGE_KEY =
  '0x0000000000000000000000000000000000000000000000000000000000000001'
// Storage slot where we will store the cumulative queued tx gas spent
const CUMULATIVE_QUEUED_GAS_STORAGE_KEY =
  '0x0000000000000000000000000000000000000000000000000000000000000002'
// Storage slot where we will store the start of the current gas rate limit epoch
const GAS_RATE_LMIT_EPOCH_START_STORAGE_KEY =
  '0x0000000000000000000000000000000000000000000000000000000000000003'
// Storage slot where we will store what the cumulative sequencer gas was at the start of the last epoch
const CUMULATIVE_SEQUENCED_GAS_AT_EPOCH_START_STORAGE_KEY =
  '0x0000000000000000000000000000000000000000000000000000000000000004'
// Storage slot where we will store what the cumulative queued gas was at the start of the last epoch
const CUMULATIVE_QUEUED_GAS_AT_EPOCH_START_STORAGE_KEY =
  '0x0000000000000000000000000000000000000000000000000000000000000005'

const DEPLOYER_WHITELIST_ADDRESS = '0x4200000000000000000000000000000000000002'
const INITIALIZED_KEY =
  '0x0000000000000000000000000000000000000000000000000000000000000010'
const OWNER_KEY =
  '0x0000000000000000000000000000000000000000000000000000000000000011'
const ALLOW_ARBITRARY_DEPLOYMENT =
  '0x0000000000000000000000000000000000000000000000000000000000000012'

const initialCumulativeSequencedGas = 0
const initialCumulativeQueuedGas = 0
const initialGasRateLimitEpochStart = 0
const initialCumulativeSequencedGasAtEpochStart = 0
const initialCumulativeQueuedGasAtEpochStart = 0

const INITIAL_OVM_GAS_STORAGE = (): any => {
  return cloneDeep([
    {
      key: CUMULATIVE_SEQUENCED_GAS_STORAGE_KEY,
      val: numberToHexString(initialCumulativeSequencedGas, 32),
    },
    {
      key: CUMULATIVE_QUEUED_GAS_STORAGE_KEY,
      val: numberToHexString(initialCumulativeQueuedGas, 32),
    },
    {
      key: GAS_RATE_LMIT_EPOCH_START_STORAGE_KEY,
      val: numberToHexString(initialGasRateLimitEpochStart, 32),
    },
    {
      key: CUMULATIVE_SEQUENCED_GAS_AT_EPOCH_START_STORAGE_KEY,
      val: numberToHexString(initialCumulativeSequencedGasAtEpochStart, 32),
    },
    {
      key: CUMULATIVE_QUEUED_GAS_AT_EPOCH_START_STORAGE_KEY,
      val: numberToHexString(initialCumulativeQueuedGasAtEpochStart, 32),
    },
  ])
}

const proveOVMGasMetadataStorage = async (
  stateTransitioner: any,
  stateTrie: any
) => {
  const stateTrieWitness = await BaseTrie.prove(
    stateTrie.trie,
    hexStrToBuf(METADATA_STORAGE_ADDRESS)
  )
  await stateTransitioner.proveContractInclusion(
    METADATA_STORAGE_ADDRESS,
    METADATA_STORAGE_ADDRESS,
    0,
    rlp.encode(stateTrieWitness)
  )
  const storageTrie = stateTrie.storage[METADATA_STORAGE_ADDRESS]

  for (const { key, val } of INITIAL_OVM_GAS_STORAGE()) {
    const storageWitness = await BaseTrie.prove(storageTrie, hexStrToBuf(key))
    await stateTransitioner.proveStorageSlotInclusion(
      METADATA_STORAGE_ADDRESS,
      key,
      val,
      rlp.encode(stateTrieWitness),
      rlp.encode(storageWitness)
    )
  }
}

// A populated state trie layout, with OVM gas metering state pre-populated
const DUMMY_INITIAL_STATE_TRIE = {
  [DUMMY_ACCOUNT_ADDRESSES[0]]: {
    state: EMPTY_ACCOUNT_STATE(),
    storage: DUMMY_ACCOUNT_STORAGE(),
  },
  [DUMMY_ACCOUNT_ADDRESSES[1]]: {
    state: EMPTY_ACCOUNT_STATE(),
    storage: DUMMY_ACCOUNT_STORAGE(),
  },
  [DUMMY_ACCOUNT_ADDRESSES[2]]: {
    state: EMPTY_ACCOUNT_STATE(),
    storage: DUMMY_ACCOUNT_STORAGE(),
  },
  [METADATA_STORAGE_ADDRESS]: {
    state: EMPTY_ACCOUNT_STATE(),
    storage: INITIAL_OVM_GAS_STORAGE(),
  },
  [DEPLOYER_WHITELIST_ADDRESS]: {
    state: EMPTY_ACCOUNT_STATE(),
    storage: DUMMY_ACCOUNT_STORAGE(),
  },
}

const encodeTransaction = (transaction: OVMTransactionData): string => {
  return toHexString(
    rlp.encode([
      transaction.timestamp,
      transaction.queueOrigin,
      transaction.ovmEntrypoint,
      transaction.callBytes,
      transaction.fromAddress,
      transaction.l1MsgSenderAddress,
      transaction.gasLimit,
      transaction.allowRevert ? 1 : 0,
    ])
  )
}

const makeInitialStateTrie = (
  account: string,
  state: any,
  storage: any[]
): any => {
  return {
    [account]: {
      state,
      storage,
    },
    ...DUMMY_INITIAL_STATE_TRIE,
  }
}

const getCodeHash = async (provider: any, address: string): Promise<string> => {
  return keccak256(await provider.getCode(address))
}

const makeTransactionData = async (
  TargetFactory: ContractFactory,
  target: Contract,
  wallet: Signer,
  functionName: string,
  functionArgs: any[]
): Promise<OVMTransactionData> => {
  const calldata = TargetFactory.interface.encodeFunctionData(
    functionName,
    functionArgs
  )

  return {
    timestamp: 1,
    blockNumber: 1,
    queueOrigin: 1,
    ovmEntrypoint: target.address,
    callBytes: calldata,
    fromAddress: target.address,
    l1MsgSenderAddress: await wallet.getAddress(),
    gasLimit: GAS_LIMIT,
    allowRevert: false,
  }
}

const proveAllStorageUpdates = async (
  stateTransitioner: Contract,
  stateManager: Contract,
  stateTrie: StateTrieMap
): Promise<string> => {
  let updateTest: AccountStorageUpdateTest
  let trie = cloneDeep(stateTrie)

  while ((await stateManager.updatedStorageSlotCounter()) > 0) {
    const [
      storageSlotContract,
      storageSlotKey,
      storageSlotValue,
    ] = await stateManager.peekUpdatedStorageSlot()

    updateTest = await makeAccountStorageUpdateTest(
      trie,
      storageSlotContract,
      storageSlotKey,
      storageSlotValue
    )

    await stateTransitioner.proveUpdatedStorageSlot(
      updateTest.stateTrieWitness,
      updateTest.storageTrieWitness
    )

    trie = makeModifiedTrie(trie, [
      {
        address: storageSlotContract,
        storage: [
          {
            key: storageSlotKey,
            val: storageSlotValue,
          },
        ],
      },
    ])
  }

  return updateTest.newStateTrieRoot
}

const proveAllContractUpdates = async (
  stateTransitioner: Contract,
  stateManager: Contract,
  stateTrie: StateTrieMap
): Promise<string> => {
  let updateTest: StateTrieUpdateTest
  let trie = cloneDeep(stateTrie)

  while ((await stateManager.updatedContractsCounter()) > 0) {
    const [
      updatedContract,
      updatedContractNonce,
      updatedCodeHash,
    ] = await stateManager.peekUpdatedContract()

    const updatedAccountState = {
      ...(updatedContract in trie
        ? trie[updatedContract].state
        : EMPTY_ACCOUNT_STATE()),
      ...{
        nonce: updatedContractNonce.toNumber(),
      },
    }

    if (updatedCodeHash !== '0x' + '00'.repeat(32)) {
      updatedAccountState.codeHash = updatedCodeHash
    }

    updateTest = await makeStateTrieUpdateTest(
      trie,
      updatedContract,
      updatedAccountState
    )

    await stateTransitioner.proveUpdatedContract(updateTest.stateTrieWitness)

    trie = makeModifiedTrie(trie, [
      {
        address: updatedContract,
        state: updatedAccountState,
      },
    ])
  }

  return updateTest.newStateTrieRoot
}

const getMappingStorageSlot = (key: string, index: number): string => {
  const hexIndex = remove0x(BigNumber.from(index).toHexString()).padStart(
    64,
    '0'
  )
  return keccak256(key + hexIndex)
}

const initStateTransitioner = async (
  StateTransitioner: ContractFactory,
  StateManager: ContractFactory,
  addressResolver: Contract,
  trieMapping: any,
  transactionData: OVMTransactionData
): Promise<[Contract, Contract, OVMTransactionData]> => {
  const stateTrie = await makeStateTrie(trieMapping)
  const stateTrieRoot = bufToHexString(stateTrie.trie.root)
  const stateTransitioner = await StateTransitioner.deploy(
    addressResolver.address,
    10,
    stateTrieRoot,
    keccak256(encodeTransaction(transactionData))
  )
  const stateManager = StateManager.attach(
    await stateTransitioner.stateManager()
  )

  await proveOVMGasMetadataStorage(stateTransitioner, stateTrie)

  return [stateTransitioner, stateManager, transactionData]
}

interface StateTrieModification {
  address: string
  state?: Partial<StateTrieNode>
  storage?: TrieNode[]
}

const makeModifiedTrie = (
  stateTrie: StateTrieMap,
  modifications: StateTrieModification[]
): StateTrieMap => {
  const trie = cloneDeep(stateTrie)

  for (let modification of modifications) {
    modification = cloneDeep(modification)

    if (!(modification.address in trie)) {
      trie[modification.address] = {
        state: {
          ...EMPTY_ACCOUNT_STATE(),
          ...modification.state,
        },
        storage: modification.storage || [],
      }
    } else {
      if (modification.state) {
        trie[modification.address].state = {
          ...trie[modification.address].state,
          ...modification.state,
        }
      }

      if (modification.storage) {
        for (const element of modification.storage) {
          const hasKey = trie[modification.address].storage.some((kv: any) => {
            return kv.key === element.key
          })

          if (!hasKey) {
            trie[modification.address].storage.push({
              key: element.key,
              val: element.val,
            })
          } else {
            trie[modification.address].storage = trie[
              modification.address
            ].storage.map((kv: any) => {
              if (kv.key === element.key) {
                kv.val = element.val
              }

              return kv
            })
          }
        }
      }
    }
  }

  return trie
}

/* Begin tests */
describe('StateTransitioner', () => {
  let wallet: Signer
  before(async () => {
    ;[wallet] = await ethers.getSigners()
  })

  let resolver: AddressResolverMapping
  before(async () => {
    resolver = await makeAddressResolver(wallet)
  })

  let executionManager: Contract
  before(async () => {
    executionManager = resolver.contracts.executionManager
  })

  let StateTransitioner: ContractFactory
  let StateManager: ContractFactory
  let FraudTesterJson: any
  let MicroFraudTesterJson: any
  let FraudTester: ContractFactory
  let fraudTester: Contract
  before(async () => {
    StateTransitioner = await ethers.getContractFactory('StateTransitioner')
    StateManager = await ethers.getContractFactory('PartialStateManager')

    console.log(
      `path is ${path.resolve(
        __dirname,
        '../../../contracts/test-helpers/FraudTester.sol'
      )}`
    )
    console.log(
      `compile esult is ${JSON.stringify(
        compile(
          solc,
          path.resolve(
            __dirname,
            '../../../contracts/test-helpers/FraudTester.sol'
          )
        ).errors
      )}`
    )
    const AllFraudTestJson = compile(
      solc,
      path.resolve(__dirname, '../../../contracts/test-helpers/FraudTester.sol')
    ).contracts['FraudTester.sol']
    FraudTesterJson = AllFraudTestJson.FraudTester
    MicroFraudTesterJson = AllFraudTestJson.MicroFraudTester

    FraudTester = new ethers.ContractFactory(
      FraudTesterJson.abi,
      FraudTesterJson.evm.bytecode.object,
      wallet
    )
    fraudTester = await FraudTester.deploy()
  })

  let stateTrie: any
  let test: AccountStorageProofTest
  let stateTransitioner: Contract
  let stateManager: Contract
  let dummyTransactionData: OVMTransactionData
  let initializedDummyTxSnapshot
  before(async () => {
    stateTrie = makeInitialStateTrie(
      fraudTester.address,
      {
        nonce: 0,
        balance: 0,
        storageRoot: null,
        codeHash: await getCodeHash(ethers.provider, fraudTester.address),
      },
      DUMMY_ACCOUNT_STORAGE()
    )

    test = await makeAccountStorageProofTest(
      stateTrie,
      fraudTester.address,
      DUMMY_ACCOUNT_STORAGE()[0].key
    )
    ;[
      stateTransitioner,
      stateManager,
      dummyTransactionData,
    ] = await initStateTransitioner(
      StateTransitioner,
      StateManager,
      resolver.addressResolver,
      stateTrie,
      makeDummyTransaction('0x00')
    )
    initializedDummyTxSnapshot = await ethers.provider.send('evm_snapshot', [])
  })

  const revertToDummyTxSnapshot = async () => {
    await ethers.provider.send('evm_revert', [initializedDummyTxSnapshot])
    // evm_revert deletes the snapshot so reset it right after
    initializedDummyTxSnapshot = await ethers.provider.send('evm_snapshot', [])
  }

  let transactionData: OVMTransactionData
  beforeEach(async () => {
    await revertToDummyTxSnapshot()
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
          fraudTester.address,
          fraudTester.address,
          0,
          test.stateTrieWitness
        )

        expect(
          await stateManager.isVerifiedContract(fraudTester.address)
        ).to.equal(true)
      })

      it('should correctly reject inclusion of a contract with an invalid nonce', async () => {
        await ethers.provider.send('evm_revert', [initializedDummyTxSnapshot])
        await TestUtils.assertRevertsAsync(async () => {
          await stateTransitioner.proveContractInclusion(
            fraudTester.address,
            fraudTester.address,
            123, // Wrong nonce.
            test.stateTrieWitness
          )
        }, 'Invalid account state provided.')

        expect(
          await stateManager.isVerifiedContract(fraudTester.address)
        ).to.equal(false)
      })
    })

    describe('proveStorageSlotInclusion(...)', async () => {
      it('should correctly prove inclusion of a valid storage slot', async () => {
        await stateTransitioner.proveContractInclusion(
          fraudTester.address,
          fraudTester.address,
          0,
          test.stateTrieWitness
        )

        await stateTransitioner.proveStorageSlotInclusion(
          fraudTester.address,
          DUMMY_ACCOUNT_STORAGE()[0].key,
          DUMMY_ACCOUNT_STORAGE()[0].val,
          test.stateTrieWitness,
          test.storageTrieWitness
        )

        expect(
          await stateManager.isVerifiedStorage(
            fraudTester.address,
            DUMMY_ACCOUNT_STORAGE()[0].key
          )
        ).to.equal(true)
      })

      it('should correctly reject inclusion of an invalid storage slot', async () => {
        await stateTransitioner.proveContractInclusion(
          fraudTester.address,
          fraudTester.address,
          0,
          test.stateTrieWitness
        )

        await TestUtils.assertRevertsAsync(async () => {
          await stateTransitioner.proveStorageSlotInclusion(
            fraudTester.address,
            DUMMY_ACCOUNT_STORAGE()[0].key,
            DUMMY_ACCOUNT_STORAGE()[1].val, // Different value.
            test.stateTrieWitness,
            test.storageTrieWitness
          )
        }, 'Invalid account state provided.')

        expect(
          await stateManager.isVerifiedStorage(
            fraudTester.address,
            DUMMY_ACCOUNT_STORAGE()[0].key
          )
        ).to.equal(false)
      })
    })
  })

  describe('applyTransaction(...)', async () => {
    it('should fail if supplied less gas than might be needed to executeTransaction(...)', async () => {
      TestUtils.assertRevertsAsync(async () => {
        await stateTransitioner.applyTransaction(dummyTransactionData, {
          gasLimit: GAS_LIMIT / 2,
        })
      }, 'Insufficient gas supplied to ensure L1 execution will not run out of gas before OVM transaction gas limit.')
    })
    it('should succeed if no state is accessed', async () => {
      ;[
        stateTransitioner,
        stateManager,
        transactionData,
      ] = await initStateTransitioner(
        StateTransitioner,
        StateManager,
        resolver.addressResolver,
        stateTrie,
        await makeTransactionData(
          FraudTester,
          fraudTester,
          wallet,
          'setStorage',
          [keccak256('0xabc'), keccak256('0xdef')]
        )
      )

      await stateTransitioner.proveContractInclusion(
        fraudTester.address,
        fraudTester.address,
        0,
        test.stateTrieWitness
      )

      await stateTransitioner.applyTransaction(transactionData, {
        gasLimit: SUFFICIENT_APPLY_TRANSACTION_GAS,
      })
      expect(await stateTransitioner.currentTransitionPhase()).to.equal(
        STATE_TRANSITIONER_PHASES.POST_EXECUTION
      )
    })

    it('should succeed initialized state is accessed', async () => {
      const testKey = keccak256('0xabc')
      const testKeySlot = getMappingStorageSlot(testKey, 0)
      const testVal = keccak256('0xdef')

      const trie = makeModifiedTrie(stateTrie, [
        {
          address: fraudTester.address,
          storage: [
            {
              key: testKeySlot,
              val: testVal,
            },
          ],
        },
      ])

      const accessTest = await makeAccountStorageProofTest(
        trie,
        fraudTester.address,
        testKeySlot
      )
      ;[
        stateTransitioner,
        stateManager,
        transactionData,
      ] = await initStateTransitioner(
        StateTransitioner,
        StateManager,
        resolver.addressResolver,
        trie,
        await makeTransactionData(
          FraudTester,
          fraudTester,
          wallet,
          'getStorage',
          [testKey]
        )
      )

      await stateTransitioner.proveContractInclusion(
        fraudTester.address,
        fraudTester.address,
        0,
        accessTest.stateTrieWitness
      )

      await stateTransitioner.proveStorageSlotInclusion(
        fraudTester.address,
        testKeySlot,
        testVal,
        accessTest.stateTrieWitness,
        accessTest.storageTrieWitness
      )

      await stateTransitioner.applyTransaction(transactionData, {
        gasLimit: SUFFICIENT_APPLY_TRANSACTION_GAS,
      })
      expect(await stateTransitioner.currentTransitionPhase()).to.equal(
        STATE_TRANSITIONER_PHASES.POST_EXECUTION
      )
    })

    it('should succeed when a new contract is created', async () => {
      // Attempting a `getStorage` call to a key that hasn't been proven.
      ;[
        stateTransitioner,
        stateManager,
        transactionData,
      ] = await initStateTransitioner(
        StateTransitioner,
        StateManager,
        resolver.addressResolver,
        stateTrie,
        await makeTransactionData(
          FraudTester,
          fraudTester,
          wallet,
          'createContract',
          ['0x' + MicroFraudTesterJson.evm.bytecode.object]
        )
      )

      await stateTransitioner.proveContractInclusion(
        fraudTester.address,
        fraudTester.address,
        0,
        test.stateTrieWitness
      )

      await stateTransitioner.applyTransaction(transactionData, {
        gasLimit: SUFFICIENT_APPLY_TRANSACTION_GAS,
      })
      expect(await stateTransitioner.currentTransitionPhase()).to.equal(
        STATE_TRANSITIONER_PHASES.POST_EXECUTION
      )
    })

    it('should fail if attempting to access uninitialized state', async () => {
      // Attempting a `getStorage` call to a key that hasn't been proven.
      ;[
        stateTransitioner,
        stateManager,
        transactionData,
      ] = await initStateTransitioner(
        StateTransitioner,
        StateManager,
        resolver.addressResolver,
        stateTrie,
        await makeTransactionData(
          FraudTester,
          fraudTester,
          wallet,
          'getStorage',
          [keccak256('0xabc')]
        )
      )

      await stateTransitioner.proveContractInclusion(
        fraudTester.address,
        fraudTester.address,
        0,
        test.stateTrieWitness
      )

      await TestUtils.assertRevertsAsync(async () => {
        await stateTransitioner.applyTransaction(transactionData, {
          gasLimit: SUFFICIENT_APPLY_TRANSACTION_GAS,
        })
      }, 'Detected an invalid state access.')

      expect(await stateTransitioner.currentTransitionPhase()).to.equal(
        STATE_TRANSITIONER_PHASES.PRE_EXECUTION
      )
    })

    it('should fail if attempting to access an uninitialized contract', async () => {
      ;[
        stateTransitioner,
        stateManager,
        transactionData,
      ] = await initStateTransitioner(
        StateTransitioner,
        StateManager,
        resolver.addressResolver,
        stateTrie,
        await makeTransactionData(
          FraudTester,
          fraudTester,
          wallet,
          'setStorage',
          [keccak256('0xabc'), keccak256('0xdef')]
        )
      )

      await TestUtils.assertRevertsAsync(async () => {
        await stateTransitioner.applyTransaction(transactionData, {
          gasLimit: SUFFICIENT_APPLY_TRANSACTION_GAS,
        })
      }, 'Detected an invalid state access.')

      expect(await stateTransitioner.currentTransitionPhase()).to.equal(
        STATE_TRANSITIONER_PHASES.PRE_EXECUTION
      )
    })
  })

  describe('Post-Execution', async () => {
    describe('proveUpdatedStorageSlot(...)', async () => {
      it('should correctly update when a slot has been changed', async () => {
        ;[
          stateTransitioner,
          stateManager,
          transactionData,
        ] = await initStateTransitioner(
          StateTransitioner,
          StateManager,
          resolver.addressResolver,
          stateTrie,
          await makeTransactionData(
            FraudTester,
            fraudTester,
            wallet,
            'setStorage',
            [keccak256('0xabc'), keccak256('0xdef')]
          )
        )

        await stateTransitioner.proveContractInclusion(
          fraudTester.address,
          fraudTester.address,
          0,
          test.stateTrieWitness
        )

        await stateTransitioner.applyTransaction(transactionData, {
          gasLimit: SUFFICIENT_APPLY_TRANSACTION_GAS,
        })

        expect(await stateManager.updatedStorageSlotCounter()).to.equal(
          DEFAULT_TX_NUM_STORAGE_UPDATES + 1
        )

        const newStateTrieRoot = await proveAllStorageUpdates(
          stateTransitioner,
          stateManager,
          stateTrie
        )

        expect(await stateTransitioner.stateRoot()).to.equal(newStateTrieRoot)
        expect(await stateManager.updatedStorageSlotCounter()).to.equal(0)
      })

      it('should correctly update when multiple slots have changed', async () => {
        ;[
          stateTransitioner,
          stateManager,
          transactionData,
        ] = await initStateTransitioner(
          StateTransitioner,
          StateManager,
          resolver.addressResolver,
          stateTrie,
          await makeTransactionData(
            FraudTester,
            fraudTester,
            wallet,
            'setStorageMultiple',
            [
              keccak256('0xabc'),
              keccak256('0xdef'),
              3, // Set three storage slots.
            ]
          )
        )

        await stateTransitioner.proveContractInclusion(
          fraudTester.address,
          fraudTester.address,
          0,
          test.stateTrieWitness
        )

        await stateTransitioner.applyTransaction(transactionData, {
          gasLimit: SUFFICIENT_APPLY_TRANSACTION_GAS,
        })

        expect(await stateManager.updatedStorageSlotCounter()).to.equal(
          DEFAULT_TX_NUM_STORAGE_UPDATES + 3
        )

        const newStateTrieRoot = await proveAllStorageUpdates(
          stateTransitioner,
          stateManager,
          stateTrie
        )

        expect(await stateTransitioner.stateRoot()).to.equal(newStateTrieRoot)
        expect(await stateManager.updatedStorageSlotCounter()).to.equal(0)
      }).timeout(80000)

      it('should correctly update when the same slot has changed multiple times', async () => {
        ;[
          stateTransitioner,
          stateManager,
          transactionData,
        ] = await initStateTransitioner(
          StateTransitioner,
          StateManager,
          resolver.addressResolver,
          stateTrie,
          await makeTransactionData(
            FraudTester,
            fraudTester,
            wallet,
            'setStorageMultipleSameKey',
            [
              keccak256('0xabc'),
              keccak256('0xdef'),
              3, // Set slot three times.
            ]
          )
        )

        await stateTransitioner.proveContractInclusion(
          fraudTester.address,
          fraudTester.address,
          0,
          test.stateTrieWitness
        )

        await stateTransitioner.applyTransaction(transactionData, {
          gasLimit: SUFFICIENT_APPLY_TRANSACTION_GAS,
        })

        expect(await stateManager.updatedStorageSlotCounter()).to.equal(
          DEFAULT_TX_NUM_STORAGE_UPDATES + 1
        )

        const newStateTrieRoot = await proveAllStorageUpdates(
          stateTransitioner,
          stateManager,
          stateTrie
        )

        expect(await stateTransitioner.stateRoot()).to.equal(newStateTrieRoot)
        expect(await stateManager.updatedStorageSlotCounter()).to.equal(0)
      })
    })

    describe('proveUpdatedContract(...)', async () => {
      it('should correctly update when a contract has been created', async () => {
        ;[
          stateTransitioner,
          stateManager,
          transactionData,
        ] = await initStateTransitioner(
          StateTransitioner,
          StateManager,
          resolver.addressResolver,
          stateTrie,
          await makeTransactionData(
            FraudTester,
            fraudTester,
            wallet,
            'createContract',
            ['0x' + MicroFraudTesterJson.evm.bytecode.object]
          )
        )

        await stateTransitioner.proveContractInclusion(
          fraudTester.address,
          fraudTester.address,
          0,
          test.stateTrieWitness
        )

        await stateTransitioner.applyTransaction(transactionData, {
          gasLimit: SUFFICIENT_APPLY_TRANSACTION_GAS,
        })

        // One update for each new contract, plus one nonce update for the creating contract.
        expect(await stateManager.updatedContractsCounter()).to.equal(2)

        const newStateTrieRoot = await proveAllContractUpdates(
          stateTransitioner,
          stateManager,
          stateTrie
        )

        expect(await stateTransitioner.stateRoot()).to.equal(newStateTrieRoot)
        expect(await stateManager.updatedContractsCounter()).to.equal(0)
      })

      it('should correctly update when multiple contracts have been created', async () => {
        ;[
          stateTransitioner,
          stateManager,
          transactionData,
        ] = await initStateTransitioner(
          StateTransitioner,
          StateManager,
          resolver.addressResolver,
          stateTrie,
          await makeTransactionData(
            FraudTester,
            fraudTester,
            wallet,
            'createContractMultiple',
            [
              '0x' + MicroFraudTesterJson.evm.bytecode.object,
              3, // Create three contracts.
            ]
          )
        )

        await stateTransitioner.proveContractInclusion(
          fraudTester.address,
          fraudTester.address,
          0,
          test.stateTrieWitness
        )

        await stateTransitioner.applyTransaction(transactionData, {
          gasLimit: SUFFICIENT_APPLY_TRANSACTION_GAS,
        })

        // One update for each new contract, plus one nonce update for the creating contract.
        expect(await stateManager.updatedContractsCounter()).to.equal(4)

        const newStateTrieRoot = await proveAllContractUpdates(
          stateTransitioner,
          stateManager,
          stateTrie
        )

        expect(await stateTransitioner.stateRoot()).to.equal(newStateTrieRoot)
        expect(await stateManager.updatedContractsCounter()).to.equal(0)
      })
    })

    describe('completeTransition(...)', async () => {
      it.skip('should correctly finalize when no slots are changed', async () => {
        const testKey = keccak256('0xabc')
        const testKeySlot = getMappingStorageSlot(testKey, 0)
        const testVal = keccak256('0xdef')

        const trie = makeModifiedTrie(stateTrie, [
          {
            address: fraudTester.address,
            storage: [
              {
                key: testKeySlot,
                val: testVal,
              },
            ],
          },
          {
            address: METADATA_STORAGE_ADDRESS,
            storage: INITIAL_OVM_GAS_STORAGE(),
          },
        ])

        const accessTest = await makeAccountStorageProofTest(
          trie,
          fraudTester.address,
          testKeySlot
        )
        ;[
          stateTransitioner,
          stateManager,
          transactionData,
        ] = await initStateTransitioner(
          StateTransitioner,
          StateManager,
          resolver.addressResolver,
          trie,
          await makeTransactionData(
            FraudTester,
            fraudTester,
            wallet,
            'getStorage',
            [testKey]
          )
        )

        await stateTransitioner.proveContractInclusion(
          fraudTester.address,
          fraudTester.address,
          0,
          accessTest.stateTrieWitness
        )

        await stateTransitioner.proveStorageSlotInclusion(
          fraudTester.address,
          testKeySlot,
          testVal,
          accessTest.stateTrieWitness,
          accessTest.storageTrieWitness
        )

        await stateTransitioner.applyTransaction(transactionData, {
          gasLimit: SUFFICIENT_APPLY_TRANSACTION_GAS,
        })
        expect(await stateManager.updatedStorageSlotCounter()).to.equal(
          DEFAULT_TX_NUM_STORAGE_UPDATES + 0
        )

        await proveAllStorageUpdates(stateTransitioner, stateManager, trie)

        await stateTransitioner.completeTransition()
        expect(await stateTransitioner.currentTransitionPhase()).to.equal(
          STATE_TRANSITIONER_PHASES.COMPLETE
        )
      })

      it('should correctly finalize when storage slots are changed', async () => {
        ;[
          stateTransitioner,
          stateManager,
          transactionData,
        ] = await initStateTransitioner(
          StateTransitioner,
          StateManager,
          resolver.addressResolver,
          stateTrie,
          await makeTransactionData(
            FraudTester,
            fraudTester,
            wallet,
            'setStorage',
            [keccak256('0xabc'), keccak256('0xdef')]
          )
        )

        await stateTransitioner.proveContractInclusion(
          fraudTester.address,
          fraudTester.address,
          0,
          test.stateTrieWitness
        )

        await stateTransitioner.applyTransaction(transactionData, {
          gasLimit: SUFFICIENT_APPLY_TRANSACTION_GAS,
        })
        expect(await stateManager.updatedStorageSlotCounter()).to.equal(
          DEFAULT_TX_NUM_STORAGE_UPDATES + 1
        )

        await proveAllStorageUpdates(stateTransitioner, stateManager, stateTrie)
        expect(await stateManager.updatedStorageSlotCounter()).to.equal(0)

        await stateTransitioner.completeTransition()
        expect(await stateTransitioner.currentTransitionPhase()).to.equal(
          STATE_TRANSITIONER_PHASES.COMPLETE
        )
      })
    })
  })
})
