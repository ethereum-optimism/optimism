/* Imports: External */
import { Contract, Signer, ethers, Wallet, BigNumber, providers } from 'ethers'
import * as rlp from 'rlp'
import { MerkleTree } from 'merkletreejs'
import { BaseTrie } from 'merkle-patricia-tree'

/* Imports: Internal */
import {
  sleep,
  ZERO_ADDRESS,
  loadContract,
  loadContractFromManager,
  L1ProviderWrapper,
  L2ProviderWrapper,
  toHexString,
  fromHexString,
  toStrippedHexString,
  encodeAccountState,
  hashOvmTransaction,
  toBytes32,
  makeTrieFromProofs,
  shuffle,
} from '@eth-optimism/core-utils'

import { loadContract, loadContractFromManager } from '@eth-optimism/contracts'

import {
  StateDiffProof,
  StateTransitionPhase,
  FraudProofData,
  OvmTransaction,
  StateRootBatchProof,
  TransactionBatchProof,
  AccountStateProof,
  StorageStateProof,
} from '../types'

interface FraudProverOptions {
  // Providers for interacting with L1 and L2.
  l1RpcProvider: providers.JsonRpcProvider
  l2RpcProvider: providers.JsonRpcProvider

  // Address of the AddressManager contract, used to resolve the various addresses we'll need
  // within this service.
  addressManagerAddress: string

  // Wallet instance, used to sign the L1 transactions.
  l1Wallet: Wallet // l1Wallet: Signer

  // Max gas.
  deployGasLimit: number
  runGasLimit: number
  
  // Height of the L2 transaction to start searching for L2->L1 messages.
  fromL2TransactionIndex?: number

  // Interval in seconds to wait between loops.
  pollingInterval?: number

  // Number of blocks that L2 is "ahead" of transaction indices. Can happen if blocks are created
  // on L2 after the genesis but before the first state commitment is published.
  l2BlockOffset?: number

  // L1 block to start querying events from. Recommended to set to the StateCommitmentChain deploy height
  l1StartOffset?: number

  // When L1 blocks are considered final
  l1BlockFinality: number
  
  // Number of blocks within each getLogs query - max is 2000
  //getLogsInterval?: number
}

const optionSettings = {
  pollingInterval: { default: 5000 },
  deployGasLimit: { default: 4_000_000 },
  runGasLimit: { default: 9_500_000 },
  fromL2TransactionIndex: { default: 0 },
  l2BlockOffset: { default: 1 },
  l1StartOffset: { default: 0 },
  l1BlockFinality: { default: 0 },
  //getLogsInterval: { default: 2000 },
}

export class FraudProverService extends BaseService<FraudProverService> {
  constructor(options: FraudProverOptions) {
    super('Fraud_Prover', options, optionSettings)
  }

  private state: {
    nextUnverifiedStateRoot: number
    lastFinalizedTxHeight: number
    nextUnfinalizedTxHeight: number
    lastQueriedL1Block: number
    eventCache: ethers.Event[]
    Lib_AddressManager: Contract
    OVM_StateCommitmentChain: Contract
    OVM_L1CrossDomainMessenger: Contract
    OVM_L2CrossDomainMessenger: Contract
    OVM_L2ToL1MessagePasser: Contract
    //l1Provider: L1ProviderWrapper
    //l2Provider: L2ProviderWrapper
    OVM_CanonicalTransactionChain: Contract
    OVM_FraudVerifier: Contract
    OVM_ExecutionManager: Contract
  }

