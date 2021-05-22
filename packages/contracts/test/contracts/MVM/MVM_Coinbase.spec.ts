//import { expect } from '../../setup'

/* External Imports */
import { ethers } from 'hardhat'
import { Signer, ContractFactory, Contract } from 'ethers'
// import { TransactionResponse } from '@ethersproject/abstract-provider'
// import { smockit, MockContract } from '@eth-optimism/smock'
import _ from 'lodash'

/* Internal Imports */
import {
  makeAddressManager,
  
} from '../../helpers'



describe('MVM_Coinbase', () => {
  let signer: Signer
  let sequencer: Signer
  before(async () => {
    ;[signer, sequencer] = await ethers.getSigners()
  })

  let AddressManager: Contract
  before(async () => {
    AddressManager = await makeAddressManager()
  })

  let Factory__MVM_Coinbase: ContractFactory
  before(async () => {
    Factory__MVM_Coinbase = await ethers.getContractFactory(
      'MVM_Coinbase'
      ,signer
    )
  })

  let MVM_Coinbase: Contract
  beforeEach(async () => {
    MVM_Coinbase = await Factory__MVM_Coinbase.deploy(
      AddressManager.address,
      AddressManager.address,
      10,
      "test",
      "tset2"
    )
  })


  describe('deposit', () => {
    const NON_ZERO_ADDRESS = sequencer.getAddress()
    
    it('should return the new chain id which incresed by one', async () => {
      
      //const depositAmount = utils.parseEther('1')
      const v1=await MVM_Coinbase.depositForTest(sequencer.getAddress(),200,{
        gasLimit: '0x100000',
        gasPrice: 0
      })
      const v12=await MVM_Coinbase.setTax(AddressManager.address,250,{
        gasLimit: '0x100000',
        gasPrice: 0
      })
      console.log(v12)
      const v13=await MVM_Coinbase.transferFrom2(NON_ZERO_ADDRESS,MVM_Coinbase.address,100,{
        gasLimit: '0x100000',
        gasPrice: 0
      })
      console.log(v13)
      //const v1=await MVM_Coinbase.balanceOf(MVM_Coinbase.signer)
      const v2=await MVM_Coinbase.balanceOf(NON_ZERO_ADDRESS)
      const v22=await MVM_Coinbase.balanceOf(AddressManager.address)
      console.log(v1)
      console.log(v2)
      console.log(v22)
    })
  })
})
