import hre from 'hardhat'

export const deploy = async (
  name: string,
  opts?: {
    args?: any[]
    signer?: any
  }
) => {
  const factory = await hre.ethers.getContractFactory(name, opts?.signer)
  return factory.deploy(...(opts?.args || []))
}