  protected async _init(): Promise<void> {
    this.logger.info('Initializing fraud prover', { options: this.options })
    // Need to improve this, sorry.
    this.state = {} as any

    const address = await this.options.l1Wallet.getAddress()
    this.logger.info('Using L1 EOA', { address })

    this.logger.info('Trying to connect to the L1 network...')
    for (let i = 0; i < 10; i++) {
      try {
        await this.options.l1RpcProvider.detectNetwork()
        this.logger.info('Successfully connected to the L1 network.')
        break
      } catch (err) {
        if (i < 9) {
          this.logger.info('Unable to connect to L1 network', {
            retryAttemptsRemaining: 10 - i,
          })
          await sleep(1000)
        } else {
          throw new Error(
            `Unable to connect to the L1 network, check that your L1 endpoint is correct.`
          )
        }
      }
    }

    this.logger.info('Trying to connect to the L2 network...')
    for (let i = 0; i < 10; i++) {
      try {
        await this.options.l2RpcProvider.detectNetwork()
        this.logger.info('Successfully connected to the L2 network.')
        break
      } catch (err) {
        if (i < 9) {
          this.logger.info('Unable to connect to L1 network', {
            retryAttemptsRemaining: 10 - i,
          })
          await sleep(1000)
        } else {
          throw new Error(
            `Unable to connect to the L2 network, check that your L2 endpoint is correct.`
          )
        }
      }
    }

    this.logger.info('Connecting to Lib_AddressManager...')
    this.state.Lib_AddressManager = loadContract(
      'Lib_AddressManager',
      this.options.addressManagerAddress,
      this.options.l1RpcProvider
    )
    this.logger.info('Connected to Lib_AddressManager', {
      address: this.state.Lib_AddressManager.address,
    })

    this.logger.info('Connecting to OVM_StateCommitmentChain...')
    this.state.OVM_StateCommitmentChain = await loadContractFromManager({
      name: 'OVM_StateCommitmentChain',
      Lib_AddressManager: this.state.Lib_AddressManager,
      provider: this.options.l1RpcProvider,
    })
    this.logger.info('Connected to OVM_StateCommitmentChain', {
      address: this.state.OVM_StateCommitmentChain.address,
    })

    this.logger.info('Connecting to OVM_CanonicalTransactionChain...')
    this.state.OVM_CanonicalTransactionChain = await loadContractFromManager(
      'OVM_CanonicalTransactionChain',
      this.state.Lib_AddressManager,
      this.options.l1RpcProvider
    )
    this.logger.info('Connected to OVM_CanonicalTransactionChain', {
      address: this.state.OVM_CanonicalTransactionChain.address,
    })

    this.logger.info('Connecting to OVM_FraudVerifier...')
    this.state.OVM_FraudVerifier = await loadContractFromManager(
      'OVM_FraudVerifier',
      this.state.Lib_AddressManager,
      this.options.l1RpcProvider
    )
    this.logger.info('Connected to OVM_FraudVerifier', {
      address: this.state.OVM_FraudVerifier.address,
    })

    this.logger.info('Connecting to OVM_ExecutionManager...')
    this.state.OVM_ExecutionManager = await loadContractFromManager(
      'OVM_ExecutionManager',
      this.state.Lib_AddressManager,
      this.options.l1RpcProvider
    )
    this.logger.info('Connected to OVM_ExecutionManager', {
      address: this.state.OVM_ExecutionManager.address,
    })

    // this.logger.info('Connecting to OVM_L1CrossDomainMessenger...')
    // this.state.OVM_L1CrossDomainMessenger = await loadContractFromManager({
    //   name: 'OVM_L1CrossDomainMessenger',
    //   proxy: 'Proxy__OVM_L1CrossDomainMessenger',
    //   Lib_AddressManager: this.state.Lib_AddressManager,
    //   provider: this.options.l1RpcProvider,
    // })
    // this.logger.info('Connected to OVM_L1CrossDomainMessenger', {
    //   address: this.state.OVM_L1CrossDomainMessenger.address,
    // })

    // this.logger.info('Connecting to OVM_L2CrossDomainMessenger...')
    // this.state.OVM_L2CrossDomainMessenger = await loadContractFromManager({
    //   name: 'OVM_L2CrossDomainMessenger',
    //   Lib_AddressManager: this.state.Lib_AddressManager,
    //   provider: this.options.l2RpcProvider,
    // })
    // this.logger.info('Connected to OVM_L2CrossDomainMessenger', {
    //   address: this.state.OVM_L2CrossDomainMessenger.address,
    // })

    // this.logger.info('Connecting to OVM_L2ToL1MessagePasser...')
    // this.state.OVM_L2ToL1MessagePasser = loadContract(
    //   'OVM_L2ToL1MessagePasser',
    //   '0x4200000000000000000000000000000000000000',
    //   this.options.l2RpcProvider
    // )
    // this.logger.info('Connected to OVM_L2ToL1MessagePasser', {
    //   address: this.state.OVM_L2ToL1MessagePasser.address,
    // })

    this.logger.info('Connected to all contracts.')

    this.state.l1Provider = new L1ProviderWrapper(
      this.options.l1RpcProvider,
      this.state.OVM_StateCommitmentChain,
      this.state.OVM_CanonicalTransactionChain,
      this.state.OVM_ExecutionManager,
      this.options.l1StartOffset,
      this.options.l1BlockFinality
    )

    this.logger.info(
      'Caching events for relevant contracts, this might take a while...'
    )
    this.logger.info('Caching events for OVM_StateCommitmentChain...')
    await this.state.l1Provider.findAllEvents(
      this.state.OVM_StateCommitmentChain,
      this.state.OVM_StateCommitmentChain.filters.StateBatchAppended()
    )

    this.logger.info('Caching events for OVM_CanonicalTransactionChain...')
    await this.state.l1Provider.findAllEvents(
      this.state.OVM_CanonicalTransactionChain,
      this.state.OVM_CanonicalTransactionChain.filters.TransactionBatchAppended()
    )
    await this.state.l1Provider.findAllEvents(
      this.state.OVM_CanonicalTransactionChain,
      this.state.OVM_CanonicalTransactionChain.filters.SequencerBatchAppended()
    )

    // if (this.options.spreadsheetMode) {
    //   this.logger.info('Running in spreadsheet mode')
    // }

    this.state.lastQueriedL1Block = this.options.l1StartOffset
    this.state.eventCache = []

    this.state.lastFinalizedTxHeight = this.options.fromL2TransactionIndex || 0
    this.state.nextUnfinalizedTxHeight =
      this.options.fromL2TransactionIndex || 0
  }

