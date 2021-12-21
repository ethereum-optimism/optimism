/* eslint-disable @typescript-eslint/no-empty-function */
import { expect } from './setup'
import { Provider } from '@ethersproject/abstract-provider'
import { Contract } from 'ethers'
import { ethers } from 'hardhat'
import {
  CrossChainProvider,
  MessageDirection,
  CONTRACT_ADDRESSES,
} from '../src'

describe('CrossChainProvider', () => {
  describe('construction', () => {
    describe('when given an ethers provider for the L1 provider', () => {
      it('should use the provider as the L1 provider', () => {
        const provider = new CrossChainProvider({
          l1Provider: ethers.provider,
          l2Provider: ethers.provider,
          l1ChainId: 1,
        })

        expect(provider.l1Provider).to.equal(ethers.provider)
      })
    })

    describe('when given an ethers provider for the L2 provider', () => {
      it('should use the provider as the L2 provider', () => {
        const provider = new CrossChainProvider({
          l1Provider: ethers.provider,
          l2Provider: ethers.provider,
          l1ChainId: 1,
        })

        expect(provider.l2Provider).to.equal(ethers.provider)
      })
    })

    describe('when given a string as the L1 provider', () => {
      it('should create a JSON-RPC provider for the L1 provider', () => {
        const provider = new CrossChainProvider({
          l1Provider: 'https://localhost:8545',
          l2Provider: ethers.provider,
          l1ChainId: 1,
        })

        expect(Provider.isProvider(provider.l1Provider)).to.be.true
      })
    })

    describe('when given a string as the L2 provider', () => {
      it('should create a JSON-RPC provider for the L2 provider', () => {
        const provider = new CrossChainProvider({
          l1Provider: ethers.provider,
          l2Provider: 'https://localhost:8545',
          l1ChainId: 1,
        })

        expect(Provider.isProvider(provider.l2Provider)).to.be.true
      })
    })

    describe('when no custom contract addresses are provided', () => {
      describe('when given a known chain ID', () => {
        it('should use the contract addresses for the known chain ID', () => {
          const provider = new CrossChainProvider({
            l1Provider: ethers.provider,
            l2Provider: 'https://localhost:8545',
            l1ChainId: 1,
          })

          const addresses = CONTRACT_ADDRESSES[1]
          for (const [contractName, contractAddress] of Object.entries(
            addresses.l1
          )) {
            const contract = provider.contracts.l1[contractName]
            expect(contract.address).to.equal(contractAddress)
          }
          for (const [contractName, contractAddress] of Object.entries(
            addresses.l2
          )) {
            const contract = provider.contracts.l2[contractName]
            expect(contract.address).to.equal(contractAddress)
          }
        })
      })

      describe('when given an unknown chain ID', () => {
        it('should throw an error', () => {
          expect(() => {
            new CrossChainProvider({
              l1Provider: ethers.provider,
              l2Provider: 'https://localhost:8545',
              l1ChainId: 1234,
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
          const provider = new CrossChainProvider({
            l1Provider: ethers.provider,
            l2Provider: 'https://localhost:8545',
            l1ChainId: 1,
            contracts: overrides,
          })

          const addresses = CONTRACT_ADDRESSES[1]
          for (const [contractName, contractAddress] of Object.entries(
            addresses.l1
          )) {
            if (overrides.l1[contractName]) {
              const contract = provider.contracts.l1[contractName]
              expect(contract.address).to.equal(overrides.l1[contractName])
            } else {
              const contract = provider.contracts.l1[contractName]
              expect(contract.address).to.equal(contractAddress)
            }
          }
          for (const [contractName, contractAddress] of Object.entries(
            addresses.l2
          )) {
            if (overrides.l2[contractName]) {
              const contract = provider.contracts.l2[contractName]
              expect(contract.address).to.equal(overrides.l2[contractName])
            } else {
              const contract = provider.contracts.l2[contractName]
              expect(contract.address).to.equal(contractAddress)
            }
          }
        })
      })

      describe('when given an unknown chain ID', () => {
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
              },
              l2: {
                L2CrossDomainMessenger: '0x' + '22'.repeat(20),
              },
            }
            const provider = new CrossChainProvider({
              l1Provider: ethers.provider,
              l2Provider: 'https://localhost:8545',
              l1ChainId: 1234,
              contracts: overrides,
            })

            const addresses = CONTRACT_ADDRESSES[1]
            for (const [contractName, contractAddress] of Object.entries(
              addresses.l1
            )) {
              if (overrides.l1[contractName]) {
                const contract = provider.contracts.l1[contractName]
                expect(contract.address).to.equal(overrides.l1[contractName])
              } else {
                const contract = provider.contracts.l1[contractName]
                expect(contract.address).to.equal(contractAddress)
              }
            }
            for (const [contractName, contractAddress] of Object.entries(
              addresses.l2
            )) {
              if (overrides.l2[contractName]) {
                const contract = provider.contracts.l2[contractName]
                expect(contract.address).to.equal(overrides.l2[contractName])
              } else {
                const contract = provider.contracts.l2[contractName]
                expect(contract.address).to.equal(contractAddress)
              }
            }
          })
        })

        describe('when not all L1 addresses are provided', () => {
          it('should throw an error', () => {
            expect(() => {
              new CrossChainProvider({
                l1Provider: ethers.provider,
                l2Provider: 'https://localhost:8545',
                l1ChainId: 1234,
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
    let provider: CrossChainProvider
    beforeEach(async () => {
      l1Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any
      l2Messenger = (await (
        await ethers.getContractFactory('MockMessenger')
      ).deploy()) as any

      provider = new CrossChainProvider({
        l1Provider: ethers.provider,
        l2Provider: ethers.provider,
        l1ChainId: 31337,
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
                return {
                  target: '0x' + '11'.repeat(20),
                  sender: '0x' + '22'.repeat(20),
                  message: '0x' + '33'.repeat(64),
                  messageNonce: 1234,
                  gasLimit: 100000,
                }
              })

              const tx = await l1Messenger.triggerSentMessageEvents(messages)
              const found = await provider.getMessagesByTransaction(tx, {
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
            const found = await provider.getMessagesByTransaction(tx, {
              direction: MessageDirection.L1_TO_L2,
            })
            expect(found).to.deep.equal([])
          })
        })
      })

      describe('when the transaction does not exist in the specified direction', () => {
        it('should throw an error', async () => {
          await expect(
            provider.getMessagesByTransaction('0x' + '11'.repeat(32), {
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
                return {
                  target: '0x' + '11'.repeat(20),
                  sender: '0x' + '22'.repeat(20),
                  message: '0x' + '33'.repeat(64),
                  messageNonce: 1234,
                  gasLimit: 100000,
                }
              })

              const tx = await l1Messenger.triggerSentMessageEvents(messages)
              const found = await provider.getMessagesByTransaction(tx)
              expect(found).to.deep.equal(
                messages.map((message, i) => {
                  return {
                    direction: MessageDirection.L1_TO_L2,
                    sender: message.sender,
                    target: message.target,
                    message: message.message,
                    messageNonce: ethers.BigNumber.from(message.messageNonce),
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
            const found = await provider.getMessagesByTransaction(tx)
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
            provider.getMessagesByTransaction('0x' + '11'.repeat(32))
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

  describe('getMessagesByAddress', () => {
    describe('when the address has sent messages', () => {
      describe('when no direction is specified', () => {
        it('should find all messages sent by the address', () => {})
      })

      describe('when a direction is specified', () => {
        it('should find all messages only in the given direction', () => {})
      })

      describe('when a block range is specified', () => {
        it('should find all messages within the block range', () => {})
      })

      describe('when both a direction and a block range are specified', () => {
        it('should find all messages only in the given direction and within the block range', () => {})
      })
    })

    describe('when the address has not sent messages', () => {
      it('should find nothing', () => {})
    })
  })

  describe('getTokenBridgeMessagesByAddress', () => {
    let l1Bridge: Contract
    let l2Bridge: Contract
    let l1Messenger: Contract
    let l2Messenger: Contract
    let provider: CrossChainProvider
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

      provider = new CrossChainProvider({
        l1Provider: ethers.provider,
        l2Provider: ethers.provider,
        l1ChainId: 31337,
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

    describe('when the address has made deposits or withdrawals', () => {
      describe('when a direction of L1 => L2 is specified', () => {
        it('should find all deposits made by the address', async () => {
          const from = '0x' + '99'.repeat(20)

          const deposit = {
            l1Token: '0x' + '11'.repeat(20),
            l2Token: '0x' + '22'.repeat(20),
            from,
            to: '0x' + '44'.repeat(20),
            amount: ethers.BigNumber.from(1234),
            data: '0x1234',
          }

          const withdrawal = {
            l1Token: '0x' + '12'.repeat(20),
            l2Token: '0x' + '23'.repeat(20),
            from,
            to: '0x' + '45'.repeat(20),
            amount: ethers.BigNumber.from(5678),
            data: '0x5678',
          }

          await l1Bridge.emitERC20DepositInitiated(deposit)
          await l2Bridge.emitWithdrawalInitiated(withdrawal)

          const found = await provider.getTokenBridgeMessagesByAddress(from, {
            direction: MessageDirection.L1_TO_L2,
          })

          expect(found.length).to.equal(1)
          expect(found[0].amount).to.deep.equal(deposit.amount)
          expect(found[0].data).to.deep.equal(deposit.data)
          expect(found[0].direction).to.equal(MessageDirection.L1_TO_L2)
          expect(found[0].l1Token).to.deep.equal(deposit.l1Token)
          expect(found[0].l2Token).to.deep.equal(deposit.l2Token)
          expect(found[0].from).to.deep.equal(deposit.from)
          expect(found[0].to).to.deep.equal(deposit.to)
        })
      })

      describe('when a direction of L2 => L1 is specified', () => {
        it('should find all withdrawals made by the address', async () => {
          const from = '0x' + '99'.repeat(20)

          const deposit = {
            l1Token: '0x' + '11'.repeat(20),
            l2Token: '0x' + '22'.repeat(20),
            from,
            to: '0x' + '44'.repeat(20),
            amount: ethers.BigNumber.from(1234),
            data: '0x1234',
          }

          const withdrawal = {
            l1Token: '0x' + '12'.repeat(20),
            l2Token: '0x' + '23'.repeat(20),
            from,
            to: '0x' + '45'.repeat(20),
            amount: ethers.BigNumber.from(5678),
            data: '0x5678',
          }

          await l1Bridge.emitERC20DepositInitiated(deposit)
          await l2Bridge.emitWithdrawalInitiated(withdrawal)

          const found = await provider.getTokenBridgeMessagesByAddress(from, {
            direction: MessageDirection.L2_TO_L1,
          })

          expect(found.length).to.equal(1)
          expect(found[0].amount).to.deep.equal(withdrawal.amount)
          expect(found[0].data).to.deep.equal(withdrawal.data)
          expect(found[0].direction).to.equal(MessageDirection.L2_TO_L1)
          expect(found[0].l1Token).to.deep.equal(withdrawal.l1Token)
          expect(found[0].l2Token).to.deep.equal(withdrawal.l2Token)
          expect(found[0].from).to.deep.equal(withdrawal.from)
          expect(found[0].to).to.deep.equal(withdrawal.to)
        })
      })

      describe('when no direction is specified', () => {
        it('should find all deposits and withdrawals made by the address', async () => {
          const from = '0x' + '99'.repeat(20)

          const deposit = {
            l1Token: '0x' + '11'.repeat(20),
            l2Token: '0x' + '22'.repeat(20),
            from,
            to: '0x' + '44'.repeat(20),
            amount: ethers.BigNumber.from(1234),
            data: '0x1234',
          }

          const withdrawal = {
            l1Token: '0x' + '12'.repeat(20),
            l2Token: '0x' + '23'.repeat(20),
            from,
            to: '0x' + '45'.repeat(20),
            amount: ethers.BigNumber.from(5678),
            data: '0x5678',
          }

          await l1Bridge.emitERC20DepositInitiated(deposit)
          await l2Bridge.emitWithdrawalInitiated(withdrawal)

          const found = await provider.getTokenBridgeMessagesByAddress(from)

          expect(found.length).to.equal(2)

          // Check the deposit (deposits get searched first)
          expect(found[0].amount).to.deep.equal(deposit.amount)
          expect(found[0].data).to.deep.equal(deposit.data)
          expect(found[0].direction).to.equal(MessageDirection.L1_TO_L2)
          expect(found[0].l1Token).to.deep.equal(deposit.l1Token)
          expect(found[0].l2Token).to.deep.equal(deposit.l2Token)
          expect(found[0].from).to.deep.equal(deposit.from)
          expect(found[0].to).to.deep.equal(deposit.to)

          // Check the withdrawal
          expect(found[1].amount).to.deep.equal(withdrawal.amount)
          expect(found[1].data).to.deep.equal(withdrawal.data)
          expect(found[1].direction).to.equal(MessageDirection.L2_TO_L1)
          expect(found[1].l1Token).to.deep.equal(withdrawal.l1Token)
          expect(found[1].l2Token).to.deep.equal(withdrawal.l2Token)
          expect(found[1].from).to.deep.equal(withdrawal.from)
          expect(found[1].to).to.deep.equal(withdrawal.to)
        })
      })

      describe('when a block range is specified', () => {
        describe('when a direction is specified', () => {
          it('should find all deposits or withdrawals only in the given direction and within the block range', async () => {
            const from = '0x' + '99'.repeat(20)

            const deposit1 = {
              l1Token: '0x' + '11'.repeat(20),
              l2Token: '0x' + '22'.repeat(20),
              from,
              to: '0x' + '44'.repeat(20),
              amount: ethers.BigNumber.from(1234),
              data: '0x1234',
            }

            const deposit2 = {
              l1Token: '0x' + '33'.repeat(20),
              l2Token: '0x' + '44'.repeat(20),
              from,
              to: '0x' + '55'.repeat(20),
              amount: ethers.BigNumber.from(1234),
              data: '0x1234',
            }

            const withdrawal = {
              l1Token: '0x' + '12'.repeat(20),
              l2Token: '0x' + '23'.repeat(20),
              from,
              to: '0x' + '45'.repeat(20),
              amount: ethers.BigNumber.from(5678),
              data: '0x5678',
            }

            await l1Bridge.emitERC20DepositInitiated(deposit1)
            const tx = await l1Bridge.emitERC20DepositInitiated(deposit2)
            await l2Bridge.emitWithdrawalInitiated(withdrawal)

            const found = await provider.getTokenBridgeMessagesByAddress(from, {
              direction: MessageDirection.L1_TO_L2,
              fromBlock: tx.blockNumber,
            })

            expect(found.length).to.equal(1)
            expect(found[0].amount).to.deep.equal(deposit2.amount)
            expect(found[0].data).to.deep.equal(deposit2.data)
            expect(found[0].direction).to.equal(MessageDirection.L1_TO_L2)
            expect(found[0].l1Token).to.deep.equal(deposit2.l1Token)
            expect(found[0].l2Token).to.deep.equal(deposit2.l2Token)
            expect(found[0].from).to.deep.equal(deposit2.from)
            expect(found[0].to).to.deep.equal(deposit2.to)
          })
        })

        describe('when a direction is not specified', () => {
          it('should throw an error', async () => {
            const from = '0x' + '99'.repeat(20)
            await expect(
              provider.getTokenBridgeMessagesByAddress(from, {
                fromBlock: 0,
                toBlock: 100,
              })
            ).to.be.rejectedWith('direction must be specified')
          })
        })
      })
    })

    describe('when the address has not made any deposits or withdrawals', () => {
      it('should find nothing', async () => {
        const from = '0x' + '99'.repeat(20)
        const found = await provider.getTokenBridgeMessagesByAddress(from)
        expect(found).to.deep.equal([])
      })
    })
  })

  describe('getMessageStatus', () => {
    describe('when the message is an L1 => L2 message', () => {
      describe('when the message has not been executed on L2 yet', () => {
        it('should return a status of UNCONFIRMED_L1_TO_L2_MESSAGE', () => {})
      })

      describe('when the message has been executed on L2', () => {
        it('should return a status of RELAYED', () => {})
      })

      describe('when the message has been executed but failed', () => {
        it('should return a status of FAILED_L1_TO_L2_MESSAGE', () => {})
      })
    })

    describe('when the message is an L2 => L1 message', () => {
      describe('when the message state root has not been published', () => {
        it('should return a status of STATE_ROOT_NOT_PUBLISHED', () => {})
      })

      describe('when the message state root is still in the challenge period', () => {
        it('should return a status of IN_CHALLENGE_PERIOD', () => {})
      })

      describe('when the message is no longer in the challenge period', () => {
        describe('when the message has been relayed successfully', () => {
          it('should return a status of RELAYED', () => {})
        })

        describe('when the message has been relayed but the relay failed', () => {
          it('should return a status of READY_FOR_RELAY', () => {})
        })

        describe('when the message has not been relayed', () => {
          it('should return a status of READY_FOR_RELAY', () => {})
        })
      })
    })

    describe('when the message does not exist', () => {
      it('should throw an error', () => {})
    })
  })

  describe('getMessageReceipt', () => {
    describe('when the message has been relayed', () => {
      describe('when the relay was successful', () => {
        it('should return the receipt of the transaction that relayed the message', () => {})
      })

      describe('when the relay failed', () => {
        it('should return the receipt of the transaction that attempted to relay the message', () => {})
      })

      describe('when the relay failed more than once', () => {
        it('should return the receipt of the last transaction that attempted to relay the message', () => {})
      })
    })

    describe('when the message has not been relayed', () => {
      it('should return null', () => {})
    })

    describe('when the message does not exist', () => {
      it('should throw an error', () => {})
    })
  })

  describe('waitForMessageReciept', () => {
    describe('when the message receipt already exists', () => {
      it('should immediately return the receipt', () => {})
    })

    describe('when the message receipt does not exist already', () => {
      describe('when no extra options are provided', () => {
        it('should wait for the receipt to be published', () => {})
        it('should wait forever for the receipt if the receipt is never published', () => {})
      })

      describe('when a timeout is provided', () => {
        it('should throw an error if the timeout is reached', () => {})
      })
    })

    describe('when the message does not exist', () => {
      it('should throw an error', () => {})
    })
  })

  describe('estimateL2MessageGasLimit', () => {
    it('should perform a gas estimation of the L2 action', () => {})
  })

  describe('estimateMessageWaitTimeBlocks', () => {
    describe('when the message exists', () => {
      describe('when the message is an L1 => L2 message', () => {
        describe('when the message has not been executed on L2 yet', () => {
          it('should return the estimated blocks until the message will be confirmed on L2', () => {})
        })

        describe('when the message has been executed on L2', () => {
          it('should return 0', () => {})
        })
      })

      describe('when the message is an L2 => L1 message', () => {
        describe('when the state root has not been published', () => {
          it('should return the estimated blocks until the state root will be published and pass the challenge period', () => {})
        })

        describe('when the state root is within the challenge period', () => {
          it('should return the estimated blocks until the state root passes the challenge period', () => {})
        })

        describe('when the state root passes the challenge period', () => {
          it('should return 0', () => {})
        })
      })
    })

    describe('when the message does not exist', () => {
      it('should throw an error', () => {})
    })
  })

  describe('estimateMessageWaitTimeSeconds', () => {
    it('should be the result of estimateMessageWaitTimeBlocks multiplied by the L1 block time', () => {})
  })
})
