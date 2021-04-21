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

  private async _getStateBatchHeader(
    height: number
  ): Promise<
    | {
        batch: StateRootBatchHeader
        stateRoots: string[]
      }
    | undefined
  > {
    const filter = this.state.OVM_StateCommitmentChain.filters.StateBatchAppended()

    let startingBlock = this.state.lastQueriedL1Block
    while (
      startingBlock < (await this.options.l1RpcProvider.getBlockNumber())
    ) {
      this.state.lastQueriedL1Block = startingBlock
      this.logger.info('Querying events', {
        startingBlock,
        endBlock: startingBlock + this.options.getLogsInterval,
      })

      const events: ethers.Event[] = await this.state.OVM_StateCommitmentChain.queryFilter(
        filter,
        startingBlock,
        startingBlock + this.options.getLogsInterval
      )

      this.state.eventCache = this.state.eventCache.concat(events)
      startingBlock += this.options.getLogsInterval
    }

    // tslint:disable-next-line
    const event = this.state.eventCache.find((event) => {
      return (
        event.args._prevTotalElements.toNumber() <= height &&
        event.args._prevTotalElements.toNumber() +
          event.args._batchSize.toNumber() >
          height
      )
    })

    if (event) {
      const transaction = await this.options.l1RpcProvider.getTransaction(
        event.transactionHash
      )
      const [
        stateRoots,
      ] = this.state.OVM_StateCommitmentChain.interface.decodeFunctionData(
        'appendStateBatch',
        transaction.data
      )

      return {
        batch: {
          batchIndex: event.args._batchIndex,
          batchRoot: event.args._batchRoot,
          batchSize: event.args._batchSize,
          prevTotalElements: event.args._prevTotalElements,
          extraData: event.args._extraData,
        },
        stateRoots,
      }
    }

    return
  }

  private async _isTransactionFinalized(height: number): Promise<boolean> {
    this.logger.info('Checking if tx is finalized', { height })
    const header = await this._getStateBatchHeader(height)

    if (header === undefined) {
      this.logger.info('No state batch header found.')
      return false
    } else {
      this.logger.info('Got state batch header', { header })
    }

    return !(await this.state.OVM_StateCommitmentChain.insideFraudProofWindow(
      header.batch
    ))
  }

  private async _getSentMessages(
    startHeight: number,
    endHeight: number
  ): Promise<SentMessage[]> {
    const filter = this.state.OVM_L2CrossDomainMessenger.filters.SentMessage()
    const events = await this.state.OVM_L2CrossDomainMessenger.queryFilter(
      filter,
      startHeight + this.options.l2BlockOffset,
      endHeight + this.options.l2BlockOffset - 1
    )

    return events.map((event) => {
      const message = event.args.message
      const decoded = this.state.OVM_L2CrossDomainMessenger.interface.decodeFunctionData(
        'relayMessage',
        message
      )

      return {
        target: decoded._target,
        sender: decoded._sender,
        message: decoded._message,
        messageNonce: decoded._messageNonce,
        encodedMessage: message,
        encodedMessageHash: ethers.utils.keccak256(message),
        parentTransactionIndex: event.blockNumber - this.options.l2BlockOffset,
        parentTransactionHash: event.transactionHash,
      }
    })
  }

  private async _wasMessageRelayed(message: SentMessage): Promise<boolean> {
    return this.state.OVM_L1CrossDomainMessenger.successfulMessages(
      message.encodedMessageHash
    )
  }

  private async _getMessageProof(
    message: SentMessage
  ): Promise<SentMessageProof> {
    const messageSlot = ethers.utils.keccak256(
      ethers.utils.keccak256(
        message.encodedMessage +
          this.state.OVM_L2CrossDomainMessenger.address.slice(2)
      ) + '00'.repeat(32)
    )

    // TODO: Complain if the proof doesn't exist.
    const proof = await this.options.l2RpcProvider.send('eth_getProof', [
      this.state.OVM_L2ToL1MessagePasser.address,
      [messageSlot],
      '0x' +
        BigNumber.from(
          message.parentTransactionIndex + this.options.l2BlockOffset
        )
          .toHexString()
          .slice(2)
          .replace(/^0+/, ''),
    ])

    // TODO: Complain if the batch doesn't exist.
    const header = await this._getStateBatchHeader(
      message.parentTransactionIndex
    )

    const elements = []
    for (
      let i = 0;
      i < Math.pow(2, Math.ceil(Math.log2(header.stateRoots.length)));
      i++
    ) {
      if (i < header.stateRoots.length) {
        elements.push(header.stateRoots[i])
      } else {
        elements.push(ethers.utils.keccak256('0x' + '00'.repeat(32)))
      }
    }

    const hash = (el: Buffer | string): Buffer => {
      return Buffer.from(ethers.utils.keccak256(el).slice(2), 'hex')
    }

    const leaves = elements.map((element) => {
      return fromHexString(element)
    })

    const tree = new MerkleTree(leaves, hash)
    const index =
      message.parentTransactionIndex - header.batch.prevTotalElements.toNumber()
    const treeProof = tree.getProof(leaves[index], index).map((element) => {
      return element.data
    })

    return {
      stateRoot: header.stateRoots[index],
      stateRootBatchHeader: header.batch,
      stateRootProof: {
        index,
        siblings: treeProof,
      },
      stateTrieWitness: rlp.encode(proof.accountProof),
      storageTrieWitness: rlp.encode(proof.storageProof[0].proof),
    }
  }

  private async _relayMessageToL1(
    message: SentMessage,
    proof: SentMessageProof
  ): Promise<void> {
    if (this.options.spreadsheetMode) {
      try {
        await this.options.spreadsheet.addRow({
          target: message.target,
          sender: message.sender,
          message: message.message,
          messageNonce: message.messageNonce.toString(),
          encodedMessage: message.encodedMessage,
          encodedMessageHash: message.encodedMessageHash,
          parentTransactionIndex: message.parentTransactionIndex,
          parentTransactionHash: message.parentTransactionIndex,
          stateRoot: proof.stateRoot,
          batchIndex: proof.stateRootBatchHeader.batchIndex.toString(),
          batchRoot: proof.stateRootBatchHeader.batchRoot,
          batchSize: proof.stateRootBatchHeader.batchSize.toString(),
          prevTotalElements: proof.stateRootBatchHeader.prevTotalElements.toString(),
          extraData: proof.stateRootBatchHeader.extraData,
          index: proof.stateRootProof.index,
          siblings: proof.stateRootProof.siblings.join(','),
          stateTrieWitness: proof.stateTrieWitness.toString('hex'),
          storageTrieWitness: proof.storageTrieWitness.toString('hex'),
        })
        this.logger.info('Submitted relay message to spreadsheet')
      } catch (e) {
        this.logger.error('Cannot submit message to spreadsheet')
        this.logger.error(e.message)
      }
    } else {
      try {
        this.logger.info(
          'Dry-run, checking to make sure proof would succeed...'
        )

        await this.state.OVM_L1CrossDomainMessenger.connect(
          this.options.l1Wallet
        ).callStatic.relayMessage(
          message.target,
          message.sender,
          message.message,
          message.messageNonce,
          proof,
          {
            gasLimit: this.options.relayGasLimit,
          }
        )

        this.logger.info(
          'Proof should succeed. Submitting for real this time...'
        )
      } catch (err) {
        this.logger.error('Proof would fail, skipping', { err })
        return
      }

      const result = await this.state.OVM_L1CrossDomainMessenger.connect(
        this.options.l1Wallet
      ).relayMessage(
        message.target,
        message.sender,
        message.message,
        message.messageNonce,
        proof,
        {
          gasLimit: this.options.relayGasLimit,
        }
      )

      try {
        const receipt = await result.wait()

        this.logger.info('Relay message transaction sent', {
          transactionHash: receipt.transactionHash,
        })
      } catch (err) {
        this.logger.error('Real relay attempt failed, skipping.', { err })
        return
      }
      this.logger.info('Message successfully relayed to Layer 1!')
    }
  }
}