  protected async _start(): Promise<void> {
    while (this.running) {
      await sleep(this.options.pollingInterval)

      try {
        this.logger.info('Looking for mismatched state roots...')
        const fraudulentStateRootIndex = await this._findNextFraudulentStateRoot()

        if (fraudulentStateRootIndex === undefined) {
          this.logger.info('Did not find any mismatched state roots', {
            nextAttemptInS: this.options.pollingInterval / 1000,
          })
          continue
        }

        this.logger.info('Found a mismatched state root', {
          index: fraudulentStateRootIndex,
        })

        this.logger.info('Pulling fraud proof data...')
        const proof = await this._getFraudProofData(fraudulentStateRootIndex)

        this.logger.info('Initializing the fraud verification process...')
        try {
          await this._initializeFraudVerification(
            proof.preStateRootProof,
            proof.transactionProof
          )
        } catch (err) {
          if (err.toString().includes('Reverted 0x')) {
            this.logger.info(
              'Fraud proof was initialized by someone else, moving on...'
            )
          } else {
            throw err
          }
        }

        this.logger.info('Loading fraud proof contracts...')
        const {
          OVM_StateTransitioner,
          OVM_StateManager,
        } = await this._getFraudProofContracts(
          await this.state.l1Provider.getStateRoot(
            fraudulentStateRootIndex - 1
          ),
          proof.transactionProof.transaction
        )

        // PRE_EXECUTION phase.
        if (
          (await OVM_StateTransitioner.phase()) ===
          StateTransitionPhase.PRE_EXECUTION
        ) {
          try {
            this.logger.info('Fraud proof is now in the PRE_EXECUTION phase.')

            this.logger.info('Proving account states...')
            await this._proveAccountStates(
              OVM_StateTransitioner,
              OVM_StateManager,
              proof.stateDiffProof.accountStateProofs,
              fraudulentStateRootIndex
            )

            this.logger.info('Proving storage slot states...')
            await this._proveContractStorageStates(
              OVM_StateTransitioner,
              OVM_StateManager,
              proof.stateDiffProof.accountStateProofs
            )

            this.logger.info('Executing transaction...')
            try {
              await (
                await OVM_StateTransitioner.applyTransaction(
                  proof.transactionProof.transaction,
                  {
                    gasLimit: this.options.runGasLimit,
                  }
                )
              ).wait()
            } catch (err) {
              await OVM_StateTransitioner.callStatic.applyTransaction(
                proof.transactionProof.transaction,
                {
                  gasLimit: this.options.runGasLimit,
                }
              )
            }

            this.logger.info('Transaction successfully executed.')
          } catch (err) {
            if (
              err
                .toString()
                .includes(
                  'Function must be called during the correct phase.'
                ) ||
              err
                .toString()
                .includes(
                  '46756e6374696f6e206d7573742062652063616c6c656420647572696e672074686520636f72726563742070686173652e'
                )
            ) {
              this.logger.info(
                'Phase was completed by someone else, moving on.'
              )
            } else {
              throw err
            }
          }
        }

        // POST_EXECUTION phase.
        if (
          (await OVM_StateTransitioner.phase()) ===
          StateTransitionPhase.POST_EXECUTION
        ) {
          try {
            this.logger.info('Fraud proof is now in the POST_EXECUTION phase.')

            this.logger.info('Committing storage slot state updates...')
            await this._updateContractStorageStates(
              OVM_StateTransitioner,
              OVM_StateManager,
              proof.stateDiffProof.accountStateProofs,
              proof.storageTries
            )

            this.logger.info('Committing account state updates...')
            await this._updateAccountStates(
              OVM_StateTransitioner,
              OVM_StateManager,
              proof.stateDiffProof.accountStateProofs,
              proof.stateTrie
            )

            this.logger.info('Completing the state transition...')
            try {
              await (await OVM_StateTransitioner.completeTransition()).wait()
            } catch (err) {
              try {
                await OVM_StateTransitioner.callStatic.completeTransition()
              } catch (err) {
                if (err.toString().includes('Reverted 0x')) {
                  this.logger.info(
                    'State transition was completed by someone else, moving on.'
                  )
                } else {
                  throw err
                }
              }
            }

            this.logger.info('State transition completed.')
          } catch (err) {
            if (
              err
                .toString()
                .includes(
                  'Function must be called during the correct phase.'
                ) ||
              err
                .toString()
                .includes(
                  '46756e6374696f6e206d7573742062652063616c6c656420647572696e672074686520636f72726563742070686173652e'
                )
            ) {
              this.logger.info(
                'Phase was completed by someone else, moving on.'
              )
            } else {
              throw err
            }
          }
        }

        // COMPLETE phase.
        if (
          (await OVM_StateTransitioner.phase()) ===
          StateTransitionPhase.COMPLETE
        ) {
          this.logger.info('Fraud proof is now in the COMPLETE phase.')

          this.logger.info('Attempting to finalize the fraud proof...')
          try {
            await this._finalizeFraudVerification(
              proof.preStateRootProof,
              proof.postStateRootProof,
              proof.transactionProof.transaction
            )

            this.logger.info('Fraud proof finalized! Congrats.')
          } catch (err) {
            if (
              err.toString().includes('Invalid batch header.') ||
              err.toString().includes('Index out of bounds.') ||
              err.toString().includes('Reverted 0x')
            ) {
              this.logger.info('Fraud proof was finalized by someone else.')
            } else {
              throw err
            }
          }
        }

        this.state.nextUnverifiedStateRoot = proof.preStateRootProof.stateRootBatchHeader.prevTotalElements.toNumber()
      } catch (err) {
        this.logger.error('Caught an unhandled error', {
          err,
        })
      }
    }
  }

