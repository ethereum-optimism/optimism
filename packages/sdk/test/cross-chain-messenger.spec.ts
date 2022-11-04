import { Provider } from '@ethersproject/abstract-provider'
import { expectApprox, hashCrossDomainMessage } from '@eth-optimism/core-utils'
import { predeploys } from '@eth-optimism/contracts'
import { Contract } from 'ethers'
import { ethers } from 'hardhat'

import { expect } from './setup'
import {
  MessageDirection,
  CONTRACT_ADDRESSES,
  omit,
  MessageStatus,
  CrossChainMessage,
  CrossChainMessenger,
  StandardBridgeAdapter,
  ETHBridgeAdapter,
  L1ChainID,
  L2ChainID,
} from '../src'
import { DUMMY_MESSAGE, DUMMY_EXTENDED_MESSAGE } from './helpers'

describe('CrossChainMessenger', () => {
  let l1Signer: any
  let l2Signer: any
  before(async () => {
    ;[l1Signer, l2Signer] = await ethers.getSigners()
  })

  describe('construction', () => {
    describe('when given an ethers provider for the L1 provider', () => {
      it('should use the provider as the L1 provider', () => {
        const messenger = new CrossChainMessenger({
          l1SignerOrProvider: ethers.provider,
          l2SignerOrProvider: ethers.provider,
          l1ChainId: L1ChainID.MAINNET,
          l2ChainId: L2ChainID.OPTIMISM,
        })

        expect(messenger.l1Provider).to.equal(ethers.provider)
      })
    })

    describe('when given an ethers provider for the L2 provider', () => {
      it('should use the provider as the L2 provider', () => {
        const messenger = new CrossChainMessenger({
          l1SignerOrProvider: ethers.provider,
          l2SignerOrProvider: ethers.provider,
          l1ChainId: L1ChainID.MAINNET,
          l2ChainId: L2ChainID.OPTIMISM,
        })

        expect(messenger.l2Provider).to.equal(ethers.provider)
      })
    })

    describe('when given a string as the L1 provider', () => {
      it('should create a JSON-RPC provider for the L1 provider', () => {
        const messenger = new CrossChainMessenger({
          l1SignerOrProvider: 'https://localhost:8545',
          l2SignerOrProvider: ethers.provider,
          l1ChainId: L1ChainID.MAINNET,
          l2ChainId: L2ChainID.OPTIMISM,
        })

        expect(Provider.isProvider(messenger.l1Provider)).to.be.true
      })
    })

    describe('when given a string as the L2 provider', () => {
      it('should create a JSON-RPC provider for the L2 provider', () => {
        const messenger = new CrossChainMessenger({
          l1SignerOrProvider: ethers.provider,
          l2SignerOrProvider: 'https://localhost:8545',
          l1ChainId: L1ChainID.MAINNET,
          l2ChainId: L2ChainID.OPTIMISM,
        })

        expect(Provider.isProvider(messenger.l2Provider)).to.be.true
      })
    })

    describe('when given a bad L1 chain ID', () => {
      it('should throw an error', () => {
        expect(() => {
          new CrossChainMessenger({
            l1SignerOrProvider: ethers.provider,
            l2SignerOrProvider: ethers.provider,
            l1ChainId: undefined as any,
            l2ChainId: L2ChainID.OPTIMISM,
          })
        }).to.throw('L1 chain ID is missing or invalid')
      })
    })

    describe('when given a bad L2 chain ID', () => {
      it('should throw an error', () => {
        expect(() => {
          new CrossChainMessenger({
            l1SignerOrProvider: ethers.provider,
            l2SignerOrProvider: ethers.provider,
            l1ChainId: L1ChainID.MAINNET,
            l2ChainId: undefined as any,
          })
        }).to.throw('L2 chain ID is missing or invalid')
      })
    })

    describe('when no custom contract addresses are provided', () => {
      describe('when given a known chain ID', () => {
        it('should use the contract addresses for the known chain ID', () => {
          const messenger = new CrossChainMessenger({
            l1SignerOrProvider: ethers.provider,
            l2SignerOrProvider: 'https://localhost:8545',
            l1ChainId: L1ChainID.MAINNET,
            l2ChainId: L2ChainID.OPTIMISM,
          })

          const addresses = CONTRACT_ADDRESSES[messenger.l2ChainId]
          for (const [contractName, contractAddress] of Object.entries(
            addresses.l1
          )) {
            const contract = messenger.contracts.l1[contractName]
            expect(contract.address).to.equal(contractAddress)
          }
          for (const [contractName, contractAddress] of Object.entries(
            addresses.l2
          )) {
            const contract = messenger.contracts.l2[contractName]
            expect(contract.address).to.equal(contractAddress)
          }
        })
      })

      describe('when given an unknown L2 chain ID', () => {
        it('should throw an error', () => {
          expect(() => {
            new CrossChainMessenger({
              l1SignerOrProvider: ethers.provider,
              l2SignerOrProvider: 'https://localhost:8545',
              l1ChainId: L1ChainID.MAINNET,
              l2ChainId: 1234,
            })
          }).to.throw()
        })
      })
    })

    describe('when custom contract addresses are provided', () => {
      describe('when given a known chain ID', () => {
        it('should use known addresses except where custom addresses are given', () => {
          const overrides = {
            l1: {
              L1CrossDomainMessenger: '0x' + '11'.repeat(20),
            },
            l2: {
              L2CrossDomainMessenger: '0x' + '22'.repeat(20),
            },
          }
          const messenger = new CrossChainMessenger({
            l1SignerOrProvider: ethers.provider,
            l2SignerOrProvider: 'https://localhost:8545',
            l1ChainId: L1ChainID.MAINNET,
            l2ChainId: L2ChainID.OPTIMISM,
            contracts: overrides,
          })

          const addresses = CONTRACT_ADDRESSES[messenger.l2ChainId]
          for (const [contractName, contractAddress] of Object.entries(
            addresses.l1
          )) {
            if (overrides.l1[contractName]) {
              const contract = messenger.contracts.l1[contractName]
              expect(contract.address).to.equal(overrides.l1[contractName])
            } else {
              const contract = messenger.contracts.l1[contractName]
              expect(contract.address).to.equal(contractAddress)
            }
          }
          for (const [contractName, contractAddress] of Object.entries(
            addresses.l2
          )) {
            if (overrides.l2[contractName]) {
              const contract = messenger.contracts.l2[contractName]
              expect(contract.address).to.equal(overrides.l2[contractName])
            } else {
              const contract = messenger.contracts.l2[contractName]
              expect(contract.address).to.equal(contractAddress)
            }
          }
        })
      })

      describe('when given an unknown L2 chain ID', () => {
        describe('when all L1 addresses are provided', () => {
          it('should use custom addresses where provided', () => {
            const overrides = {
              l1: {
                AddressManager: '0x' + '11'.repeat(20),
                L1CrossDomainMessenger: '0x' + '12'.repeat(20),
                L1StandardBridge: '0x' + '13'.repeat(20),
                StateCommitmentChain: '0x' + '14'.repeat(20),
                CanonicalTransactionChain: '0x' + '15'.repeat(20),
                BondManager: '0x' + '16'.repeat(20),
                OptimismPortal: '0x' + '17'.repeat(20),
                L2OutputOracle: '0x' + '18'.repeat(20),
              },
              l2: {
                L2CrossDomainMessenger: '0x' + '22'.repeat(20),
              },
            }

            const messenger = new CrossChainMessenger({
              l1SignerOrProvider: ethers.provider,
              l2SignerOrProvider: 'https://localhost:8545',
              l1ChainId: L1ChainID.MAINNET,
              l2ChainId: 1234,
              contracts: overrides,
            })

            const addresses = CONTRACT_ADDRESSES[L2ChainID.OPTIMISM]
            for (const [contractName, contractAddress] of Object.entries(
              addresses.l1
            )) {
              if (overrides.l1[contractName]) {
                const contract = messenger.contracts.l1[contractName]
                expect(contract.address).to.equal(overrides.l1[contractName])
              } else {
                const contract = messenger.contracts.l1[contractName]
                expect(contract.address).to.equal(contractAddress)
              }
            }
            for (const [contractName, contractAddress] of Object.entries(
              addresses.l2
            )) {
              if (overrides.l2[contractName]) {
                const contract = messenger.contracts.l2[contractName]
                expect(contract.address).to.equal(overrides.l2[contractName])
              } else {
                const contract = messenger.contracts.l2[contractName]
                expect(contract.address).to.equal(contractAddress)
              }
            }
          })
        })

        describe('when not all L1 addresses are provided', () => {
          it('should throw an error', () => {
            expect(() => {
              new CrossChainMessenger({
                l1SignerOrProvider: ethers.provider,
                l2SignerOrProvider: 'https://localhost:8545',
                l1ChainId: L1ChainID.MAINNET,
                l2ChainId: 1234,
                contracts: {
                  l1: {
                    // Missing some required L1 addresses
                    AddressManager: '0x' + '11'.repeat(20),
                    L1CrossDomainMessenger: '0x' + '12'.repeat(20),
                    L1StandardBridge: '0x' + '13'.repeat(20),
                  },
                  l2: {
                    L2CrossDomainMessenger: '0x' + '22'.repeat(20),
                  },
                },
              })
            }).to.throw()
          })
        })
      })
    })
  })

  describe('getMessagesByTransaction', () => {
    let l1Messenger: Contract
    let l2Messenger: Contract
    let messenger: CrossChainMessenger
    beforeEach(async () => {
      l1Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any
      l2Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any

      messenger = new CrossChainMessenger({
        l1SignerOrProvider: ethers.provider,
        l2SignerOrProvider: ethers.provider,
        l1ChainId: L1ChainID.HARDHAT_LOCAL,
        l2ChainId: L2ChainID.OPTIMISM_HARDHAT_LOCAL,
        contracts: {
          l1: {
            L1CrossDomainMessenger: l1Messenger.address,
          },
          l2: {
            L2CrossDomainMessenger: l2Messenger.address,
          },
        },
      })
    })

    describe('when a direction is specified', () => {
      describe('when the transaction exists', () => {
        describe('when the transaction has messages', () => {
          for (const n of [1, 2, 4, 8]) {
            it(`should find ${n} messages when the transaction emits ${n} messages`, async () => {
              const messages = [...Array(n)].map(() => {
                return DUMMY_MESSAGE
              })

              const tx = await l1Messenger.triggerSentMessageEvents(messages)
              const found = await messenger.getMessagesByTransaction(tx, {
                direction: MessageDirection.L1_TO_L2,
              })
              expect(found).to.deep.equal(
                messages.map((message, i) => {
                  return {
                    direction: MessageDirection.L1_TO_L2,
                    sender: message.sender,
                    target: message.target,
                    message: message.message,
                    messageNonce: ethers.BigNumber.from(message.messageNonce),
                    minGasLimit: ethers.BigNumber.from(message.minGasLimit),
                    value: ethers.BigNumber.from(message.value),
                    logIndex: i,
                    blockNumber: tx.blockNumber,
                    transactionHash: tx.hash,
                  }
                })
              )
            })
          }
        })

        describe('when the transaction has no messages', () => {
          it('should find nothing', async () => {
            const tx = await l1Messenger.doNothing()
            const found = await messenger.getMessagesByTransaction(tx, {
              direction: MessageDirection.L1_TO_L2,
            })
            expect(found).to.deep.equal([])
          })
        })
      })

      describe('when the transaction does not exist in the specified direction', () => {
        it('should throw an error', async () => {
          await expect(
            messenger.getMessagesByTransaction('0x' + '11'.repeat(32), {
              direction: MessageDirection.L1_TO_L2,
            })
          ).to.be.rejectedWith('unable to find transaction receipt')
        })
      })
    })

    describe('when a direction is not specified', () => {
      describe('when the transaction exists only on L1', () => {
        describe('when the transaction has messages', () => {
          for (const n of [1, 2, 4, 8]) {
            it(`should find ${n} messages when the transaction emits ${n} messages`, async () => {
              const messages = [...Array(n)].map(() => {
                return DUMMY_MESSAGE
              })

              const tx = await l1Messenger.triggerSentMessageEvents(messages)
              const found = await messenger.getMessagesByTransaction(tx)
              expect(found).to.deep.equal(
                messages.map((message, i) => {
                  return {
                    direction: MessageDirection.L1_TO_L2,
                    sender: message.sender,
                    target: message.target,
                    message: message.message,
                    messageNonce: ethers.BigNumber.from(message.messageNonce),
                    minGasLimit: ethers.BigNumber.from(message.minGasLimit),
                    value: ethers.BigNumber.from(message.value),
                    logIndex: i,
                    blockNumber: tx.blockNumber,
                    transactionHash: tx.hash,
                  }
                })
              )
            })
          }
        })

        describe('when the transaction has no messages', () => {
          it('should find nothing', async () => {
            const tx = await l1Messenger.doNothing()
            const found = await messenger.getMessagesByTransaction(tx)
            expect(found).to.deep.equal([])
          })
        })
      })

      describe('when the transaction exists only on L2', () => {
        describe('when the transaction has messages', () => {
          for (const n of [1, 2, 4, 8]) {
            it(`should find ${n} messages when the transaction emits ${n} messages`, () => {
              // TODO: Need support for simulating more than one network.
            })
          }
        })

        describe('when the transaction has no messages', () => {
          it('should find nothing', () => {
            // TODO: Need support for simulating more than one network.
          })
        })
      })

      describe('when the transaction does not exist', () => {
        it('should throw an error', async () => {
          await expect(
            messenger.getMessagesByTransaction('0x' + '11'.repeat(32))
          ).to.be.rejectedWith('unable to find transaction receipt')
        })
      })

      describe('when the transaction exists on both L1 and L2', () => {
        it('should throw an error', async () => {
          // TODO: Need support for simulating more than one network.
        })
      })
    })
  })

  // Skipped until getMessagesByAddress can be implemented
  describe.skip('getMessagesByAddress', () => {
    describe('when the address has sent messages', () => {
      describe('when no direction is specified', () => {
        it('should find all messages sent by the address')
      })

      describe('when a direction is specified', () => {
        it('should find all messages only in the given direction')
      })

      describe('when a block range is specified', () => {
        it('should find all messages within the block range')
      })

      describe('when both a direction and a block range are specified', () => {
        it(
          'should find all messages only in the given direction and within the block range'
        )
      })
    })

    describe('when the address has not sent messages', () => {
      it('should find nothing')
    })
  })

  describe('toCrossChainMessage', () => {
    let l1Bridge: Contract
    let l2Bridge: Contract
    let l1Messenger: Contract
    let l2Messenger: Contract
    let messenger: CrossChainMessenger
    beforeEach(async () => {
      l1Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any
      l2Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any
      l1Bridge = (await (
        await ethers.getContractFactory('MockBridge')
      ).deploy(l1Messenger.address)) as any
      l2Bridge = (await (
        await ethers.getContractFactory('MockBridge')
      ).deploy(l2Messenger.address)) as any

      messenger = new CrossChainMessenger({
        l1SignerOrProvider: ethers.provider,
        l2SignerOrProvider: ethers.provider,
        l1ChainId: L1ChainID.HARDHAT_LOCAL,
        l2ChainId: L2ChainID.OPTIMISM_HARDHAT_LOCAL,
        contracts: {
          l1: {
            L1CrossDomainMessenger: l1Messenger.address,
            L1StandardBridge: l1Bridge.address,
          },
          l2: {
            L2CrossDomainMessenger: l2Messenger.address,
            L2StandardBridge: l2Bridge.address,
          },
        },
        bridges: {
          Standard: {
            Adapter: StandardBridgeAdapter,
            l1Bridge: l1Bridge.address,
            l2Bridge: l2Bridge.address,
          },
        },
      })
    })

    describe('when the input is a CrossChainMessage', () => {
      it('should return the input', async () => {
        const message = {
          ...DUMMY_EXTENDED_MESSAGE,
          direction: MessageDirection.L1_TO_L2,
        }

        expect(await messenger.toCrossChainMessage(message)).to.deep.equal(
          message
        )
      })
    })

    describe('when the input is a TokenBridgeMessage', () => {
      // TODO: There are some edge cases here with custom bridges that conform to the interface but
      // not to the behavioral spec. Possibly worth testing those. For now this is probably
      // sufficient.
      it('should return the sent message event that came after the deposit or withdrawal', async () => {
        const from = '0x' + '99'.repeat(20)
        const deposit = {
          l1Token: '0x' + '11'.repeat(20),
          l2Token: '0x' + '22'.repeat(20),
          from,
          to: '0x' + '44'.repeat(20),
          amount: ethers.BigNumber.from(1234),
          data: '0x1234',
        }

        const tx = await l1Bridge.emitERC20DepositInitiated(deposit)

        const foundCrossChainMessages =
          await messenger.getMessagesByTransaction(tx)
        const foundTokenBridgeMessages = await messenger.getDepositsByAddress(
          from
        )
        const resolved = await messenger.toCrossChainMessage(
          foundTokenBridgeMessages[0]
        )

        expect(resolved).to.deep.equal(foundCrossChainMessages[0])
      })
    })

    describe('when the input is a TransactionLike', () => {
      describe('when the transaction sent exactly one message', () => {
        it('should return the CrossChainMessage sent in the transaction', async () => {
          const tx = await l1Messenger.triggerSentMessageEvents([DUMMY_MESSAGE])
          const foundCrossChainMessages =
            await messenger.getMessagesByTransaction(tx)
          const resolved = await messenger.toCrossChainMessage(tx)
          expect(resolved).to.deep.equal(foundCrossChainMessages[0])
        })
      })

      describe('when the transaction sent more than one message', () => {
        it('should throw an error', async () => {
          const messages = [...Array(2)].map(() => {
            return DUMMY_MESSAGE
          })

          const tx = await l1Messenger.triggerSentMessageEvents(messages)
          await expect(messenger.toCrossChainMessage(tx)).to.be.rejectedWith(
            'expected 1 message, got 2'
          )
        })
      })

      describe('when the transaction sent no messages', () => {
        it('should throw an error', async () => {
          const tx = await l1Messenger.triggerSentMessageEvents([])
          await expect(messenger.toCrossChainMessage(tx)).to.be.rejectedWith(
            'expected 1 message, got 0'
          )
        })
      })
    })
  })

  describe('getMessageStatus', () => {
    let scc: Contract
    let l1Messenger: Contract
    let l2Messenger: Contract
    let messenger: CrossChainMessenger
    beforeEach(async () => {
      // TODO: Get rid of the nested awaits here. Could be a good first issue for someone.
      scc = (await (await ethers.getContractFactory('MockSCC')).deploy()) as any
      l1Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any
      l2Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any

      messenger = new CrossChainMessenger({
        l1SignerOrProvider: ethers.provider,
        l2SignerOrProvider: ethers.provider,
        l1ChainId: L1ChainID.HARDHAT_LOCAL,
        l2ChainId: L2ChainID.OPTIMISM_HARDHAT_LOCAL,
        contracts: {
          l1: {
            L1CrossDomainMessenger: l1Messenger.address,
            StateCommitmentChain: scc.address,
          },
          l2: {
            L2CrossDomainMessenger: l2Messenger.address,
          },
        },
      })
    })

    const sendAndGetDummyMessage = async (direction: MessageDirection) => {
      const mockMessenger =
        direction === MessageDirection.L1_TO_L2 ? l1Messenger : l2Messenger
      const tx = await mockMessenger.triggerSentMessageEvents([DUMMY_MESSAGE])
      return (
        await messenger.getMessagesByTransaction(tx, {
          direction,
        })
      )[0]
    }

    const submitStateRootBatchForMessage = async (
      message: CrossChainMessage
    ) => {
      await scc.setSBAParams({
        batchIndex: 0,
        batchRoot: ethers.constants.HashZero,
        batchSize: 1,
        prevTotalElements: message.blockNumber,
        extraData: '0x',
      })
      await scc.appendStateBatch([ethers.constants.HashZero], 0)
    }

    describe('when the message is an L1 => L2 message', () => {
      describe('when the message has not been executed on L2 yet', () => {
        it('should return a status of UNCONFIRMED_L1_TO_L2_MESSAGE', async () => {
          const message = await sendAndGetDummyMessage(
            MessageDirection.L1_TO_L2
          )

          expect(await messenger.getMessageStatus(message)).to.equal(
            MessageStatus.UNCONFIRMED_L1_TO_L2_MESSAGE
          )
        })
      })

      describe('when the message has been executed on L2', () => {
        it('should return a status of RELAYED', async () => {
          const message = await sendAndGetDummyMessage(
            MessageDirection.L1_TO_L2
          )

          await l2Messenger.triggerRelayedMessageEvents([
            hashCrossDomainMessage(
              message.messageNonce,
              message.sender,
              message.target,
              message.value,
              message.minGasLimit,
              message.message
            ),
          ])

          expect(await messenger.getMessageStatus(message)).to.equal(
            MessageStatus.RELAYED
          )
        })
      })

      describe('when the message has been executed but failed', () => {
        it('should return a status of FAILED_L1_TO_L2_MESSAGE', async () => {
          const message = await sendAndGetDummyMessage(
            MessageDirection.L1_TO_L2
          )

          await l2Messenger.triggerFailedRelayedMessageEvents([
            hashCrossDomainMessage(
              message.messageNonce,
              message.sender,
              message.target,
              message.value,
              message.minGasLimit,
              message.message
            ),
          ])

          expect(await messenger.getMessageStatus(message)).to.equal(
            MessageStatus.FAILED_L1_TO_L2_MESSAGE
          )
        })
      })
    })

    describe('when the message is an L2 => L1 message', () => {
      describe('when the message state root has not been published', () => {
        it('should return a status of STATE_ROOT_NOT_PUBLISHED', async () => {
          const message = await sendAndGetDummyMessage(
            MessageDirection.L2_TO_L1
          )

          expect(await messenger.getMessageStatus(message)).to.equal(
            MessageStatus.STATE_ROOT_NOT_PUBLISHED
          )
        })
      })

      describe('when the message state root is still in the challenge period', () => {
        it('should return a status of IN_CHALLENGE_PERIOD', async () => {
          const message = await sendAndGetDummyMessage(
            MessageDirection.L2_TO_L1
          )

          await submitStateRootBatchForMessage(message)

          expect(await messenger.getMessageStatus(message)).to.equal(
            MessageStatus.IN_CHALLENGE_PERIOD
          )
        })
      })

      describe('when the message is no longer in the challenge period', () => {
        describe('when the message has been relayed successfully', () => {
          it('should return a status of RELAYED', async () => {
            const message = await sendAndGetDummyMessage(
              MessageDirection.L2_TO_L1
            )

            await submitStateRootBatchForMessage(message)

            const challengePeriod = await messenger.getChallengePeriodSeconds()
            ethers.provider.send('evm_increaseTime', [challengePeriod + 1])
            ethers.provider.send('evm_mine', [])

            await l1Messenger.triggerRelayedMessageEvents([
              hashCrossDomainMessage(
                message.messageNonce,
                message.sender,
                message.target,
                message.value,
                message.minGasLimit,
                message.message
              ),
            ])

            expect(await messenger.getMessageStatus(message)).to.equal(
              MessageStatus.RELAYED
            )
          })
        })

        describe('when the message has been relayed but the relay failed', () => {
          it('should return a status of READY_FOR_RELAY', async () => {
            const message = await sendAndGetDummyMessage(
              MessageDirection.L2_TO_L1
            )

            await submitStateRootBatchForMessage(message)

            const challengePeriod = await messenger.getChallengePeriodSeconds()
            ethers.provider.send('evm_increaseTime', [challengePeriod + 1])
            ethers.provider.send('evm_mine', [])

            await l1Messenger.triggerFailedRelayedMessageEvents([
              hashCrossDomainMessage(
                message.messageNonce,
                message.sender,
                message.target,
                message.value,
                message.minGasLimit,
                message.message
              ),
            ])

            expect(await messenger.getMessageStatus(message)).to.equal(
              MessageStatus.READY_FOR_RELAY
            )
          })
        })

        describe('when the message has not been relayed', () => {
          it('should return a status of READY_FOR_RELAY', async () => {
            const message = await sendAndGetDummyMessage(
              MessageDirection.L2_TO_L1
            )

            await submitStateRootBatchForMessage(message)

            const challengePeriod = await messenger.getChallengePeriodSeconds()
            ethers.provider.send('evm_increaseTime', [challengePeriod + 1])
            ethers.provider.send('evm_mine', [])

            expect(await messenger.getMessageStatus(message)).to.equal(
              MessageStatus.READY_FOR_RELAY
            )
          })
        })
      })
    })

    describe('when the message does not exist', () => {
      // TODO: Figure out if this is the correct behavior. Mark suggests perhaps returning null.
      it('should throw an error')
    })
  })

  describe('getMessageReceipt', () => {
    let l1Bridge: Contract
    let l2Bridge: Contract
    let l1Messenger: Contract
    let l2Messenger: Contract
    let messenger: CrossChainMessenger
    beforeEach(async () => {
      l1Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any
      l2Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any
      l1Bridge = (await (
        await ethers.getContractFactory('MockBridge')
      ).deploy(l1Messenger.address)) as any
      l2Bridge = (await (
        await ethers.getContractFactory('MockBridge')
      ).deploy(l2Messenger.address)) as any

      messenger = new CrossChainMessenger({
        l1SignerOrProvider: ethers.provider,
        l2SignerOrProvider: ethers.provider,
        l1ChainId: L1ChainID.HARDHAT_LOCAL,
        l2ChainId: L2ChainID.OPTIMISM_HARDHAT_LOCAL,
        contracts: {
          l1: {
            L1CrossDomainMessenger: l1Messenger.address,
            L1StandardBridge: l1Bridge.address,
          },
          l2: {
            L2CrossDomainMessenger: l2Messenger.address,
            L2StandardBridge: l2Bridge.address,
          },
        },
      })
    })

    describe('when the message has been relayed', () => {
      describe('when the relay was successful', () => {
        it('should return the receipt of the transaction that relayed the message', async () => {
          const message = {
            ...DUMMY_EXTENDED_MESSAGE,
            direction: MessageDirection.L1_TO_L2,
          }

          const tx = await l2Messenger.triggerRelayedMessageEvents([
            hashCrossDomainMessage(
              message.messageNonce,
              message.sender,
              message.target,
              message.value,
              message.minGasLimit,
              message.message
            ),
          ])

          const messageReceipt = await messenger.getMessageReceipt(message)
          expect(messageReceipt.receiptStatus).to.equal(1)
          expect(
            omit(messageReceipt.transactionReceipt, 'confirmations')
          ).to.deep.equal(
            omit(
              await ethers.provider.getTransactionReceipt(tx.hash),
              'confirmations'
            )
          )
        })
      })

      describe('when the relay failed', () => {
        it('should return the receipt of the transaction that attempted to relay the message', async () => {
          const message = {
            ...DUMMY_EXTENDED_MESSAGE,
            direction: MessageDirection.L1_TO_L2,
          }

          const tx = await l2Messenger.triggerFailedRelayedMessageEvents([
            hashCrossDomainMessage(
              message.messageNonce,
              message.sender,
              message.target,
              message.value,
              message.minGasLimit,
              message.message
            ),
          ])

          const messageReceipt = await messenger.getMessageReceipt(message)
          expect(messageReceipt.receiptStatus).to.equal(0)
          expect(
            omit(messageReceipt.transactionReceipt, 'confirmations')
          ).to.deep.equal(
            omit(
              await ethers.provider.getTransactionReceipt(tx.hash),
              'confirmations'
            )
          )
        })
      })

      describe('when the relay failed more than once', () => {
        it('should return the receipt of the last transaction that attempted to relay the message', async () => {
          const message = {
            ...DUMMY_EXTENDED_MESSAGE,
            direction: MessageDirection.L1_TO_L2,
          }

          await l2Messenger.triggerFailedRelayedMessageEvents([
            hashCrossDomainMessage(
              message.messageNonce,
              message.sender,
              message.target,
              message.value,
              message.minGasLimit,
              message.message
            ),
          ])

          const tx = await l2Messenger.triggerFailedRelayedMessageEvents([
            hashCrossDomainMessage(
              message.messageNonce,
              message.sender,
              message.target,
              message.value,
              message.minGasLimit,
              message.message
            ),
          ])

          const messageReceipt = await messenger.getMessageReceipt(message)
          expect(messageReceipt.receiptStatus).to.equal(0)
          expect(
            omit(messageReceipt.transactionReceipt, 'confirmations')
          ).to.deep.equal(
            omit(
              await ethers.provider.getTransactionReceipt(tx.hash),
              'confirmations'
            )
          )
        })
      })
    })

    describe('when the message has not been relayed', () => {
      it('should return null', async () => {
        const message = {
          ...DUMMY_EXTENDED_MESSAGE,
          direction: MessageDirection.L1_TO_L2,
        }

        await l2Messenger.doNothing()

        const messageReceipt = await messenger.getMessageReceipt(message)
        expect(messageReceipt).to.equal(null)
      })
    })

    // TODO: Go over all of these tests and remove the empty functions so we can accurately keep
    // track of
  })

  describe('waitForMessageReceipt', () => {
    let l2Messenger: Contract
    let messenger: CrossChainMessenger
    beforeEach(async () => {
      l2Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any

      messenger = new CrossChainMessenger({
        l1SignerOrProvider: ethers.provider,
        l2SignerOrProvider: ethers.provider,
        l1ChainId: L1ChainID.HARDHAT_LOCAL,
        l2ChainId: L2ChainID.OPTIMISM_HARDHAT_LOCAL,
        contracts: {
          l2: {
            L2CrossDomainMessenger: l2Messenger.address,
          },
        },
      })
    })

    describe('when the message receipt already exists', () => {
      it('should immediately return the receipt', async () => {
        const message = {
          ...DUMMY_EXTENDED_MESSAGE,
          direction: MessageDirection.L1_TO_L2,
        }

        const tx = await l2Messenger.triggerRelayedMessageEvents([
          hashCrossDomainMessage(
            message.messageNonce,
            message.sender,
            message.target,
            message.value,
            message.minGasLimit,
            message.message
          ),
        ])

        const messageReceipt = await messenger.waitForMessageReceipt(message)
        expect(messageReceipt.receiptStatus).to.equal(1)
        expect(
          omit(messageReceipt.transactionReceipt, 'confirmations')
        ).to.deep.equal(
          omit(
            await ethers.provider.getTransactionReceipt(tx.hash),
            'confirmations'
          )
        )
      })
    })

    describe('when the message receipt does not exist already', () => {
      describe('when no extra options are provided', () => {
        it('should wait for the receipt to be published', async () => {
          const message = {
            ...DUMMY_EXTENDED_MESSAGE,
            direction: MessageDirection.L1_TO_L2,
          }

          setTimeout(async () => {
            await l2Messenger.triggerRelayedMessageEvents([
              hashCrossDomainMessage(
                message.messageNonce,
                message.sender,
                message.target,
                message.value,
                message.minGasLimit,
                message.message
              ),
            ])
          }, 5000)

          const tick = Date.now()
          const messageReceipt = await messenger.waitForMessageReceipt(message)
          const tock = Date.now()
          expect(messageReceipt.receiptStatus).to.equal(1)
          expect(tock - tick).to.be.greaterThan(5000)
        })

        it('should wait forever for the receipt if the receipt is never published', () => {
          // Not sure how to easily test this without introducing some sort of cancellation token
          // I don't want the promise to loop forever and make the tests never finish.
        })
      })

      describe('when a timeout is provided', () => {
        it('should throw an error if the timeout is reached', async () => {
          const message = {
            ...DUMMY_EXTENDED_MESSAGE,
            direction: MessageDirection.L1_TO_L2,
          }

          await expect(
            messenger.waitForMessageReceipt(message, {
              timeoutMs: 10000,
            })
          ).to.be.rejectedWith('timed out waiting for message receipt')
        })
      })
    })
  })

  describe('estimateL2MessageGasLimit', () => {
    let messenger: CrossChainMessenger
    beforeEach(async () => {
      messenger = new CrossChainMessenger({
        l1SignerOrProvider: ethers.provider,
        l2SignerOrProvider: ethers.provider,
        l1ChainId: L1ChainID.HARDHAT_LOCAL,
        l2ChainId: L2ChainID.OPTIMISM_HARDHAT_LOCAL,
      })
    })

    describe('when the message is an L1 to L2 message', () => {
      it('should return an accurate gas estimate plus a ~20% buffer', async () => {
        const message = {
          direction: MessageDirection.L1_TO_L2,
          target: '0x' + '11'.repeat(20),
          sender: '0x' + '22'.repeat(20),
          message: '0x' + '33'.repeat(64),
          messageNonce: 1234,
          logIndex: 0,
          blockNumber: 1234,
          transactionHash: '0x' + '44'.repeat(32),
        }

        const estimate = await ethers.provider.estimateGas({
          to: message.target,
          from: message.sender,
          data: message.message,
        })

        // Approximately 20% greater than the estimate, +/- 1%.
        expectApprox(
          await messenger.estimateL2MessageGasLimit(message),
          estimate.mul(120).div(100),
          {
            percentUpperDeviation: 1,
            percentLowerDeviation: 1,
          }
        )
      })

      it('should return an accurate gas estimate when a custom buffer is provided', async () => {
        const message = {
          direction: MessageDirection.L1_TO_L2,
          target: '0x' + '11'.repeat(20),
          sender: '0x' + '22'.repeat(20),
          message: '0x' + '33'.repeat(64),
          messageNonce: 1234,
          logIndex: 0,
          blockNumber: 1234,
          transactionHash: '0x' + '44'.repeat(32),
        }

        const estimate = await ethers.provider.estimateGas({
          to: message.target,
          from: message.sender,
          data: message.message,
        })

        // Approximately 30% greater than the estimate, +/- 1%.
        expectApprox(
          await messenger.estimateL2MessageGasLimit(message, {
            bufferPercent: 30,
          }),
          estimate.mul(130).div(100),
          {
            percentUpperDeviation: 1,
            percentLowerDeviation: 1,
          }
        )
      })
    })

    describe('when the message is an L2 to L1 message', () => {
      it('should throw an error', async () => {
        const message = {
          direction: MessageDirection.L2_TO_L1,
          target: '0x' + '11'.repeat(20),
          sender: '0x' + '22'.repeat(20),
          message: '0x' + '33'.repeat(64),
          messageNonce: 1234,
          logIndex: 0,
          blockNumber: 1234,
          transactionHash: '0x' + '44'.repeat(32),
        }

        await expect(messenger.estimateL2MessageGasLimit(message)).to.be
          .rejected
      })
    })
  })

  describe('estimateMessageWaitTimeSeconds', () => {
    let scc: Contract
    let l1Messenger: Contract
    let l2Messenger: Contract
    let messenger: CrossChainMessenger
    beforeEach(async () => {
      // TODO: Get rid of the nested awaits here. Could be a good first issue for someone.
      scc = (await (await ethers.getContractFactory('MockSCC')).deploy()) as any
      l1Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any
      l2Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any

      messenger = new CrossChainMessenger({
        l1SignerOrProvider: ethers.provider,
        l2SignerOrProvider: ethers.provider,
        l1ChainId: L1ChainID.HARDHAT_LOCAL,
        l2ChainId: L2ChainID.OPTIMISM_HARDHAT_LOCAL,
        contracts: {
          l1: {
            L1CrossDomainMessenger: l1Messenger.address,
            StateCommitmentChain: scc.address,
          },
          l2: {
            L2CrossDomainMessenger: l2Messenger.address,
          },
        },
      })
    })

    const sendAndGetDummyMessage = async (direction: MessageDirection) => {
      const mockMessenger =
        direction === MessageDirection.L1_TO_L2 ? l1Messenger : l2Messenger
      const tx = await mockMessenger.triggerSentMessageEvents([DUMMY_MESSAGE])
      return (
        await messenger.getMessagesByTransaction(tx, {
          direction,
        })
      )[0]
    }

    const submitStateRootBatchForMessage = async (
      message: CrossChainMessage
    ) => {
      await scc.setSBAParams({
        batchIndex: 0,
        batchRoot: ethers.constants.HashZero,
        batchSize: 1,
        prevTotalElements: message.blockNumber,
        extraData: '0x',
      })
      await scc.appendStateBatch([ethers.constants.HashZero], 0)
    }

    describe('when the message is an L1 => L2 message', () => {
      describe('when the message has not been executed on L2 yet', () => {
        it('should return the estimated seconds until the message will be confirmed on L2', async () => {
          const message = await sendAndGetDummyMessage(
            MessageDirection.L1_TO_L2
          )

          await l1Messenger.triggerSentMessageEvents([message])

          expect(
            await messenger.estimateMessageWaitTimeSeconds(message)
          ).to.equal(1)
        })
      })

      describe('when the message has been executed on L2', () => {
        it('should return 0', async () => {
          const message = await sendAndGetDummyMessage(
            MessageDirection.L1_TO_L2
          )

          await l1Messenger.triggerSentMessageEvents([message])
          await l2Messenger.triggerRelayedMessageEvents([
            hashCrossDomainMessage(
              message.messageNonce,
              message.sender,
              message.target,
              message.value,
              message.minGasLimit,
              message.message
            ),
          ])

          expect(
            await messenger.estimateMessageWaitTimeSeconds(message)
          ).to.equal(0)
        })
      })
    })

    describe('when the message is an L2 => L1 message', () => {
      describe('when the state root has not been published', () => {
        it('should return the estimated seconds until the state root will be published and pass the challenge period', async () => {
          const message = await sendAndGetDummyMessage(
            MessageDirection.L2_TO_L1
          )

          expect(
            await messenger.estimateMessageWaitTimeSeconds(message)
          ).to.equal(await messenger.getChallengePeriodSeconds())
        })
      })

      describe('when the state root is within the challenge period', () => {
        it('should return the estimated seconds until the state root passes the challenge period', async () => {
          const message = await sendAndGetDummyMessage(
            MessageDirection.L2_TO_L1
          )

          await submitStateRootBatchForMessage(message)

          const challengePeriod = await messenger.getChallengePeriodSeconds()
          ethers.provider.send('evm_increaseTime', [challengePeriod / 2])
          ethers.provider.send('evm_mine', [])

          expectApprox(
            await messenger.estimateMessageWaitTimeSeconds(message),
            challengePeriod / 2,
            {
              percentUpperDeviation: 5,
              percentLowerDeviation: 5,
            }
          )
        })
      })

      describe('when the state root passes the challenge period', () => {
        it('should return 0', async () => {
          const message = await sendAndGetDummyMessage(
            MessageDirection.L2_TO_L1
          )

          await submitStateRootBatchForMessage(message)

          const challengePeriod = await messenger.getChallengePeriodSeconds()
          ethers.provider.send('evm_increaseTime', [challengePeriod + 1])
          ethers.provider.send('evm_mine', [])

          expect(
            await messenger.estimateMessageWaitTimeSeconds(message)
          ).to.equal(0)
        })
      })

      describe('when the message has been executed', () => {
        it('should return 0', async () => {
          const message = await sendAndGetDummyMessage(
            MessageDirection.L2_TO_L1
          )

          await l2Messenger.triggerSentMessageEvents([message])
          await l1Messenger.triggerRelayedMessageEvents([
            hashCrossDomainMessage(
              message.messageNonce,
              message.sender,
              message.target,
              message.value,
              message.minGasLimit,
              message.message
            ),
          ])

          expect(
            await messenger.estimateMessageWaitTimeSeconds(message)
          ).to.equal(0)
        })
      })
    })
  })

  describe('sendMessage', () => {
    let l1Messenger: Contract
    let l2Messenger: Contract
    let messenger: CrossChainMessenger
    beforeEach(async () => {
      l1Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any
      l2Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any

      messenger = new CrossChainMessenger({
        l1SignerOrProvider: l1Signer,
        l2SignerOrProvider: l2Signer,
        l1ChainId: L1ChainID.HARDHAT_LOCAL,
        l2ChainId: L2ChainID.OPTIMISM_HARDHAT_LOCAL,
        contracts: {
          l1: {
            L1CrossDomainMessenger: l1Messenger.address,
          },
          l2: {
            L2CrossDomainMessenger: l2Messenger.address,
          },
        },
      })
    })

    describe('when the message is an L1 to L2 message', () => {
      describe('when no l2GasLimit is provided', () => {
        it('should send a message with an estimated l2GasLimit', async () => {
          const message = {
            direction: MessageDirection.L1_TO_L2,
            target: '0x' + '11'.repeat(20),
            message: '0x' + '22'.repeat(32),
          }

          const estimate = await messenger.estimateL2MessageGasLimit(message)
          await expect(messenger.sendMessage(message))
            .to.emit(l1Messenger, 'SentMessage')
            .withArgs(
              message.target,
              await l1Signer.getAddress(),
              message.message,
              0,
              estimate
            )
        })
      })

      describe('when an l2GasLimit is provided', () => {
        it('should send a message with the provided l2GasLimit', async () => {
          const message = {
            direction: MessageDirection.L1_TO_L2,
            target: '0x' + '11'.repeat(20),
            message: '0x' + '22'.repeat(32),
          }

          await expect(
            messenger.sendMessage(message, {
              l2GasLimit: 1234,
            })
          )
            .to.emit(l1Messenger, 'SentMessage')
            .withArgs(
              message.target,
              await l1Signer.getAddress(),
              message.message,
              0,
              1234
            )
        })
      })
    })

    describe('when the message is an L2 to L1 message', () => {
      it('should send a message', async () => {
        const message = {
          direction: MessageDirection.L2_TO_L1,
          target: '0x' + '11'.repeat(20),
          message: '0x' + '22'.repeat(32),
        }

        await expect(messenger.sendMessage(message))
          .to.emit(l2Messenger, 'SentMessage')
          .withArgs(
            message.target,
            await l2Signer.getAddress(),
            message.message,
            0,
            0
          )
      })
    })
  })

  describe('resendMessage', () => {
    let l1Messenger: Contract
    let l2Messenger: Contract
    let messenger: CrossChainMessenger
    beforeEach(async () => {
      l1Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any
      l2Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any

      messenger = new CrossChainMessenger({
        l1SignerOrProvider: l1Signer,
        l2SignerOrProvider: l2Signer,
        l1ChainId: L1ChainID.HARDHAT_LOCAL,
        l2ChainId: L2ChainID.OPTIMISM_HARDHAT_LOCAL,
        contracts: {
          l1: {
            L1CrossDomainMessenger: l1Messenger.address,
          },
          l2: {
            L2CrossDomainMessenger: l2Messenger.address,
          },
        },
      })
    })

    describe('when resending an L1 to L2 message', () => {
      it('should resend the message with the new gas limit', async () => {
        const message = {
          direction: MessageDirection.L1_TO_L2,
          target: '0x' + '11'.repeat(20),
          message: '0x' + '22'.repeat(32),
        }

        const sent = await messenger.sendMessage(message, {
          l2GasLimit: 1234,
        })

        await expect(messenger.resendMessage(sent, 10000))
          .to.emit(l1Messenger, 'SentMessage')
          .withArgs(
            message.target,
            await l1Signer.getAddress(),
            message.message,
            1, // nonce is now 1
            10000
          )
      })
    })

    describe('when resending an L2 to L1 message', () => {
      it('should throw an error', async () => {
        const message = {
          direction: MessageDirection.L2_TO_L1,
          target: '0x' + '11'.repeat(20),
          message: '0x' + '22'.repeat(32),
        }

        const sent = await messenger.sendMessage(message, {
          l2GasLimit: 1234,
        })

        await expect(messenger.resendMessage(sent, 10000)).to.be.rejected
      })
    })
  })

  describe('finalizeMessage', () => {
    describe('when the message being finalized exists', () => {
      describe('when the message is ready to be finalized', () => {
        it('should finalize the message')
      })

      describe('when the message is not ready to be finalized', () => {
        it('should throw an error')
      })

      describe('when the message has already been finalized', () => {
        it('should throw an error')
      })
    })

    describe('when the message being finalized does not exist', () => {
      it('should throw an error')
    })
  })

  describe('depositETH', () => {
    let l1Messenger: Contract
    let l2Messenger: Contract
    let l1Bridge: Contract
    let l2Bridge: Contract
    let messenger: CrossChainMessenger
    beforeEach(async () => {
      l1Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any
      l1Bridge = (await (
        await ethers.getContractFactory('MockBridge')
      ).deploy(l1Messenger.address)) as any
      l2Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any
      l2Bridge = (await (
        await ethers.getContractFactory('MockBridge')
      ).deploy(l2Messenger.address)) as any

      messenger = new CrossChainMessenger({
        l1SignerOrProvider: l1Signer,
        l2SignerOrProvider: l2Signer,
        l1ChainId: L1ChainID.HARDHAT_LOCAL,
        l2ChainId: L2ChainID.OPTIMISM_HARDHAT_LOCAL,
        contracts: {
          l1: {
            L1CrossDomainMessenger: l1Messenger.address,
            L1StandardBridge: l1Bridge.address,
          },
          l2: {
            L2CrossDomainMessenger: l2Messenger.address,
            L2StandardBridge: l2Bridge.address,
          },
        },
        bridges: {
          ETH: {
            Adapter: ETHBridgeAdapter,
            l1Bridge: l1Bridge.address,
            l2Bridge: l2Bridge.address,
          },
        },
      })
    })

    it('should trigger the deposit ETH function with the given amount', async () => {
      await expect(messenger.depositETH(100000))
        .to.emit(l1Bridge, 'ETHDepositInitiated')
        .withArgs(
          await l1Signer.getAddress(),
          await l1Signer.getAddress(),
          100000,
          '0x'
        )
    })
  })

  describe('withdrawETH', () => {
    let l1Messenger: Contract
    let l2Messenger: Contract
    let l1Bridge: Contract
    let l2Bridge: Contract
    let messenger: CrossChainMessenger
    beforeEach(async () => {
      l1Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any
      l1Bridge = (await (
        await ethers.getContractFactory('MockBridge')
      ).deploy(l1Messenger.address)) as any
      l2Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any
      l2Bridge = (await (
        await ethers.getContractFactory('MockBridge')
      ).deploy(l2Messenger.address)) as any

      messenger = new CrossChainMessenger({
        l1SignerOrProvider: l1Signer,
        l2SignerOrProvider: l2Signer,
        l1ChainId: L1ChainID.HARDHAT_LOCAL,
        l2ChainId: L2ChainID.OPTIMISM_HARDHAT_LOCAL,
        contracts: {
          l1: {
            L1CrossDomainMessenger: l1Messenger.address,
            L1StandardBridge: l1Bridge.address,
          },
          l2: {
            L2CrossDomainMessenger: l2Messenger.address,
            L2StandardBridge: l2Bridge.address,
          },
        },
        bridges: {
          ETH: {
            Adapter: ETHBridgeAdapter,
            l1Bridge: l1Bridge.address,
            l2Bridge: l2Bridge.address,
          },
        },
      })
    })

    it('should trigger the withdraw ETH function with the given amount', async () => {
      await expect(messenger.withdrawETH(100000))
        .to.emit(l2Bridge, 'WithdrawalInitiated')
        .withArgs(
          ethers.constants.AddressZero,
          predeploys.OVM_ETH,
          await l2Signer.getAddress(),
          await l2Signer.getAddress(),
          100000,
          '0x'
        )
    })
  })
})
