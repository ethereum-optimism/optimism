/* Imports: External */
import hre from 'hardhat'
import { expect } from 'chai'
import { Contract } from 'ethers'

/* Imports: Internal */
import { MockContract, smockit } from '../../src'

describe('[smock]: call assertion tests', () => {
  const ethers = (hre as any).ethers

  let mock: MockContract
  beforeEach(async () => {
    mock = await smockit('TestHelpers_BasicReturnContract')
  })

  let mockCaller: Contract
  before(async () => {
    const mockCallerFactory = await ethers.getContractFactory(
      'TestHelpers_MockCaller'
    )
    mockCaller = await mockCallerFactory.deploy()
  })

  describe('call assertions for functions', () => {
    it('should be able to make assertions about a non-overloaded function', async () => {
      mock.smocked.getInputtedUint256.will.return.with(0)

      const expected1 = ethers.BigNumber.from(1234)
      await mockCaller.callMock(
        mock.address,
        mock.interface.encodeFunctionData('getInputtedUint256(uint256)', [
          expected1,
        ])
      )

      expect(mock.smocked.getInputtedUint256.calls[0]).to.deep.equal([
        expected1,
      ])
    })

    it('should be able to make assertions about both versions of an overloaded function', async () => {
      mock.smocked['overloadedFunction(uint256)'].will.return.with(0)
      mock.smocked['overloadedFunction(uint256,uint256)'].will.return.with(0)

      const expected1 = ethers.BigNumber.from(1234)
      await mockCaller.callMock(
        mock.address,
        mock.interface.encodeFunctionData('overloadedFunction(uint256)', [
          expected1,
        ])
      )

      expect(
        mock.smocked['overloadedFunction(uint256)'].calls[0]
      ).to.deep.equal([expected1])

      const expected2 = ethers.BigNumber.from(5678)
      await mockCaller.callMock(
        mock.address,
        mock.interface.encodeFunctionData(
          'overloadedFunction(uint256,uint256)',
          [expected2, expected2]
        )
      )

      expect(
        mock.smocked['overloadedFunction(uint256,uint256)'].calls[0]
      ).to.deep.equal([expected2, expected2])
    })
  })
})