    /**
   * Finds the index of the next fraudulent state root.
   * @return Index of the next fraudulent state root, if any.
   */
  private async _findNextFraudulentStateRoot(): Promise<number | undefined> {
    let nextBatchHeader = await this.state.l1Provider.getStateRootBatchHeader(
      this.state.nextUnverifiedStateRoot
    )

    while (nextBatchHeader !== undefined) {
      const nextBatchStateRoots = await this.state.l1Provider.getBatchStateRoots(
        this.state.nextUnverifiedStateRoot
      )

      for (let i = 0; i < nextBatchHeader.batchSize.toNumber(); i++) {
        const index = i + nextBatchHeader.prevTotalElements.toNumber()
        this.logger.info('Checking state root for mismatch', { index })

        const l1StateRoot = nextBatchStateRoots[i]
        const l2StateRoot = await this.state.l2Provider.getStateRoot(
          index + this.options.l2BlockOffset
        )

        if (l1StateRoot !== l2StateRoot) {
          this.logger.info('State roots do not match')
          this.logger.info('L1 State Root', { l1StateRoot })
          this.logger.info('L2 State Root', { l2StateRoot })
          return index
        } else {
          this.logger.info('State root was not mismatched ✓')
        }
      }

      this.state.nextUnverifiedStateRoot =
        nextBatchHeader.prevTotalElements.toNumber() +
        nextBatchHeader.batchSize.toNumber()
      nextBatchHeader = await this.state.l1Provider.getStateRootBatchHeader(
        this.state.nextUnverifiedStateRoot
      )
    }
  }

  /**
   * Generates all transaction proof data for a given transaction index.
   * @param transactionIndex Transaction index to get proof data for.
   * @return Transaction proof data.
   */
  private async _getFraudProofData(
    transactionIndex: number
  ): Promise<FraudProofData> {
    this.logger.info('Getting pre-state root inclusion proof...')
    const preStateRootProof = await this.state.l1Provider.getStateRootBatchProof(
      transactionIndex - 1
    )

    this.logger.info('Getting post-state root inclusion proof...')
    const postStateRootProof = await this.state.l1Provider.getStateRootBatchProof(
      transactionIndex
    )

    this.logger.info('Getting transaction inclusion proof...')
    const transactionProof = await this.state.l1Provider.getTransactionBatchProof(
      transactionIndex
    )

    this.logger.info('Getting state diff proof...')
    const stateDiffProof: StateDiffProof = await this.state.l2Provider.getStateDiffProof(
      transactionIndex + this.options.l2BlockOffset - 1
    )

    const stateTrie = await this._makeStateTrie(stateDiffProof)
    const storageTries = await this._makeAccountTries(stateDiffProof)

    return {
      stateDiffProof,
      transactionProof,
      preStateRootProof,
      postStateRootProof,
      stateTrie,
      storageTries,
    }
  }

