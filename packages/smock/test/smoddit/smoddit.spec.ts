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

        await smod.smodify.put({
          _uint256: ret,
        })

        expect(await smod.getUint256()).to.equal(ret)
      })

      it('should be able to return a boolean', async () => {
        const ret = true

        await smod.smodify.put({
          _bool: ret,
        })

        expect(await smod.getBool()).to.equal(ret)
      })

      it('should be able to return an address', async () => {
        const ret = '0x558ba9b8d78713fbf768c1f8a584485B4003f43F'

        await smod.smodify.put({
          _address: ret,
        })

        expect(await smod.getAddress()).to.equal(ret)
      })

      // TODO: Need to solve this with a rewrite.
      it.skip('should be able to return an address in a packed storage slot', async () => {
        const ret = '0x558ba9b8d78713fbf768c1f8a584485B4003f43F'

        await smod.smodify.put({
          _packedB: ret,
        })

        expect(await smod.getPackedAddress()).to.equal(ret)
      })

      it('should be able to return a simple struct', async () => {
        const ret = {
          valueA: BigNumber.from(1234),
          valueB: true,
        }

        await smod.smodify.put({
          _SimpleStruct: ret,
        })

        const result = _.toPlainObject(await smod.getSimpleStruct())
        expect(result.valueA).to.deep.equal(ret.valueA)
        expect(result.valueB).to.deep.equal(ret.valueB)
      })

      it('should be able to return a simple uint256 => uint256 mapping value', async () => {
        const retKey = 1234
        const retVal = 5678

        await smod.smodify.put({
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

        await smod.smodify.put({
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

        await smod.smodify.put({
          _uint256: ret,
        })

        await smod.setUint256(4321)

        expect(await smod.getUint256()).to.equal(4321)
      })

      it('should return the set value if it was set in the constructor', async () => {
        const ret = 1234

        await smod.smodify.put({
          _constructorUint256: ret,
        })

        expect(await smod.getConstructorUint256()).to.equal(1234)
      })

      it('should be able to set values in a bytes5 => bool mapping', async () => {
        const key = '0x0000005678'
        const val = true

        await smod.smodify.put({
          _bytes5ToBoolMap: {
            [key]: val,
          },
        })

        expect(await smod.getBytes5ToBoolMapValue(key)).to.equal(val)
      })

      it('should be able to set values in a address => bool mapping', async () => {
        const key = '0x558ba9b8d78713fbf768c1f8a584485B4003f43F'
        const val = true

        await smod.smodify.put({
          _addressToBoolMap: {
            [key]: val,
          },
        })

        expect(await smod.getAddressToBoolMapValue(key)).to.equal(val)
      })

      it('should be able to set values in a address => address mapping', async () => {
        const key = '0x558ba9b8d78713fbf768c1f8a584485B4003f43F'
        const val = '0x063bE0Af9711a170BE4b07028b320C90705fec7C'

        await smod.smodify.put({
          _addressToAddressMap: {
            [key]: val,
          },
        })

        expect(await smod.getAddressToAddressMapValue(key)).to.equal(val)
      })
    })
  })
})
