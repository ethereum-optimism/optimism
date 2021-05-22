import { expect } from '../../setup'

/* External Imports */
import { ethers } from 'hardhat'
import { ContractFactory, Contract} from 'ethers'
import _ from 'lodash'

/* Internal Imports */
import {
  makeAddressManager,
} from '../../helpers'
import { keccak256 } from 'ethers/lib/utils'


const applyL2ChainId = async (contact) => {
    const methodId = keccak256(Buffer.from('applyL2ChainId()')).slice(2, 10);
    return contact.signer.sendTransaction({
        to: contact.address,
        data: '0x' + methodId,
    });
};

describe('MVM_L2ChainManagerOnL1', () => {
  before(async () => {
  })

  let AddressManager: Contract
  before(async () => {
    AddressManager = await makeAddressManager()
  })

  let Factory__MVM_L2ChainManagerOnL1: ContractFactory
  before(async () => {
    Factory__MVM_L2ChainManagerOnL1 = await ethers.getContractFactory(
      'MVM_L2ChainManagerOnL1'
    )
  })

  let MVM_L2ChainManagerOnL1: Contract
  beforeEach(async () => {
    MVM_L2ChainManagerOnL1 = await Factory__MVM_L2ChainManagerOnL1.deploy(
      AddressManager.address,
      AddressManager.address
    )
  })

  describe('applyL2ChainId', () => {
    
    it('should return the new chain id which incresed by one', async () => {
      const t=await applyL2ChainId(MVM_L2ChainManagerOnL1);
      await t.wait();
      const id=await MVM_L2ChainManagerOnL1.l2chainIds(MVM_L2ChainManagerOnL1.signer.getAddress())
      expect(
       id
      ).to.gte(1)
    })
  })
})