  /**
   * Pulls the fraud proof contracts.
   * @param preStateRoot Pre-state root to pull contracts for.
   * @param transaction Transaction to pull contracts for.
   * @return Fraud proof contracts.
   */
  private async _getFraudProofContracts(
    preStateRoot: string,
    transaction: OvmTransaction
  ): Promise<{
    OVM_StateTransitioner: Contract
    OVM_StateManager: Contract
  }> {
    this.logger.info('Loading the state transitioner...')

    const stateTransitionerAddress = await this._getStateTransitioner(
      preStateRoot,
      transaction
    )

    const OVM_StateTransitioner = loadContract(
      'OVM_StateTransitioner',
      stateTransitionerAddress,
      this.options.l1RpcProvider
    ).connect(this.options.l1Wallet)

    this.logger.info('State transitioner', { stateTransitionerAddress })

    this.logger.info('Loading the corresponding state manager...')

    const stateManagerAddress = await OVM_StateTransitioner.ovmStateManager()
    const OVM_StateManager = loadContract(
      'OVM_StateManager',
      stateManagerAddress,
      this.options.l1RpcProvider
    ).connect(this.options.l1Wallet)

    this.logger.info('State manager', { stateManagerAddress })

    return {
      OVM_StateTransitioner,
      OVM_StateManager,
    }
  }

  /**
   * Generates a view of the state trie from a state diff proof.
   * @param proof State diff proof to generate a trie from.
   * @return View of the state trie.
   */
  private async _makeStateTrie(proof: StateDiffProof): Promise<BaseTrie> {
    return makeTrieFromProofs(
      proof.accountStateProofs.map((accountStateProof) => {
        return accountStateProof.accountProof
      })
    )
  }

  /**
   * Generates a view of a set of account tries from a state diff proof.
   * @param proof State diff proof to generate tries from.
   * @return View of a set of all account tries.
   */
  private async _makeAccountTries(
    proof: StateDiffProof
  ): Promise<{
    [address: string]: BaseTrie
  }> {
    const accountTries: { [address: string]: BaseTrie } = {}

    for (const accountStateProof of proof.accountStateProofs) {
      accountTries[accountStateProof.address] = await makeTrieFromProofs(
        accountStateProof.storageProof.map((storageProof) => {
          return storageProof.proof
        })
      )
    }

    return accountTries
  }

  /**
   * Retrieves the state transitioner correspondng to a given pre-state root and transaction.
   * @param preStateRoot Pre-state root to retreive a state transitioner for.
   * @param transaction Transaction to retreive a state transitioner for.
   * @return Address of the corresponding state transitioner.
   */
  private async _getStateTransitioner(
    preStateRoot: string,
    transaction: OvmTransaction
  ): Promise<string> {
    return this.state.OVM_FraudVerifier.getStateTransitioner(
      preStateRoot,
      hashOvmTransaction(transaction)
    )
  }

  /**
   * Simple mechanism for deploying an exact bytecode to a given address. Resulting contract will
   * have code exactly matching the given `code` variable, and none of the code will be executed
   * during creation.
   * @param code Code to store at a given address.
   * @return Address of the newly created contract.
   */
  private async _deployContractCode(code: string): Promise<string> {
    // "Magic" prefix to be prepended to the contract code. Contains a series of opcodes that will
    // copy the given code into memory and return it, thereby storing at the contract address.
    const prefix = '0x600D380380600D6000396000f3'
    const deployCode = prefix + toHexString(code).slice(2)

    const response = await this.options.l1Wallet.sendTransaction({
      to: null,
      data: deployCode,
      gasLimit: this.options.deployGasLimit,
    })

    const result = await response.wait()
    return result.contractAddress
  }

