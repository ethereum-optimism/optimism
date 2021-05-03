import { expect } from '../setup'

/* Imports: Internal */
import { parseChugSplashConfig } from '../../src'

describe('ChugSplash config parsing', () => {
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
})
