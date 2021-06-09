import { ethers } from 'hardhat'
import { Contract, Signer, BigNumber } from 'ethers'
import { expect } from 'chai'

export class GasMeasurement {
  GasMeasurementContract: Contract

  public async init(wallet: Signer) {
    this.GasMeasurementContract = await (
      await (await ethers.getContractFactory('Helper_GasMeasurer')).deploy()
    ).connect(wallet)
  }

  public async getGasCost(
    targetContract: Contract,
    methodName: string,
    methodArgs: Array<any> = []
  ): Promise<number> {
    const gasCost: number = await this.GasMeasurementContract.callStatic.measureCallGas(
      targetContract.address,
      targetContract.interface.encodeFunctionData(methodName, methodArgs)
    )

    return gasCost
  }
}

// Utility function which checks
export const expectApproxGasCost = (
  expectedGas: BigNumber,
  actualGas: BigNumber,
  maxPercentIncrease: number,
  maxPercentDecrease: number
): void => {

  expect(actualGas).to.be.lessThanOrEqual(
    expectedGas.mul(1 + maxPercentIncrease / 100)
  )
  expect(actualGas).to.be.greaterThanOrEqual(
    expectedGas.mul(1 - (maxPercentDecrease / 100))
  )
}
