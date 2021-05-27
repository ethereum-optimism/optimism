/* Imports: External */
import hre from 'hardhat'
import { expect } from 'chai'
import { toPlainObject } from 'lodash'
import { BigNumber } from 'ethers'

/* Imports: Internal */
import { MockContract, smockit } from '../../src'

describe('[smock]: function manipulation tests', () => {
  const ethers = (hre as any).ethers

  let mock: MockContract
  beforeEach(async () => {
    mock = await smockit('TestHelpers_BasicReturnContract')
  })

  describe('manipulating fallback functions', () => {
    it('should return with no data by default', async () => {
      const expected = '0x'

      expect(
        await ethers.provider.call({
          to: mock.address,
        })
      ).to.equal(expected)
    })

    it('should be able to make a fallback function return without any data', async () => {
      const expected = '0x'
      mock.smocked.fallback.will.return()

      expect(
        await ethers.provider.call({
          to: mock.address,
        })
      ).to.equal(expected)
    })

    it('should be able to make a fallback function return with data', async () => {
      const expected = '0x1234123412341234'
      mock.smocked.fallback.will.return.with(expected)

      expect(
        await ethers.provider.call({
          to: mock.address,
        })
      ).to.equal(expected)
    })

    it('should be able to make a fallback function revert without any data', async () => {
      mock.smocked.fallback.will.revert()

      await expect(
        ethers.provider.call({
          to: mock.address,
        })
      ).to.be.reverted
    })

    it('should be able to make a fallback function revert with a string', async () => {
      const expected = 'this is a revert message'

      mock.smocked.fallback.will.revert.with(expected)

      await expect(
        ethers.provider.call({
          to: mock.address,
        })
      ).to.be.revertedWith(expected)
    })

    it('should be able to make a fallback function emit an event', async () => {
      // TODO
    })

    it('should be able to change behaviors', async () => {
      mock.smocked.fallback.will.revert()

      await expect(
        ethers.provider.call({
          to: mock.address,
        })
      ).to.be.reverted

      const expected = '0x'
      mock.smocked.fallback.will.return()

      expect(
        await ethers.provider.call({
          to: mock.address,
        })
      ).to.equal(expected)
    })

    describe.skip('resetting the fallback function', () => {
      it('should go back to default behavior when reset', async () => {
        mock.smocked.fallback.will.revert()

        await expect(
          ethers.provider.call({
            to: mock.address,
          })
        ).to.be.reverted

        const expected = '0x'
        mock.smocked.fallback.reset()

        expect(
          await ethers.provider.call({
            to: mock.address,
          })
        ).to.equal(expected)
      })
    })
  })

  describe('manipulating functions', () => {
    it('should be able to make a function return without any data', async () => {
      const expected = []
      mock.smocked.empty.will.return()

      expect(await mock.callStatic.empty()).to.deep.equal(expected)
    })

    it('should be able to make a function revert without any data', async () => {
      mock.smocked.empty.will.revert()

      await expect(mock.callStatic.empty()).to.be.reverted
    })

    it('should be able to make a function emit an event', async () => {
      // TODO
    })

    describe('overloaded functions', () => {
      it('should be able to modify both versions of an overloaded function', async () => {
        const expected1 = 1234
        const expected2 = 5678
        mock.smocked['overloadedFunction(uint256)'].will.return.with(expected1)
        mock.smocked['overloadedFunction(uint256,uint256)'].will.return.with(
          expected2
        )
        expect(
          await mock.callStatic['overloadedFunction(uint256)'](0)
        ).to.equal(expected1)
        expect(
          await mock.callStatic['overloadedFunction(uint256,uint256)'](0, 0)
        ).to.equal(expected2)
      })
    })

    describe('returning with data', () => {
      describe('fixed data types', () => {
        describe('default behaviors', () => {
          it('should return false for a boolean', async () => {
            const expected = false

            expect(await mock.callStatic.getBoolean()).to.equal(expected)
          })

          it('should return zero for a uint256', async () => {
            const expected = 0

            expect(await mock.callStatic.getUint256()).to.equal(expected)
          })

          it('should return 32 zero bytes for a bytes32', async () => {
            const expected =
              '0x0000000000000000000000000000000000000000000000000000000000000000'

            expect(await mock.callStatic.getBytes32()).to.equal(expected)
          })
        })

        describe('from a specified value', () => {
          it('should be able to return a boolean', async () => {
            const expected = true
            mock.smocked.getBoolean.will.return.with(expected)

            expect(await mock.callStatic.getBoolean()).to.equal(expected)
          })

          it('should be able to return a uint256', async () => {
            const expected = 1234
            mock.smocked.getUint256.will.return.with(expected)

            expect(await mock.callStatic.getUint256()).to.equal(expected)
          })

          it('should be able to return a bytes32', async () => {
            const expected =
              '0x1234123412341234123412341234123412341234123412341234123412341234'
            mock.smocked.getBytes32.will.return.with(expected)

            expect(await mock.callStatic.getBytes32()).to.equal(expected)
          })
        })

        describe('from a function', () => {
          describe('without input arguments', () => {
            it('should be able to return a boolean', async () => {
              const expected = true
              mock.smocked.getBoolean.will.return.with(() => {
                return expected
              })

              expect(await mock.callStatic.getBoolean()).to.equal(expected)
            })

            it('should be able to return a uint256', async () => {
              const expected = 1234
              mock.smocked.getUint256.will.return.with(() => {
                return expected
              })

              expect(await mock.callStatic.getUint256()).to.equal(expected)
            })

            it('should be able to return a bytes32', async () => {
              const expected =
                '0x1234123412341234123412341234123412341234123412341234123412341234'
              mock.smocked.getBytes32.will.return.with(() => {
                return expected
              })

              expect(await mock.callStatic.getBytes32()).to.equal(expected)
            })
          })

          describe('with input arguments', () => {
            it('should be able to return a boolean', async () => {
              const expected = true
              mock.smocked.getInputtedBoolean.will.return.with(
                (arg1: boolean) => {
                  return arg1
                }
              )

              expect(
                await mock.callStatic.getInputtedBoolean(expected)
              ).to.equal(expected)
            })

            it('should be able to return a uint256', async () => {
              const expected = 1234
              mock.smocked.getInputtedUint256.will.return.with(
                (arg1: number) => {
                  return arg1
                }
              )

              expect(
                await mock.callStatic.getInputtedUint256(expected)
              ).to.equal(expected)
            })

            it('should be able to return a bytes32', async () => {
              const expected =
                '0x1234123412341234123412341234123412341234123412341234123412341234'
              mock.smocked.getInputtedBytes32.will.return.with(
                (arg1: string) => {
                  return arg1
                }
              )

              expect(
                await mock.callStatic.getInputtedBytes32(expected)
              ).to.equal(expected)
            })
          })
        })

        describe('from an asynchronous function', () => {
          describe('without input arguments', () => {
            it('should be able to return a boolean', async () => {
              const expected = async () => {
                return true
              }
              mock.smocked.getBoolean.will.return.with(async () => {
                return expected()
              })

              expect(await mock.callStatic.getBoolean()).to.equal(
                await expected()
              )
            })

            it('should be able to return a uint256', async () => {
              const expected = async () => {
                return 1234
              }
              mock.smocked.getUint256.will.return.with(async () => {
                return expected()
              })

              expect(await mock.callStatic.getUint256()).to.equal(
                await expected()
              )
            })

            it('should be able to return a bytes32', async () => {
              const expected = async () => {
                return '0x1234123412341234123412341234123412341234123412341234123412341234'
              }
              mock.smocked.getBytes32.will.return.with(async () => {
                return expected()
              })

              expect(await mock.callStatic.getBytes32()).to.equal(
                await expected()
              )
            })
          })
        })

        describe.skip('resetting function behavior', () => {
          describe('for a boolean', () => {
            it('should return false after resetting', async () => {
              const expected1 = true
              mock.smocked.getBoolean.will.return.with(expected1)

              expect(await mock.callStatic.getBoolean()).to.equal(expected1)

              const expected2 = false
              mock.smocked.getBoolean.reset()

              expect(await mock.callStatic.getBoolean()).to.equal(expected2)
            })

            it('should be able to reset and change behaviors', async () => {
              const expected1 = true
              mock.smocked.getBoolean.will.return.with(expected1)

              expect(await mock.callStatic.getBoolean()).to.equal(expected1)

              const expected2 = false
              mock.smocked.getBoolean.reset()

              expect(await mock.callStatic.getBoolean()).to.equal(expected2)

              const expected3 = true
              mock.smocked.getBoolean.will.return.with(expected3)

              expect(await mock.callStatic.getBoolean()).to.equal(expected3)
            })
          })

          describe('for a uint256', () => {
            it('should return zero after resetting', async () => {
              const expected1 = 1234
              mock.smocked.getUint256.will.return.with(expected1)

              expect(await mock.callStatic.getUint256()).to.equal(expected1)

              const expected2 = 0
              mock.smocked.getUint256.reset()

              expect(await mock.callStatic.getUint256()).to.equal(expected2)
            })

            it('should be able to reset and change behaviors', async () => {
              const expected1 = 1234
              mock.smocked.getUint256.will.return.with(expected1)

              expect(await mock.callStatic.getUint256()).to.equal(expected1)

              const expected2 = 0
              mock.smocked.getUint256.reset()

              expect(await mock.callStatic.getUint256()).to.equal(expected2)

              const expected3 = 4321
              mock.smocked.getUint256.will.return.with(expected3)

              expect(await mock.callStatic.getUint256()).to.equal(expected3)
            })
          })

          describe('for a bytes32', () => {
            it('should return 32 zero bytes after resetting', async () => {
              const expected1 =
                '0x1234123412341234123412341234123412341234123412341234123412341234'
              mock.smocked.getBytes32.will.return.with(expected1)

              expect(await mock.callStatic.getBytes32()).to.equal(expected1)

              const expected2 =
                '0x0000000000000000000000000000000000000000000000000000000000000000'
              mock.smocked.getBytes32.reset()

              expect(await mock.callStatic.getBytes32()).to.equal(expected2)
            })

            it('should be able to reset and change behaviors', async () => {
              const expected1 =
                '0x1234123412341234123412341234123412341234123412341234123412341234'
              mock.smocked.getBytes32.will.return.with(expected1)

              expect(await mock.callStatic.getBytes32()).to.equal(expected1)

              const expected2 =
                '0x0000000000000000000000000000000000000000000000000000000000000000'
              mock.smocked.getBytes32.reset()

              expect(await mock.callStatic.getBytes32()).to.equal(expected2)

              const expected3 =
                '0x4321432143214321432143214321432143214321432143214321432143214321'
              mock.smocked.getBytes32.will.return.with(expected3)

              expect(await mock.callStatic.getBytes32()).to.equal(expected3)
            })
          })
        })
      })

      describe('dynamic data types', () => {
        describe('from a specified value', () => {
          it('should be able to return a bytes value', async () => {
            const expected =
              '0x56785678567856785678567856785678567856785678567856785678567856785678567856785678567856785678567856785678567856785678567856785678'
            mock.smocked.getBytes.will.return.with(expected)

            expect(await mock.callStatic.getBytes()).to.equal(expected)
          })

          it('should be able to return a string value', async () => {
            const expected = 'this is an expected return string'
            mock.smocked.getString.will.return.with(expected)

            expect(await mock.callStatic.getString()).to.equal(expected)
          })

          it('should be able to return a struct with fixed size values', async () => {
            const expected = {
              valBoolean: true,
              valUint256: BigNumber.from(1234),
              valBytes32:
                '0x1234123412341234123412341234123412341234123412341234123412341234',
            }
            mock.smocked.getStructFixedSize.will.return.with(expected)

            const result = toPlainObject(
              await mock.callStatic.getStructFixedSize()
            )
            expect(result.valBoolean).to.equal(expected.valBoolean)
            expect(result.valUint256).to.deep.equal(expected.valUint256)
            expect(result.valBytes32).to.equal(expected.valBytes32)
          })

          it('should be able to return a struct with dynamic size values', async () => {
            const expected = {
              valBytes:
                '0x56785678567856785678567856785678567856785678567856785678567856785678567856785678567856785678567856785678567856785678567856785678',
              valString: 'this is an expected return string',
            }
            mock.smocked.getStructDynamicSize.will.return.with(expected)

            const result = toPlainObject(
              await mock.callStatic.getStructDynamicSize()
            )
            expect(result.valBytes).to.equal(expected.valBytes)
            expect(result.valString).to.equal(expected.valString)
          })

          it('should be able to return a struct with both fixed and dynamic size values', async () => {
            const expected = {
              valBoolean: true,
              valUint256: BigNumber.from(1234),
              valBytes32:
                '0x1234123412341234123412341234123412341234123412341234123412341234',
              valBytes:
                '0x56785678567856785678567856785678567856785678567856785678567856785678567856785678567856785678567856785678567856785678567856785678',
              valString: 'this is an expected return string',
            }
            mock.smocked.getStructMixedSize.will.return.with(expected)

            const result = toPlainObject(
              await mock.callStatic.getStructMixedSize()
            )
            expect(result.valBoolean).to.equal(expected.valBoolean)
            expect(result.valUint256).to.deep.equal(expected.valUint256)
            expect(result.valBytes32).to.equal(expected.valBytes32)
            expect(result.valBytes).to.equal(expected.valBytes)
            expect(result.valString).to.equal(expected.valString)
          })

          it('should be able to return a nested struct', async () => {
            const expected = {
              valStructFixedSize: {
                valBoolean: true,
                valUint256: BigNumber.from(1234),
                valBytes32:
                  '0x1234123412341234123412341234123412341234123412341234123412341234',
              },
              valStructDynamicSize: {
                valBytes:
                  '0x56785678567856785678567856785678567856785678567856785678567856785678567856785678567856785678567856785678567856785678567856785678',
                valString: 'this is an expected return string',
              },
            }
            mock.smocked.getStructNested.will.return.with(expected)

            const result = toPlainObject(
              await mock.callStatic.getStructNested()
            )
            expect(result.valStructFixedSize[0]).to.deep.equal(
              expected.valStructFixedSize.valBoolean
            )
            expect(result.valStructFixedSize[1]).to.deep.equal(
              expected.valStructFixedSize.valUint256
            )
            expect(result.valStructFixedSize[2]).to.deep.equal(
              expected.valStructFixedSize.valBytes32
            )
            expect(result.valStructDynamicSize[0]).to.deep.equal(
              expected.valStructDynamicSize.valBytes
            )
            expect(result.valStructDynamicSize[1]).to.deep.equal(
              expected.valStructDynamicSize.valString
            )
          })

          it('should be able to return an array of uint256 values', async () => {
            const expected = [1234, 2345, 3456, 4567, 5678, 6789].map((n) => {
              return BigNumber.from(n)
            })
            mock.smocked.getArrayUint256.will.return.with(expected)

            const result = await mock.callStatic.getArrayUint256()
            for (let i = 0; i < result.length; i++) {
              expect(result[i]).to.deep.equal(expected[i])
            }
          })
        })
      })
    })

    describe('reverting with data', () => {
      describe('from a specified value', () => {
        it('should be able to revert with a string value', async () => {
          const expected = 'this is a revert string'
          mock.smocked.getUint256.will.revert.with(expected)

          await expect(mock.callStatic.getUint256()).to.be.revertedWith(
            expected
          )
        })
      })

      describe('from a function', () => {
        it('should be able to revert with a string value', async () => {
          const expected = 'this is a revert string'
          mock.smocked.getUint256.will.revert.with(() => {
            return expected
          })

          await expect(mock.callStatic.getUint256()).to.be.revertedWith(
            expected
          )
        })
      })

      describe('from an asynchronous function', () => {
        it('should be able to revert with a string value', async () => {
          const expected = async () => {
            return 'this is a revert string'
          }
          mock.smocked.getUint256.will.revert.with(async () => {
            return expected()
          })

          await expect(mock.callStatic.getUint256()).to.be.revertedWith(
            await expected()
          )
        })
      })

      describe.skip('resetting function behavior', async () => {
        describe('for a boolean', () => {
          it('should return false after resetting', async () => {
            const expected1 = 'this is a revert string'
            mock.smocked.getBoolean.will.revert.with(expected1)

            await expect(mock.callStatic.getBoolean()).to.be.revertedWith(
              expected1
            )

            const expected2 = false
            mock.smocked.getBoolean.reset()

            expect(await mock.callStatic.getBoolean()).to.equal(expected2)
          })

          it('should be able to reset and change behaviors', async () => {
            const expected1 = 'this is a revert string'
            mock.smocked.getBoolean.will.revert.with(expected1)

            await expect(mock.callStatic.getBoolean()).to.be.revertedWith(
              expected1
            )

            const expected2 = false
            mock.smocked.getBoolean.reset()

            expect(await mock.callStatic.getBoolean()).to.equal(expected2)

            const expected3 = true
            mock.smocked.getBoolean.will.return.with(expected3)

            expect(await mock.callStatic.getBoolean()).to.equal(expected3)
          })
        })

        describe('for a uint256', () => {
          it('should return zero after resetting', async () => {
            const expected1 = 'this is a revert string'
            mock.smocked.getUint256.will.revert.with(expected1)

            await expect(mock.callStatic.getUint256()).to.be.revertedWith(
              expected1
            )

            const expected2 = 0
            mock.smocked.getUint256.reset()

            expect(await mock.callStatic.getUint256()).to.equal(expected2)
          })

          it('should be able to reset and change behaviors', async () => {
            const expected1 = 'this is a revert string'
            mock.smocked.getUint256.will.revert.with(expected1)

            await expect(mock.callStatic.getUint256()).to.be.revertedWith(
              expected1
            )

            const expected2 = 0
            mock.smocked.getUint256.reset()

            expect(await mock.callStatic.getUint256()).to.equal(expected2)

            const expected3 = 1234
            mock.smocked.getUint256.will.return.with(expected3)

            expect(await mock.callStatic.getUint256()).to.equal(expected3)
          })
        })

        describe('for a bytes32', () => {
          it('should return 32 zero bytes after resetting', async () => {
            const expected1 = 'this is a revert string'
            mock.smocked.getBytes32.will.revert.with(expected1)

            await expect(mock.callStatic.getBytes32()).to.be.revertedWith(
              expected1
            )

            const expected2 =
              '0x0000000000000000000000000000000000000000000000000000000000000000'
            mock.smocked.getBytes32.reset()

            expect(await mock.callStatic.getBytes32()).to.equal(expected2)
          })

          it('should be able to reset and change behaviors', async () => {
            const expected1 = 'this is a revert string'
            mock.smocked.getBytes32.will.revert.with(expected1)

            await expect(mock.callStatic.getBytes32()).to.be.revertedWith(
              expected1
            )

            const expected2 =
              '0x0000000000000000000000000000000000000000000000000000000000000000'
            mock.smocked.getBytes32.reset()

            expect(await mock.callStatic.getBytes32()).to.equal(expected2)

            const expected3 =
              '0x4321432143214321432143214321432143214321432143214321432143214321'
            mock.smocked.getBytes32.will.return.with(expected3)

            expect(await mock.callStatic.getBytes32()).to.equal(expected3)
          })
        })
      })
    })
  })
})