  /**
   * Proves the state of all given accounts.
   * @param OVM_StateTransitioner Ethers contract instance pointed at the state transitioner.
   * @param OVM_StateManager Ethers contract instance pointed at the state manager.
   * @param accountStateProofs All account state proofs.
   * @param fraudulentStateRootIndex Index of the fraudulent state root.
   */
  private async _proveAccountStates(
    OVM_StateTransitioner: Contract,
    OVM_StateManager: Contract,
    accountStateProofs: AccountStateProof[],
    fraudulentStateRootIndex: number
  ): Promise<void> {
    for (const accountStateProof of shuffle(accountStateProofs)) {
      this.logger.info('Attempting to prove account state', {
        address: accountStateProof.address,
      })

      if (await OVM_StateManager.hasAccount(accountStateProof.address)) {
        this.logger.info(
          'Someone else already proved this account, skipping...'
        )
        continue
      }

      const accountCode = await this.options.l2RpcProvider.getCode(
        accountStateProof.address,
        fraudulentStateRootIndex + this.options.l2BlockOffset
      )

      let ethContractAddress = '0x0000c0De0000C0DE0000c0de0000C0DE0000c0De'
      if (accountCode !== '0x') {
        this.logger.info('Need to deploy a copy of the account first...')
        ethContractAddress = await this._deployContractCode(accountCode)
        this.logger.info('Deployed a copy of the account, attempting proof...')
      }

      try {
        await (
          await OVM_StateTransitioner.proveContractState(
            accountStateProof.address,
            ethContractAddress,
            rlp.encode(accountStateProof.accountProof)
          )
        ).wait()

        this.logger.info('Account state proven.')
      } catch (err) {
        try {
          await OVM_StateTransitioner.callStatic.proveContractState(
            accountStateProof.address,
            ethContractAddress,
            rlp.encode(accountStateProof.accountProof)
          )
        } catch (err) {
          if (
            err.toString().includes('Account state has already been proven') ||
            err.toString().includes('Reverted 0x')
          ) {
            this.logger.info(
              'Someone else has already proven this account, skipping.'
            )
          } else {
            throw err
          }
        }
      }
    }
  }

  /**
   * Proves all contract storage slot states.
   * @param OVM_StateTransitioner Ethers contract instance pointed at the state transitioner.
   * @param OVM_StateManager Ethers contract instance pointed at the state manager.
   * @param accountStateProofs All account state proofs.
   */
  private async _proveContractStorageStates(
    OVM_StateTransitioner: Contract,
    OVM_StateManager: Contract,
    accountStateProofs: AccountStateProof[]
  ): Promise<void> {
    for (const accountStateProof of shuffle(accountStateProofs)) {
      for (const slot of shuffle(accountStateProof.storageProof)) {
        this.logger.info('Attempting to prove slot.', {
          address: accountStateProof.address,
          key: slot.key,
          value: slot.value,
        })
        if (
          await OVM_StateManager.hasContractStorage(
            accountStateProof.address,
            toBytes32(slot.key)
          )
        ) {
          this.logger.info(
            'Someone else has already proven this slot, skipping...'
          )
          continue
        }

        try {
          await (
            await OVM_StateTransitioner.proveStorageSlot(
              accountStateProof.address,
              toBytes32(slot.key),
              rlp.encode(slot.proof)
            )
          ).wait()

          this.logger.info('Slot value proven.')
        } catch (err) {
          try {
            await OVM_StateTransitioner.callStatic.proveStorageSlot(
              accountStateProof.address,
              toBytes32(slot.key),
              rlp.encode(slot.proof)
            )
          } catch (err) {
            if (
              err
                .toString()
                .includes('Storage slot has already been proven.') ||
              err.toString().includes('Reverted 0x')
            ) {
              this.logger.info(
                'Someone else has already proven this slot, skipping.'
              )
            } else {
              throw err
            }
          }
        }
      }
    }
  }

