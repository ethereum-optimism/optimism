import '../../../setup'

/* External Imports */
import { ethers } from 'hardhat'
import { constants, Contract, ContractFactory, Signer } from 'ethers'
import _ from 'lodash'

/* Internal Imports */
import {
  DUMMY_ACCOUNTS,
  DUMMY_BYTES32,
  EMPTY_ACCOUNT_CODE_HASH,
  NON_ZERO_ADDRESS,
  NON_NULL_BYTES32,
  STORAGE_XOR_VALUE,
  GasMeasurement,
} from '../../../helpers'

const DUMMY_ACCOUNT = DUMMY_ACCOUNTS[0]
const DUMMY_KEY = DUMMY_BYTES32[0]
const DUMMY_VALUE_1 = DUMMY_BYTES32[1]
const DUMMY_VALUE_2 = DUMMY_BYTES32[2]

describe('OVM_StateManager gas consumption [ @skip-on-coverage ]', () => {
  let owner: Signer
  before(async () => {
    ;[owner] = await ethers.getSigners()
  })

  let Factory__OVM_StateManager: ContractFactory
  let gasMeasurement: GasMeasurement
  before(async () => {
    Factory__OVM_StateManager = await ethers.getContractFactory(
      'OVM_StateManager'
    )
    gasMeasurement = new GasMeasurement()
    await gasMeasurement.init(owner)
  })

  let OVM_StateManager: Contract
  beforeEach(async () => {
    OVM_StateManager = (
      await Factory__OVM_StateManager.deploy(await owner.getAddress())
    ).connect(owner)

    await OVM_StateManager.setExecutionManager(
      gasMeasurement.GasMeasurementContract.address
    )
  })

  const measure = (
    methodName: string,
    methodArgs: Array<any> = [],
    doFirst: () => Promise<any> = async () => {
      return
    }
  ) => {
    it('measured consumption!', async () => {
      await doFirst()
      const gasCost = await gasMeasurement.getGasCost(
        OVM_StateManager,
        methodName,
        methodArgs
      )
      console.log(`          calculated gas cost of ${gasCost}`)
    })
  }

  const setupFreshAccount = async () => {
    await OVM_StateManager.putAccount(DUMMY_ACCOUNT.address, {
      ...DUMMY_ACCOUNT.data,
      isFresh: true,
    })
  }
  const setupNonFreshAccount = async () => {
    await OVM_StateManager.putAccount(DUMMY_ACCOUNT.address, DUMMY_ACCOUNT.data)
  }
  const putSlot = async (value: string) => {
    await OVM_StateManager.putContractStorage(
      DUMMY_ACCOUNT.address,
      DUMMY_KEY,
      value
    )
  }

  describe('ItemState testAndSetters', () => {
    describe('testAndSetAccountLoaded', () => {
      describe('when account ItemState is ITEM_UNTOUCHED', () => {
        measure('testAndSetAccountLoaded', [NON_ZERO_ADDRESS])
      })
      describe('when account ItemState is ITEM_LOADED', () => {
        measure('testAndSetAccountLoaded', [NON_ZERO_ADDRESS], async () => {
          await OVM_StateManager.testAndSetAccountLoaded(NON_ZERO_ADDRESS)
        })
      })
      describe('when account ItemState is ITEM_CHANGED', () => {
        measure('testAndSetAccountLoaded', [NON_ZERO_ADDRESS], async () => {
          await OVM_StateManager.testAndSetAccountChanged(NON_ZERO_ADDRESS)
        })
      })
    })

    describe('testAndSetAccountChanged', () => {
      describe('when account ItemState is ITEM_UNTOUCHED', () => {
        measure('testAndSetAccountChanged', [NON_ZERO_ADDRESS])
      })
      describe('when account ItemState is ITEM_LOADED', () => {
        measure('testAndSetAccountChanged', [NON_ZERO_ADDRESS], async () => {
          await OVM_StateManager.testAndSetAccountLoaded(NON_ZERO_ADDRESS)
        })
      })
      describe('when account ItemState is ITEM_CHANGED', () => {
        measure('testAndSetAccountChanged', [NON_ZERO_ADDRESS], async () => {
          await OVM_StateManager.testAndSetAccountChanged(NON_ZERO_ADDRESS)
        })
      })
    })

    describe('testAndSetContractStorageLoaded', () => {
      describe('when storage ItemState is ITEM_UNTOUCHED', () => {
        measure('testAndSetContractStorageLoaded', [
          NON_ZERO_ADDRESS,
          DUMMY_KEY,
        ])
      })
      describe('when storage ItemState is ITEM_LOADED', () => {
        measure(
          'testAndSetContractStorageLoaded',
          [NON_ZERO_ADDRESS, DUMMY_KEY],
          async () => {
            await OVM_StateManager.testAndSetContractStorageLoaded(
              NON_ZERO_ADDRESS,
              DUMMY_KEY
            )
          }
        )
      })
      describe('when storage ItemState is ITEM_CHANGED', () => {
        measure(
          'testAndSetContractStorageLoaded',
          [NON_ZERO_ADDRESS, DUMMY_KEY],
          async () => {
            await OVM_StateManager.testAndSetContractStorageChanged(
              NON_ZERO_ADDRESS,
              DUMMY_KEY
            )
          }
        )
      })
    })

    describe('testAndSetContractStorageChanged', () => {
      describe('when storage ItemState is ITEM_UNTOUCHED', () => {
        measure('testAndSetContractStorageChanged', [
          NON_ZERO_ADDRESS,
          DUMMY_KEY,
        ])
      })
      describe('when storage ItemState is ITEM_LOADED', () => {
        measure(
          'testAndSetContractStorageChanged',
          [NON_ZERO_ADDRESS, DUMMY_KEY],
          async () => {
            await OVM_StateManager.testAndSetContractStorageLoaded(
              NON_ZERO_ADDRESS,
              DUMMY_KEY
            )
          }
        )
      })
      describe('when storage ItemState is ITEM_CHANGED', () => {
        measure(
          'testAndSetContractStorageChanged',
          [NON_ZERO_ADDRESS, DUMMY_KEY],
          async () => {
            await OVM_StateManager.testAndSetContractStorageChanged(
              NON_ZERO_ADDRESS,
              DUMMY_KEY
            )
          }
        )
      })
    })
  })

  describe('incrementTotalUncommittedAccounts', () => {
    describe('when totalUncommittedAccounts is 0', () => {
      measure('incrementTotalUncommittedAccounts')
    })
    describe('when totalUncommittedAccounts is nonzero', () => {
      const doFirst = async () => {
        await OVM_StateManager.incrementTotalUncommittedAccounts()
      }
      measure('incrementTotalUncommittedAccounts', [], doFirst)
    })
  })

  describe('incrementTotalUncommittedContractStorage', () => {
    describe('when totalUncommittedContractStorage is 0', () => {
      measure('incrementTotalUncommittedContractStorage')
    })
    describe('when totalUncommittedContractStorage is nonzero', () => {
      const doFirst = async () => {
        await OVM_StateManager.incrementTotalUncommittedContractStorage()
      }
      measure('incrementTotalUncommittedContractStorage', [], doFirst)
    })
  })

  describe('hasAccount', () => {
    describe('when it does have the account', () => {
      const doFirst = async () => {
        await OVM_StateManager.putAccount(
          DUMMY_ACCOUNT.address,
          DUMMY_ACCOUNT.data
        )
      }
      measure('hasAccount', [DUMMY_ACCOUNT.address], doFirst)
    })
  })

  describe('hasEmptyAccount', () => {
    describe('when it does have an empty account', () => {
      measure('hasEmptyAccount', [DUMMY_ACCOUNT.address], async () => {
        await OVM_StateManager.putAccount(DUMMY_ACCOUNT.address, {
          ...DUMMY_ACCOUNT.data,
          codeHash: EMPTY_ACCOUNT_CODE_HASH,
        })
      })
    })
    describe('when it has an account which is not emtpy', () => {
      measure('hasEmptyAccount', [DUMMY_ACCOUNT.address], async () => {
        await OVM_StateManager.putAccount(
          DUMMY_ACCOUNT.address,
          DUMMY_ACCOUNT.data
        )
      })
    })
  })

  describe('setAccountNonce', () => {
    describe('when the nonce is 0 and set to 0', () => {
      measure('setAccountNonce', [DUMMY_ACCOUNT.address, 0], async () => {
        await OVM_StateManager.putAccount(DUMMY_ACCOUNT.address, {
          ...DUMMY_ACCOUNT.data,
          nonce: 0,
        })
      })
    })
    describe('when the nonce is 0 and set to nonzero', () => {
      measure('setAccountNonce', [DUMMY_ACCOUNT.address, 1], async () => {
        await OVM_StateManager.putAccount(DUMMY_ACCOUNT.address, {
          ...DUMMY_ACCOUNT.data,
          nonce: 0,
        })
      })
    })
    describe('when the nonce is nonzero and set to 0', () => {
      measure('setAccountNonce', [DUMMY_ACCOUNT.address, 0], async () => {
        await OVM_StateManager.putAccount(DUMMY_ACCOUNT.address, {
          ...DUMMY_ACCOUNT.data,
          nonce: 1,
        })
      })
    })
    describe('when the nonce is nonzero and set to nonzero', () => {
      measure('setAccountNonce', [DUMMY_ACCOUNT.address, 2], async () => {
        await OVM_StateManager.putAccount(DUMMY_ACCOUNT.address, {
          ...DUMMY_ACCOUNT.data,
          nonce: 1,
        })
      })
    })
  })

  describe('getAccountNonce', () => {
    describe('when the nonce is 0', () => {
      measure('getAccountNonce', [DUMMY_ACCOUNT.address])
    })
    describe('when the nonce is nonzero', () => {
      measure('getAccountNonce', [DUMMY_ACCOUNT.address], async () => {
        await OVM_StateManager.putAccount(DUMMY_ACCOUNT.address, {
          ...DUMMY_ACCOUNT.data,
          nonce: 1,
        })
      })
    })
  })

  describe('getAccountEthAddress', () => {
    describe('when the ethAddress is a random address', () => {
      measure('getAccountEthAddress', [DUMMY_ACCOUNT.address], async () => {
        await OVM_StateManager.putAccount(DUMMY_ACCOUNT.address, {
          ...DUMMY_ACCOUNT.data,
          ethAddress: NON_ZERO_ADDRESS,
        })
      })
    })
    describe('when the ethAddress is zero', () => {
      measure('getAccountEthAddress', [DUMMY_ACCOUNT.address], async () => {
        await OVM_StateManager.putAccount(DUMMY_ACCOUNT.address, {
          ...DUMMY_ACCOUNT.data,
          ethAddress: constants.AddressZero,
        })
      })
    })
  })

  describe('initPendingAccount', () => {
    // note: this method should only be accessibl if _hasEmptyAccount is true, so it should always be empty nonce etc
    measure('initPendingAccount', [NON_ZERO_ADDRESS])
  })

  describe('commitPendingAccount', () => {
    // this should only set ethAddress and codeHash from ZERO to NONZERO, so one case should be sufficient
    measure(
      'commitPendingAccount',
      [NON_ZERO_ADDRESS, NON_ZERO_ADDRESS, NON_NULL_BYTES32],
      async () => {
        await OVM_StateManager.initPendingAccount(NON_ZERO_ADDRESS)
      }
    )
  })

  describe('getContractStorage', () => {
    // confirm with kelvin that this covers all cases
    describe('when the account isFresh', () => {
      describe('when the storage slot value has not been set', () => {
        measure(
          'getContractStorage',
          [DUMMY_ACCOUNT.address, DUMMY_KEY],
          setupFreshAccount
        )
      })
      describe('when the storage slot has already been set', () => {
        describe('when the storage slot value is STORAGE_XOR_VALUE', () => {
          measure(
            'getContractStorage',
            [DUMMY_ACCOUNT.address, DUMMY_KEY],
            async () => {
              await setupFreshAccount()
              await putSlot(STORAGE_XOR_VALUE)
            }
          )
        })
        describe('when the storage slot value is something other than STORAGE_XOR_VALUE', () => {
          measure(
            'getContractStorage',
            [DUMMY_ACCOUNT.address, DUMMY_KEY],
            async () => {
              await setupFreshAccount()
              await putSlot(DUMMY_VALUE_1)
            }
          )
        })
      })
    })
    describe('when the account is not fresh', () => {
      describe('when the storage slot value is STORAGE_XOR_VALUE', () => {
        measure(
          'getContractStorage',
          [DUMMY_ACCOUNT.address, DUMMY_KEY],
          async () => {
            await setupNonFreshAccount()
            await putSlot(STORAGE_XOR_VALUE)
          }
        )
      })
      describe('when the storage slot value is something other than STORAGE_XOR_VALUE', () => {
        measure(
          'getContractStorage',
          [DUMMY_ACCOUNT.address, DUMMY_KEY],
          async () => {
            await setupNonFreshAccount()
            await putSlot(DUMMY_VALUE_1)
          }
        )
      })
    })
  })

  describe('putContractStorage', () => {
    const relevantValues = [DUMMY_VALUE_1, DUMMY_VALUE_2, STORAGE_XOR_VALUE]
    for (const preValue of relevantValues) {
      for (const postValue of relevantValues) {
        describe(`when overwriting ${preValue} with ${postValue}`, () => {
          measure(
            'putContractStorage',
            [NON_ZERO_ADDRESS, DUMMY_KEY, postValue],
            async () => {
              await OVM_StateManager.putContractStorage(
                NON_ZERO_ADDRESS,
                DUMMY_KEY,
                preValue
              )
            }
          )
        })
      }
    }
  })

  describe('hasContractStorage', () => {
    describe('when the account is fresh', () => {
      describe('when the storage slot has not been set', () => {
        measure(
          'hasContractStorage',
          [DUMMY_ACCOUNT.address, DUMMY_KEY],
          setupFreshAccount
        )
      })
      describe('when the slot has already been set', () => {
        measure(
          'hasContractStorage',
          [DUMMY_ACCOUNT.address, DUMMY_KEY],
          async () => {
            await setupFreshAccount()
            await OVM_StateManager.putContractStorage(
              DUMMY_ACCOUNT.address,
              DUMMY_KEY,
              DUMMY_VALUE_1
            )
          }
        )
      })
    })
    describe('when the account is not fresh', () => {
      measure(
        'hasContractStorage',
        [DUMMY_ACCOUNT.address, DUMMY_KEY],
        async () => {
          await OVM_StateManager.putContractStorage(
            DUMMY_ACCOUNT.address,
            DUMMY_KEY,
            DUMMY_VALUE_1
          )
        }
      )
    })
  })
})
