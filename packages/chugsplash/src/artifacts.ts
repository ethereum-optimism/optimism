// TODO
export type ContractArtifact = any
export type BuildInfo = any

export const getContractArtifact = async (
  name: string
): Promise<ContractArtifact> => {
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  const hre = require('hardhat')
  return hre.artifacts.readArtifactSync(name)
}

export const getBuildInfo = async (name: string): Promise<BuildInfo> => {
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  const hre = require('hardhat')
  return hre.artifacts.getBuildInfo(name)
}