  /**
   * Commits all account state changes.
   * @param OVM_StateTransitioner Ethers contract instance pointed at the state transitioner.
   * @param OVM_StateManager Ethers contract instance pointed at the state manager.
   * @param accountStateProofs All account state proofs.
   * @param stateTrie State trie view generated from proof data.
   */
  private async _updateAccountStates(
    OVM_StateTransitioner: Contract,
    OVM_StateManager: Contract,
    accountStateProofs: AccountStateProof[],
    stateTrie: BaseTrie
  ): Promise<void> {
    while ((await OVM_StateManager.getTotalUncommittedAccounts()) > 0) {
      const accountCommittedEvents = await this.state.l1Provider.findAllEvents(
        OVM_StateTransitioner,
        OVM_StateTransitioner.filters.AccountCommitted()
      )

      // Use events to figure out which accounts we've already committed.
      const committedAccounts = accountStateProofs.filter((account) => {
        return accountCommittedEvents.some((event) => {
          return (
            event.args._address.toLowerCase() === account.address.toLowerCase()
          )
        })
      })

      // Update our trie with the values of any accounts that have already been committed. Order
      // here doesn't matter because the trie will still end up with the same root. We can also
      // repeatedly update a key with the same value since it won't have an impact on the trie.
      for (const account of committedAccounts) {
        const accountState = await OVM_StateManager.getAccount(account.address)

        await stateTrie.put(
          fromHexString(ethers.utils.keccak256(account.address)),
          encodeAccountState({
            ...accountState,
            ...{
              nonce: accountState.nonce.toNumber(),
            },
          })
        )
      }

      // Find an uncommitted account to attempt to commit.
      let nextUncommittedAccount: AccountStateProof
      for (const account of shuffle(accountStateProofs)) {
        if (
          !(await OVM_StateManager.wasAccountCommitted(account.address)) &&
          (await OVM_StateManager.wasAccountChanged(account.address))
        ) {
          nextUncommittedAccount = account
          break
        }
      }

      if (nextUncommittedAccount === undefined) {
        if ((await OVM_StateManager.getTotalUncommittedAccounts()) > 0) {
          throw new Error(
            `We still have accounts to commit, but we don't have any more proof data. Something went very wrong.`
          )
        } else {
          return
        }
      }

      // Generate an inclusion proof for the account, will be used to update the value on-chain.
      const accountInclusionProof = toHexString(
        rlp.encode(
          await BaseTrie.createProof(
            stateTrie,
            fromHexString(
              ethers.utils.keccak256(nextUncommittedAccount.address)
            )
          )
        )
      )

      const updatedAccountState = await OVM_StateManager.getAccount(
        nextUncommittedAccount.address
      )

      this.logger.info('Attempting to commit account.', {
        address: nextUncommittedAccount.address,
        balance: updatedAccountState.balance,
        nonce: updatedAccountState.nonce,
        storageRoot: updatedAccountState.storageRoot,
        codeHash: updatedAccountState.codeHash,
      })

      try {
        await (
          await OVM_StateTransitioner.commitContractState(
            nextUncommittedAccount.address,
            accountInclusionProof,
            {
              gasLimit: this.options.deployGasLimit,
            }
          )
        ).wait()

        this.logger.info('Account committed.')
      } catch (err) {
        try {
          await OVM_StateTransitioner.callStatic.commitContractState(
            nextUncommittedAccount.address,
            accountInclusionProof,
            {
              gasLimit: this.options.deployGasLimit,
            }
          )
        } catch (err) {
          if (
            err.toString().includes('invalid opcode') ||
            err.toString().includes('Invalid root hash') ||
            err
              .toString()
              .includes(
                `Account state wasn't changed or has already been committed.`
              ) ||
            err.toString().includes('Reverted 0x')
          ) {
            this.logger.info(
              'Could not commit account because another commitment invalidated our proof, skipping for now...'
            )
          } else {
            throw err
          }
        }
      }
    }
  }

  /**
   * Commits all contract storage slot changes.
   * @param OVM_StateTransitioner Ethers contract instance pointed at the state transitioner.
   * @param OVM_StateManager Ethers contract instance pointed at the state manager.
   * @param accountStateProofs All account state proofs.
   * @param stateTrie State trie view generated from proof data.
   * @param storageTries Storage trie views generated from proof data.
   */
  private async _updateContractStorageStates(
    OVM_StateTransitioner: Contract,
    OVM_StateManager: Contract,
    accountStateProofs: AccountStateProof[],
    storageTries: {
      [address: string]: BaseTrie
    }
  ) {
    while ((await OVM_StateManager.getTotalUncommittedContractStorage()) > 0) {
      const storageCommittedEvents = await this.state.l1Provider.findAllEvents(
        OVM_StateTransitioner,
        OVM_StateTransitioner.filters.ContractStorageCommitted()
      )

      for (const accountStateProof of accountStateProofs) {
        const committedStorageSlots = accountStateProof.storageProof.filter(
          (storageProof) => {
            return storageCommittedEvents.some((event) => {
              return (
                event.args._address.toLowerCase() ===
                  accountStateProof.address.toLowerCase() &&
                event.args._key.toLowerCase() === storageProof.key.toLowerCase()
              )
            })
          }
        )

        for (const storageProof of committedStorageSlots) {
          const updatedSlotValue = await OVM_StateManager.getContractStorage(
            accountStateProof.address,
            storageProof.key
          )

          await storageTries[accountStateProof.address].put(
            fromHexString(ethers.utils.keccak256(storageProof.key)),
            fromHexString(rlp.encode(toStrippedHexString(updatedSlotValue)))
          )
        }
      }

      for (const accountStateProof of accountStateProofs) {
        let nextUncommittedStorageProof: StorageStateProof
        for (const storageProof of accountStateProof.storageProof) {
          if (
            !(await OVM_StateManager.wasContractStorageCommitted(
              accountStateProof.address,
              storageProof.key
            )) &&
            (await OVM_StateManager.wasContractStorageChanged(
              accountStateProof.address,
              storageProof.key
            ))
          ) {
            nextUncommittedStorageProof = storageProof
            break
          }
        }

        if (nextUncommittedStorageProof === undefined) {
          continue
        }

        const slotInclusionProof = toHexString(
          rlp.encode(
            await BaseTrie.createProof(
              storageTries[accountStateProof.address],
              fromHexString(
                ethers.utils.keccak256(nextUncommittedStorageProof.key)
              )
            )
          )
        )

        const updatedSlotValue = await OVM_StateManager.getContractStorage(
          accountStateProof.address,
          nextUncommittedStorageProof.key
        )

        this.logger.info('Attempting to commit storage slot.', {
          address: accountStateProof.address,
          key: nextUncommittedStorageProof.key,
          value: updatedSlotValue,
        })

        try {
          await (
            await OVM_StateTransitioner.commitStorageSlot(
              accountStateProof.address,
              nextUncommittedStorageProof.key,
              slotInclusionProof,
              {
                gasLimit: this.options.deployGasLimit,
              }
            )
          ).wait()

          this.logger.info('Storage slot committed.')
        } catch (err) {
          try {
            await OVM_StateTransitioner.callStatic.commitStorageSlot(
              accountStateProof.address,
              nextUncommittedStorageProof.key,
              slotInclusionProof,
              {
                gasLimit: this.options.deployGasLimit,
              }
            )
          } catch (err) {
            if (
              err.toString().includes('invalid opcode') ||
              err.toString().includes('Invalid root hash') ||
              err
                .toString()
                .includes(
                  `Storage slot value wasn't changed or has already been committed.`
                ) ||
              err.toString().includes('Reverted 0x')
            ) {
              this.logger.info(
                'Could not commit slot because another commitment invalidated our proof, skipping for now...'
              )
            } else {
              throw err
            }
          }
        }
      }
    }
  }

