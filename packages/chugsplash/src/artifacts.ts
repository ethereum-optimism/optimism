/* Imports: External */
import hre from 'hardhat'

// TODO
export type ContractArtifact = any
export type BuildInfo = any

export const getContractArtifact = async (
  name: string
): Promise<ContractArtifact> => {
  return hre.artifacts.readArtifactSync(name)
}

export const getBuildInfo = async (name: string): Promise<BuildInfo> => {
  return hre.artifacts.getBuildInfo(name)
}
