import { expect } from './setup'

/* Imports: External */
import hre from 'hardhat'
import { remove0x } from '@eth-optimism/core-utils'

/* Imports: Internal */
import {
  parseChugSplashConfig,
  ChugSplashActionType,
  makeActionBundleFromConfig,
} from '../src'
import { NON_NULL_BYTES32, NON_ZERO_ADDRESS } from './helpers'

describe('ChugSplash config parsing', () => {
  const ethers = (hre as any).ethers

  let storageHelperCode: string
  before(async () => {
    const factory = await ethers.getContractFactory('Helper_StorageHelper')
    const contract = await factory.deploy()
    storageHelperCode = await ethers.provider.getCode(contract.address)
  })

  describe('parseChugSplashConfig', () => {
    it('should correctly parse a basic config file with no template variables', () => {
      expect(
        parseChugSplashConfig({
          contracts: {
            MyContract: {
              address: `0x${'11'.repeat(20)}`,
              source: 'MyContract',
              variables: {},
            },
          },
        })
      ).to.deep.equal({
        contracts: {
          MyContract: {
            address: `0x${'11'.repeat(20)}`,
            source: 'MyContract',
            variables: {},
          },
        },
      })
    })

    it('should correctly parse a basic config file with multiple input contracts', () => {
      expect(
        parseChugSplashConfig({
          contracts: {
            MyContract: {
              address: `0x${'11'.repeat(20)}`,
              source: 'MyContract',
              variables: {},
            },
            MyOtherContract: {
              address: `0x${'22'.repeat(20)}`,
              source: 'MyOtherContract',
              variables: {},
            },
          },
        })
      ).to.deep.equal({
        contracts: {
          MyContract: {
            address: `0x${'11'.repeat(20)}`,
            source: 'MyContract',
            variables: {},
          },
          MyOtherContract: {
            address: `0x${'22'.repeat(20)}`,
            source: 'MyOtherContract',
            variables: {},
          },
        },
      })
    })

    it('should correctly parse a config file with a templated variable', () => {
      expect(
        parseChugSplashConfig(
          {
            contracts: {
              MyContract: {
                address: `0x${'11'.repeat(20)}`,
                source: 'MyContract',
                variables: {
                  myVariable: '{{ env.MY_VARIABLE_VALUE }}',
                },
              },
            },
          },
          {
            MY_VARIABLE_VALUE: '1234',
          }
        )
      ).to.deep.equal({
        contracts: {
          MyContract: {
            address: `0x${'11'.repeat(20)}`,
            source: 'MyContract',
            variables: {
              myVariable: '1234',
            },
          },
        },
      })
    })

    it('should correctly parse a config file with multiple templated values', () => {
      expect(
        parseChugSplashConfig(
          {
            contracts: {
              MyContract: {
                address: `0x${'11'.repeat(20)}`,
                source: 'MyContract',
                variables: {
                  myVariable: '{{ env.MY_VARIABLE_VALUE }}',
                  mySecondVariable: '{{ env.MY_SECOND_VARIABLE_VALUE }}',
                  myThirdVariable: '{{ env.MY_THIRD_VARIABLE_VALUE }}',
                },
              },
            },
          },
          {
            MY_VARIABLE_VALUE: '1234',
            MY_SECOND_VARIABLE_VALUE: 'banana',
            MY_THIRD_VARIABLE_VALUE: 'cake',
          }
        )
      ).to.deep.equal({
        contracts: {
          MyContract: {
            address: `0x${'11'.repeat(20)}`,
            source: 'MyContract',
            variables: {
              myVariable: '1234',
              mySecondVariable: 'banana',
              myThirdVariable: 'cake',
            },
          },
        },
      })
    })

    it('should correctly parse a config file with a templated contract address', () => {
      expect(
        parseChugSplashConfig({
          contracts: {
            MyContract: {
              address: `0x${'11'.repeat(20)}`,
              source: 'MyContract',
              variables: {},
            },
            MyOtherContract: {
              address: `0x${'22'.repeat(20)}`,
              source: 'MyOtherContract',
              variables: {
                myContractAddress: '{{ contracts.MyContract }}',
              },
            },
          },
        })
      ).to.deep.equal({
        contracts: {
          MyContract: {
            address: `0x${'11'.repeat(20)}`,
            source: 'MyContract',
            variables: {},
          },
          MyOtherContract: {
            address: `0x${'22'.repeat(20)}`,
            source: 'MyOtherContract',
            variables: {
              myContractAddress: `0x${'11'.repeat(20)}`,
            },
          },
        },
      })
    })

    it('should correctly parse a config file with multiple templated contract addresses', () => {
      expect(
        parseChugSplashConfig({
          contracts: {
            MyContract: {
              address: `0x${'11'.repeat(20)}`,
              source: 'MyContract',
              variables: {},
            },
            MyOtherContract: {
              address: `0x${'22'.repeat(20)}`,
              source: 'MyOtherContract',
              variables: {
                myContractAddress: '{{ contracts.MyContract }}',
              },
            },
            MyThirdContract: {
              address: `0x${'33'.repeat(20)}`,
              source: 'MyThirdContract',
              variables: {
                myContractAddress: '{{ contracts.MyContract }}',
                myOtherContractAddress: '{{ contracts.MyOtherContract }}',
              },
            },
          },
        })
      ).to.deep.equal({
        contracts: {
          MyContract: {
            address: `0x${'11'.repeat(20)}`,
            source: 'MyContract',
            variables: {},
          },
          MyOtherContract: {
            address: `0x${'22'.repeat(20)}`,
            source: 'MyOtherContract',
            variables: {
              myContractAddress: `0x${'11'.repeat(20)}`,
            },
          },
          MyThirdContract: {
            address: `0x${'33'.repeat(20)}`,
            source: 'MyThirdContract',
            variables: {
              myContractAddress: `0x${'11'.repeat(20)}`,
              myOtherContractAddress: `0x${'22'.repeat(20)}`,
            },
          },
        },
      })
    })

    it('should correctly parse a config file with a contract referencing its own address', () => {
      expect(
        parseChugSplashConfig({
          contracts: {
            MyContract: {
              address: `0x${'11'.repeat(20)}`,
              source: 'MyContract',
              variables: {
                myContractAddress: '{{ contracts.MyContract }}',
              },
            },
          },
        })
      ).to.deep.equal({
        contracts: {
          MyContract: {
            address: `0x${'11'.repeat(20)}`,
            source: 'MyContract',
            variables: {
              myContractAddress: `0x${'11'.repeat(20)}`,
            },
          },
        },
      })
    })

    it('should correctly parse a config file with a templated contract address and a templated variable', () => {
      expect(
        parseChugSplashConfig(
          {
            contracts: {
              MyContract: {
                address: `0x${'11'.repeat(20)}`,
                source: 'MyContract',
                variables: {},
              },
              MyOtherContract: {
                address: `0x${'22'.repeat(20)}`,
                source: 'MyOtherContract',
                variables: {
                  myContractAddress: '{{ contracts.MyContract }}',
                  myVariable: '{{ env.MY_VARIABLE_VALUE }}',
                },
              },
            },
          },
          {
            MY_VARIABLE_VALUE: '0x1234',
          }
        )
      ).to.deep.equal({
        contracts: {
          MyContract: {
            address: `0x${'11'.repeat(20)}`,
            source: 'MyContract',
            variables: {},
          },
          MyOtherContract: {
            address: `0x${'22'.repeat(20)}`,
            source: 'MyOtherContract',
            variables: {
              myContractAddress: `0x${'11'.repeat(20)}`,
              myVariable: '0x1234',
            },
          },
        },
      })
    })

    it('should throw an error when a variable is not supplied', () => {
      expect(() => {
        parseChugSplashConfig({
          contracts: {
            MyContract: {
              address: `0x${'11'.repeat(20)}`,
              source: 'MyContract',
              variables: {
                myVariable: '{{ env.MY_FAKE_VARIABLE_VALUE }}',
              },
            },
          },
        })
      }).to.throw(
        'attempted to access unknown env value: MY_FAKE_VARIABLE_VALUE'
      )
    })

    it('should throw an error when accessing a contract that does not exist', () => {
      expect(() => {
        parseChugSplashConfig({
          contracts: {
            MyContract: {
              address: `0x${'11'.repeat(20)}`,
              source: 'MyContract',
              variables: {
                myVariable: '{{ contracts.MyFakeContract }}',
              },
            },
          },
        })
      }).to.throw('attempted to access unknown contract: MyFakeContract')
    })

    it('should throw an error if trying to use a template in an address', () => {
      expect(() => {
        parseChugSplashConfig({
          contracts: {
            MyContract: {
              address: `{{ env.NOT_AN_ADDRESS }}`,
              source: 'MyContract',
              variables: {
                myVariable: '{{ contracts.MyFakeContract }}',
              },
            },
          },
        })
      }).to.throw(
        'contract address is not a valid address: {{ env.NOT_AN_ADDRESS }}'
      )
    })

    it('should throw an error if trying to use a template in a contract source', () => {
      expect(() => {
        parseChugSplashConfig({
          contracts: {
            MyContract: {
              address: `0x${'11'.repeat(20)}`,
              source: '{{ env.MY_CONTRACT_SOURCE }}',
              variables: {
                myVariable: '{{ contracts.MyFakeContract }}',
              },
            },
          },
        })
      }).to.throw(
        'cannot use template strings in contract source names: {{ env.MY_CONTRACT_SOURCE }}'
      )
    })

    it('should throw an error if trying to use a template in a contract name', () => {
      expect(() => {
        parseChugSplashConfig({
          contracts: {
            '{{ env.MY_CONTRACT_NAME }}': {
              address: `0x${'11'.repeat(20)}`,
              source: 'MyContract',
              variables: {
                myVariable: '{{ contracts.MyFakeContract }}',
              },
            },
          },
        })
      }).to.throw(
        'cannot use template strings in contract names: {{ env.MY_CONTRACT_NAME }}'
      )
    })
  })

  describe('makeActionBundleFromConfig', () => {
    it('should make a bundle from config with one contract and no variables', async () => {
      const bundle = await makeActionBundleFromConfig({
        contracts: {
          MyContract: {
            address: `0x${'11'.repeat(20)}`,
            source: 'Helper_StorageHelper',
            variables: {},
          },
        },
      })

      expect(bundle.actions.length).to.equal(1)
      expect(bundle.actions[0].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_CODE,
        target: `0x${'11'.repeat(20)}`,
        data: storageHelperCode,
      })
    })

    it('should make a bundle from config with two contracts and no variables', async () => {
      const bundle = await makeActionBundleFromConfig({
        contracts: {
          MyContract1: {
            address: `0x${'11'.repeat(20)}`,
            source: 'Helper_StorageHelper',
            variables: {},
          },
          MyContract2: {
            address: `0x${'22'.repeat(20)}`,
            source: 'Helper_StorageHelper',
            variables: {},
          },
        },
      })

      expect(bundle.actions.length).to.equal(2)
      expect(bundle.actions[0].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_CODE,
        target: `0x${'11'.repeat(20)}`,
        data: storageHelperCode,
      })
      expect(bundle.actions[1].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_CODE,
        target: `0x${'22'.repeat(20)}`,
        data: storageHelperCode,
      })
    })

    it('should make a bundle from config with one contract with variables', async () => {
      const bundle = await makeActionBundleFromConfig({
        contracts: {
          MyContract1: {
            address: `0x${'11'.repeat(20)}`,
            source: 'Helper_StorageHelper',
            variables: {
              _uint8: 123,
              _bytes32: NON_NULL_BYTES32,
            },
          },
        },
      })

      expect(bundle.actions.length).to.equal(3)
      expect(bundle.actions[0].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_CODE,
        target: `0x${'11'.repeat(20)}`,
        data: storageHelperCode,
      })
      expect(bundle.actions[1].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'11'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            ethers.constants.HashZero,
            '0x000000000000000000000000000000000000000000000000000000000000007b',
          ]
        ),
      })
      expect(bundle.actions[2].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'11'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            '0x000000000000000000000000000000000000000000000000000000000000000a',
            NON_NULL_BYTES32,
          ]
        ),
      })
    })

    it('should make a bundle from config with two contracts with variables', async () => {
      const bundle = await makeActionBundleFromConfig({
        contracts: {
          MyContract1: {
            address: `0x${'11'.repeat(20)}`,
            source: 'Helper_StorageHelper',
            variables: {
              _uint8: 123,
              _bytes32: NON_NULL_BYTES32,
            },
          },
          MyContract2: {
            address: `0x${'22'.repeat(20)}`,
            source: 'Helper_StorageHelper',
            variables: {
              _address: NON_ZERO_ADDRESS,
              _bool: true,
            },
          },
        },
      })

      expect(bundle.actions.length).to.equal(6)
      expect(bundle.actions[0].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_CODE,
        target: `0x${'11'.repeat(20)}`,
        data: storageHelperCode,
      })
      expect(bundle.actions[1].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'11'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            ethers.constants.HashZero,
            '0x000000000000000000000000000000000000000000000000000000000000007b',
          ]
        ),
      })
      expect(bundle.actions[2].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'11'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            '0x000000000000000000000000000000000000000000000000000000000000000a',
            NON_NULL_BYTES32,
          ]
        ),
      })
      expect(bundle.actions[3].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_CODE,
        target: `0x${'22'.repeat(20)}`,
        data: storageHelperCode,
      })
      expect(bundle.actions[4].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'22'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            '0x000000000000000000000000000000000000000000000000000000000000000e',
            `0x000000000000000000000000${remove0x(NON_ZERO_ADDRESS)}`,
          ]
        ),
      })
      expect(bundle.actions[5].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'22'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            '0x000000000000000000000000000000000000000000000000000000000000000c',
            '0x0000000000000000000000000000000000000000000000000000000000000001',
          ]
        ),
      })
    })

    it('should make a bundle from config with one contract and templated variables', async () => {
      const bundle = await makeActionBundleFromConfig(
        {
          contracts: {
            MyContract1: {
              address: `0x${'11'.repeat(20)}`,
              source: 'Helper_StorageHelper',
              variables: {
                _uint8: `{{ env.MY_UINT8_VALUE }}`,
                _bytes32: `{{ env.MY_BYTES32_VALUE }}`,
              },
            },
          },
        },
        {
          MY_UINT8_VALUE: 123,
          MY_BYTES32_VALUE: NON_NULL_BYTES32,
        }
      )

      expect(bundle.actions.length).to.equal(3)
      expect(bundle.actions[0].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_CODE,
        target: `0x${'11'.repeat(20)}`,
        data: storageHelperCode,
      })
      expect(bundle.actions[1].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'11'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            ethers.constants.HashZero,
            '0x000000000000000000000000000000000000000000000000000000000000007b',
          ]
        ),
      })
      expect(bundle.actions[2].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'11'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            '0x000000000000000000000000000000000000000000000000000000000000000a',
            NON_NULL_BYTES32,
          ]
        ),
      })
    })

    it('should make a bundle from config with two contracts with variables and templated variables', async () => {
      const bundle = await makeActionBundleFromConfig(
        {
          contracts: {
            MyContract1: {
              address: `0x${'11'.repeat(20)}`,
              source: 'Helper_StorageHelper',
              variables: {
                _uint8: 123,
                _bytes32: NON_NULL_BYTES32,
              },
            },
            MyContract2: {
              address: `0x${'22'.repeat(20)}`,
              source: 'Helper_StorageHelper',
              variables: {
                _address: `{{ env.MY_ADDRESS_VALUE }}`,
                _bool: `{{ env.MY_BOOLEAN_VALUE }}`,
              },
            },
          },
        },
        {
          MY_ADDRESS_VALUE: NON_ZERO_ADDRESS,
          MY_BOOLEAN_VALUE: true,
        }
      )

      expect(bundle.actions.length).to.equal(6)
      expect(bundle.actions[0].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_CODE,
        target: `0x${'11'.repeat(20)}`,
        data: storageHelperCode,
      })
      expect(bundle.actions[1].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'11'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            ethers.constants.HashZero,
            '0x000000000000000000000000000000000000000000000000000000000000007b',
          ]
        ),
      })
      expect(bundle.actions[2].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'11'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            '0x000000000000000000000000000000000000000000000000000000000000000a',
            NON_NULL_BYTES32,
          ]
        ),
      })
      expect(bundle.actions[3].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_CODE,
        target: `0x${'22'.repeat(20)}`,
        data: storageHelperCode,
      })
      expect(bundle.actions[4].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'22'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            '0x000000000000000000000000000000000000000000000000000000000000000e',
            `0x000000000000000000000000${remove0x(NON_ZERO_ADDRESS)}`,
          ]
        ),
      })
      expect(bundle.actions[5].action).to.deep.equal({
        actionType: ChugSplashActionType.SET_STORAGE,
        target: `0x${'22'.repeat(20)}`,
        data: ethers.utils.defaultAbiCoder.encode(
          ['bytes32', 'bytes32'],
          [
            '0x000000000000000000000000000000000000000000000000000000000000000c',
            '0x0000000000000000000000000000000000000000000000000000000000000001',
          ]
        ),
      })
    })
  })
})