  /**
   * Initializes the fraud verification process.
   * @param preStateRootProof Proof data for the pre-state root.
   * @param transactionProof Proof data for the transaction being verified.
   */
  private async _initializeFraudVerification(
    preStateRootProof: StateRootBatchProof,
    transactionProof: TransactionBatchProof
  ): Promise<void> {
    const stateTransitionerAddress = await this._getStateTransitioner(
      preStateRootProof.stateRoot,
      transactionProof.transaction
    )

    if (stateTransitionerAddress !== ZERO_ADDRESS) {
      return
    }

    try {
      await (
        await this.state.OVM_FraudVerifier.connect(
          this.options.l1Wallet
        ).initializeFraudVerification(
          preStateRootProof.stateRoot,
          preStateRootProof.stateRootBatchHeader,
          preStateRootProof.stateRootProof,
          transactionProof.transaction,
          transactionProof.transactionChainElement,
          transactionProof.transactionBatchHeader,
          transactionProof.transactionProof
        )
      ).wait()
    } catch (err) {
      await this.state.OVM_FraudVerifier.connect(
        this.options.l1Wallet
      ).callStatic.initializeFraudVerification(
        preStateRootProof.stateRoot,
        preStateRootProof.stateRootBatchHeader,
        preStateRootProof.stateRootProof,
        transactionProof.transaction,
        transactionProof.transactionChainElement,
        transactionProof.transactionBatchHeader,
        transactionProof.transactionProof
      )
    }
  }

  /**
   * Finalizes the fraud verification process.
   * @param preStateRootProof Proof data for the pre-state root.
   * @param postStateRootProof Proof data for the post-state root.
   * @param transaction Transaction being verified.
   */
  private async _finalizeFraudVerification(
    preStateRootProof: StateRootBatchProof,
    postStateRootProof: StateRootBatchProof,
    transaction: OvmTransaction
  ): Promise<void> {
    try {
      await (
        await this.state.OVM_FraudVerifier.connect(
          this.options.l1Wallet
        ).finalizeFraudVerification(
          preStateRootProof.stateRoot,
          preStateRootProof.stateRootBatchHeader,
          preStateRootProof.stateRootProof,
          hashOvmTransaction(transaction),
          postStateRootProof.stateRoot,
          postStateRootProof.stateRootBatchHeader,
          postStateRootProof.stateRootProof
        )
      ).wait()
    } catch (err) {
      await this.state.OVM_FraudVerifier.connect(
        this.options.l1Wallet
      ).callStatic.finalizeFraudVerification(
        preStateRootProof.stateRoot,
        preStateRootProof.stateRootBatchHeader,
        preStateRootProof.stateRootProof,
        hashOvmTransaction(transaction),
        postStateRootProof.stateRoot,
        postStateRootProof.stateRootBatchHeader,
        postStateRootProof.stateRootProof
      )
    }
  }
}
