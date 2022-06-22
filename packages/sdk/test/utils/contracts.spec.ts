/* eslint-disable @typescript-eslint/no-empty-function */
import { Signer } from 'ethers'
import { ethers } from 'hardhat'

import { expect } from '../setup'
import {
  getOEContract,
  getAllOEContracts,
  CONTRACT_ADDRESSES,
  DEFAULT_L2_CONTRACT_ADDRESSES,
  L2ChainID,
} from '../../src'

describe('contract connection utils', () => {
  let signers: Signer[]
  before(async () => {
    signers = (await ethers.getSigners()) as any
  })

  describe('getOEContract', () => {
    describe('when given a known chain ID', () => {
      describe('when not given an address override', () => {
        it('should use the address for the given contract name and chain ID', () => {
          const addresses = CONTRACT_ADDRESSES[L2ChainID.OPTIMISM]
          for (const [contractName, contractAddress] of [
            ...Object.entries(addresses.l1),
            ...Object.entries(addresses.l2),
          ]) {
            const contract = getOEContract(
              contractName as any,
              L2ChainID.OPTIMISM
            )
            expect(contract.address).to.equal(contractAddress)
          }
        })
      })

      describe('when given an address override', () => {
        it('should use the custom address', () => {
          const addresses = CONTRACT_ADDRESSES[L2ChainID.OPTIMISM]
          for (const contractName of [
            ...Object.keys(addresses.l1),
            ...Object.keys(addresses.l2),
          ]) {
            const address = '0x' + '11'.repeat(20)
            const contract = getOEContract(contractName as any, 1, {
              address,
            })
            expect(contract.address).to.equal(address)
          }
        })
      })
    })

    describe('when given an unknown chain ID', () => {
      describe('when not given an address override', () => {
        it('should throw an error', () => {
          expect(() => getOEContract('L1CrossDomainMessenger', 3)).to.throw()
        })
      })

      describe('when given an address override', () => {
        it('should use the custom address', () => {
          const address = '0x' + '11'.repeat(20)
          const contract = getOEContract('L1CrossDomainMessenger', 3, {
            address,
          })
          expect(contract.address).to.equal(address)
        })
      })
    })

    describe('when connected to a valid address', () => {
      it('should have the correct interface for the contract name', () => {
        const contract = getOEContract(
          'L1CrossDomainMessenger',
          L2ChainID.OPTIMISM
        )
        expect(contract.sendMessage).to.not.be.undefined
      })

      describe('when not given a signer or provider', () => {
        it('should not have a signer or provider', () => {
          const contract = getOEContract(
            'L1CrossDomainMessenger',
            L2ChainID.OPTIMISM
          )
          expect(contract.signer).to.be.null
          expect(contract.provider).to.be.null
        })
      })

      describe('when given a signer', () => {
        it('should attach the given signer', () => {
          const contract = getOEContract(
            'L1CrossDomainMessenger',
            L2ChainID.OPTIMISM,
            {
              signerOrProvider: signers[0],
            }
          )
          expect(contract.signer).to.deep.equal(signers[0])
        })
      })

      describe('when given a provider', () => {
        it('should attach the given provider', () => {
          const contract = getOEContract(
            'L1CrossDomainMessenger',
            L2ChainID.OPTIMISM,
            {
              signerOrProvider: ethers.provider as any,
            }
          )
          expect(contract.signer).to.be.null
          expect(contract.provider).to.deep.equal(ethers.provider)
        })
      })
    })
  })

  describe('getAllOEContracts', () => {
    describe('when given a known chain ID', () => {
      describe('when not given any address overrides', () => {
        it('should return all contracts connected to the default addresses', () => {
          const contracts = getAllOEContracts(L2ChainID.OPTIMISM)
          const addresses = CONTRACT_ADDRESSES[L2ChainID.OPTIMISM]
          for (const [contractName, contractAddress] of Object.entries(
            addresses.l1
          )) {
            const contract = contracts.l1[contractName]
            expect(contract.address).to.equal(contractAddress)
          }
          for (const [contractName, contractAddress] of Object.entries(
            addresses.l2
          )) {
            const contract = contracts.l2[contractName]
            expect(contract.address).to.equal(contractAddress)
          }
        })
      })

      describe('when given address overrides', () => {
        it('should return contracts connected to the overridden addresses where given', () => {
          const overrides = {
            l1: {
              L1CrossDomainMessenger: '0x' + '11'.repeat(20),
            },
            l2: {
              L2CrossDomainMessenger: '0x' + '22'.repeat(20),
            },
          }
          const contracts = getAllOEContracts(L2ChainID.OPTIMISM, { overrides })
          const addresses = CONTRACT_ADDRESSES[L2ChainID.OPTIMISM]
          for (const [contractName, contractAddress] of Object.entries(
            addresses.l1
          )) {
            const contract = contracts.l1[contractName]
            if (overrides.l1[contractName]) {
              expect(contract.address).to.equal(overrides.l1[contractName])
            } else {
              expect(contract.address).to.equal(contractAddress)
            }
          }
          for (const [contractName, contractAddress] of Object.entries(
            addresses.l2
          )) {
            const contract = contracts.l2[contractName]
            if (overrides.l2[contractName]) {
              expect(contract.address).to.equal(overrides.l2[contractName])
            } else {
              expect(contract.address).to.equal(contractAddress)
            }
          }
        })
      })
    })

    describe('when given an unknown chain ID', () => {
      describe('when given address overrides for all L1 contracts', () => {
        describe('when given address overrides for L2 contracts', () => {
          it('should return contracts connected to the overridden addresses where given', () => {
            const l1Overrides = {}
            for (const contractName of Object.keys(
              CONTRACT_ADDRESSES[L2ChainID.OPTIMISM].l1
            )) {
              l1Overrides[contractName] = '0x' + '11'.repeat(20)
            }

            const contracts = getAllOEContracts(3, {
              overrides: {
                l1: l1Overrides as any,
                l2: {
                  L2CrossDomainMessenger: '0x' + '22'.repeat(20),
                },
              },
            })

            for (const [contractName, contract] of Object.entries(
              contracts.l1
            )) {
              expect(contract.address).to.equal(l1Overrides[contractName])
            }

            expect(contracts.l2.L2CrossDomainMessenger.address).to.equal(
              '0x' + '22'.repeat(20)
            )
          })
        })

        describe('when not given address overrides for L2 contracts', () => {
          it('should return contracts connected to the default L2 addresses and custom L1 addresses', () => {
            const l1Overrides = {}
            for (const contractName of Object.keys(
              CONTRACT_ADDRESSES[L2ChainID.OPTIMISM].l1
            )) {
              l1Overrides[contractName] = '0x' + '11'.repeat(20)
            }

            const contracts = getAllOEContracts(3, {
              overrides: {
                l1: l1Overrides as any,
              },
            })

            for (const [contractName, contract] of Object.entries(
              contracts.l1
            )) {
              expect(contract.address).to.equal(l1Overrides[contractName])
            }

            for (const [contractName, contract] of Object.entries(
              contracts.l2
            )) {
              expect(contract.address).to.equal(
                DEFAULT_L2_CONTRACT_ADDRESSES[contractName]
              )
            }
          })
        })
      })

      describe('when given address overrides for some L1 contracts', () => {
        it('should throw an error', () => {
          expect(() =>
            getAllOEContracts(3, {
              overrides: {
                l1: {
                  L1CrossDomainMessenger: '0x' + '11'.repeat(20),
                },
              },
            })
          ).to.throw()
        })
      })

      describe('when given address overrides for no L1 contracts', () => {
        it('should throw an error', () => {
          expect(() => getAllOEContracts(3)).to.throw()
        })
      })
    })

    describe('when not given a signer or provider', () => {
      it('should not attach a signer or provider to any contracts', () => {
        const contracts = getAllOEContracts(L2ChainID.OPTIMISM)
        for (const contract of Object.values(contracts.l1)) {
          expect(contract.signer).to.be.null
          expect(contract.provider).to.be.null
        }
        for (const contract of Object.values(contracts.l2)) {
          expect(contract.signer).to.be.null
          expect(contract.provider).to.be.null
        }
      })
    })

    describe('when given an L1 signer', () => {
      it('should attach the signer to the L1 contracts only', () => {
        const contracts = getAllOEContracts(L2ChainID.OPTIMISM, {
          l1SignerOrProvider: signers[0],
        })
        for (const contract of Object.values(contracts.l1)) {
          expect(contract.signer).to.deep.equal(signers[0])
        }
        for (const contract of Object.values(contracts.l2)) {
          expect(contract.signer).to.be.null
          expect(contract.provider).to.be.null
        }
      })
    })

    describe('when given an L2 signer', () => {
      it('should attach the signer to the L2 contracts only', () => {
        const contracts = getAllOEContracts(L2ChainID.OPTIMISM, {
          l2SignerOrProvider: signers[0],
        })
        for (const contract of Object.values(contracts.l1)) {
          expect(contract.signer).to.be.null
          expect(contract.provider).to.be.null
        }
        for (const contract of Object.values(contracts.l2)) {
          expect(contract.signer).to.deep.equal(signers[0])
        }
      })
    })

    describe('when given an L1 signer and an L2 signer', () => {
      it('should attach the signer to both sets of contracts', () => {
        const contracts = getAllOEContracts(L2ChainID.OPTIMISM, {
          l1SignerOrProvider: signers[0],
          l2SignerOrProvider: signers[1],
        })
        for (const contract of Object.values(contracts.l1)) {
          expect(contract.signer).to.deep.equal(signers[0])
        }
        for (const contract of Object.values(contracts.l2)) {
          expect(contract.signer).to.deep.equal(signers[1])
        }
      })
    })

    describe('when given an L1 provider', () => {
      it('should attach the provider to the L1 contracts only', () => {
        const contracts = getAllOEContracts(L2ChainID.OPTIMISM, {
          l1SignerOrProvider: ethers.provider as any,
        })
        for (const contract of Object.values(contracts.l1)) {
          expect(contract.signer).to.be.null
          expect(contract.provider).to.deep.equal(ethers.provider)
        }
        for (const contract of Object.values(contracts.l2)) {
          expect(contract.signer).to.be.null
          expect(contract.provider).to.be.null
        }
      })
    })

    describe('when given an L2 provider', () => {
      it('should attach the provider to the L2 contracts only', () => {
        const contracts = getAllOEContracts(L2ChainID.OPTIMISM, {
          l2SignerOrProvider: ethers.provider as any,
        })
        for (const contract of Object.values(contracts.l1)) {
          expect(contract.signer).to.be.null
          expect(contract.provider).to.be.null
        }
        for (const contract of Object.values(contracts.l2)) {
          expect(contract.signer).to.be.null
          expect(contract.provider).to.deep.equal(ethers.provider)
        }
      })
    })

    describe('when given an L1 provider and an L2 provider', () => {
      it('should attach the provider to both sets of contracts', () => {
        const contracts = getAllOEContracts(L2ChainID.OPTIMISM, {
          l1SignerOrProvider: ethers.provider as any,
          l2SignerOrProvider: ethers.provider as any,
        })
        for (const contract of Object.values(contracts.l1)) {
          expect(contract.signer).to.be.null
          expect(contract.provider).to.deep.equal(ethers.provider)
        }
        for (const contract of Object.values(contracts.l2)) {
          expect(contract.signer).to.be.null
          expect(contract.provider).to.deep.equal(ethers.provider)
        }
      })
    })
  })
})
