import { expect } from 'chai'
import { ethers } from 'hardhat'
import { BigNumber, Contract, Signer } from 'ethers'

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
    const gasCost: number =
      await this.GasMeasurementContract.callStatic.measureCallGas(
        targetContract.address,
        targetContract.interface.encodeFunctionData(methodName, methodArgs)
      )

    return gasCost
  }
}

interface percentDeviationRange {
  upperPercentDeviation: number
  lowerPercentDeviation?: number
}

export const expectApprox = (
  actual: BigNumber | number,
  target: BigNumber | number,
  { upperPercentDeviation, lowerPercentDeviation = 100 }: percentDeviationRange
): void => {
  actual = BigNumber.from(actual)
  target = BigNumber.from(target)

  const validDeviations =
    upperPercentDeviation >= 0 &&
    upperPercentDeviation <= 100 &&
    lowerPercentDeviation >= 0 &&
    lowerPercentDeviation <= 100
  if (!validDeviations) {
    throw new Error(
      'Upper and lower deviation percentage arguments should be between 0 and 100'
    )
  }
  const upper = target.mul(100 + upperPercentDeviation).div(100)
  const lower = target.mul(100 - lowerPercentDeviation).div(100)

  expect(
    actual.lte(upper),
    `Actual value is more than ${upperPercentDeviation}% greater than target`
  ).to.be.true
  expect(
    actual.gte(lower),
    `Actual value is more than ${lowerPercentDeviation}% less than target`
  ).to.be.true
}
