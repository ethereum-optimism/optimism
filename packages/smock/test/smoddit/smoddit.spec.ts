/* Imports: External */
import { expect } from 'chai'
import { BigNumber } from 'ethers'
import _ from 'lodash'

/* Imports: Internal */
import {
  ModifiableContractFactory,
  ModifiableContract,
  smoddit,
} from '../../src/smoddit'

describe('smoddit', () => {
  describe('via contract factory', () => {
    describe('for functions with a single fixed return value', () => {
      let SmodFactory: ModifiableContractFactory
      before(async () => {
        SmodFactory = await smoddit('SimpleStorageGetter')
      })

      let smod: ModifiableContract
      beforeEach(async () => {
        smod = await SmodFactory.deploy(4321)
      })

      it('should be able to return a uint256', async () => {
        const ret = 1234

        smod.smodify.put({
          _uint256: ret,
        })

        expect(await smod.getUint256()).to.equal(ret)
      })

      it('should be able to return a boolean', async () => {
        const ret = true

        smod.smodify.put({
          _bool: ret,
        })

        expect(await smod.getBool()).to.equal(ret)
      })

      it('should be able to return a simple struct', async () => {
        const ret = {
          valueA: BigNumber.from(1234),
          valueB: true,
        }

        smod.smodify.put({
          _SimpleStruct: ret,
        })

        const result = _.toPlainObject(await smod.getSimpleStruct())
        expect(result.valueA).to.deep.equal(ret.valueA)
        expect(result.valueB).to.deep.equal(ret.valueB)
      })

      it('should be able to return a simple uint256 => uint256 mapping value', async () => {
        const retKey = 1234
        const retVal = 5678

        smod.smodify.put({
          _uint256Map: {
            [retKey]: retVal,
          },
        })

        expect(await smod.getUint256MapValue(retKey)).to.equal(retVal)
      })

      it('should be able to return a nested uint256 => uint256 mapping value', async () => {
        const retKeyA = 1234
        const retKeyB = 4321
        const retVal = 5678

        smod.smodify.put({
          _uint256NestedMap: {
            [retKeyA]: {
              [retKeyB]: retVal,
            },
          },
        })

        expect(await smod.getNestedUint256MapValue(retKeyA, retKeyB)).to.equal(
          retVal
        )
      })

      it('should not return the set value if the value has been changed by the contract', async () => {
        const ret = 1234

        smod.smodify.put({
          _uint256: ret,
        })

        await smod.setUint256(4321)

        expect(await smod.getUint256()).to.equal(4321)
      })

      it('should return the set value if it was set in the constructor', async () => {
        const ret = 1234

        smod.smodify.put({
          _constructorUint256: ret,
        })

        expect(await smod.getConstructorUint256()).to.equal(1234)
      })
    })
  })
})
