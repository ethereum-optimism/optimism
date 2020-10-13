import { expect } from '../../setup'

/* External Imports */
import { ethers } from '@nomiclabs/buidler'
import { Contract, BigNumber } from 'ethers'

export const runJsonTest = (contractName: string, json: any): void => {
  let contract: Contract
  before(async () => {
    contract = await (await ethers.getContractFactory(contractName)).deploy()
  })

  for (const [functionName, functionTests] of Object.entries(json)) {
    describe(functionName, () => {
      for (const [key, test] of Object.entries(functionTests)) {
        it(`should run test: ${key}`, async () => {
          if (test.revert) {
            await expect(contract.functions[functionName](...test.in)).to.be
              .reverted
          } else {
            expect(
              await contract.functions[functionName](...test.in)
            ).to.deep.equal(
              test.out.map((out: any) => {
                if (typeof out === 'number') {
                  return BigNumber.from(out)
                } else {
                  return out
                }
              })
            )
          }
        })
      }
    })
  }
}
