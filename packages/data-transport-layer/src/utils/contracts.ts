/* Imports: External */
import { constants, Contract, Signer } from 'ethers'
import { BaseProvider } from '@ethersproject/providers'
import { getContractInterface } from '@eth-optimism/contracts'

export const loadContract = (
  name: string,
  address: string,
  provider: BaseProvider
): Contract => {
  return new Contract(address, getContractInterface(name) as any, provider)
}

export const loadProxyFromManager = async (
  name: string,
  proxy: string,
  Lib_AddressManager: Contract,
  provider: BaseProvider
): Promise<Contract> => {
  const address = await Lib_AddressManager.getAddress(proxy)

  if (address === constants.AddressZero) {
    throw new Error(
      `Lib_AddressManager does not have a record for a contract named: ${proxy}`
    )
  }

  return loadContract(name, address, provider)
}

export interface OptimismContracts {
  Lib_AddressManager: Contract
  StateCommitmentChain: Contract
  CanonicalTransactionChain: Contract
}

export const loadOptimismContracts = async (
  l1RpcProvider: BaseProvider,
  addressManagerAddress: string,
  signer?: Signer
): Promise<OptimismContracts> => {
  const Lib_AddressManager = loadContract(
    'Lib_AddressManager',
    addressManagerAddress,
    l1RpcProvider
  )

  const inputs = [
    {
      name: 'StateCommitmentChain',
      interface: 'IStateCommitmentChain',
    },
    {
      name: 'CanonicalTransactionChain',
      interface: 'ICanonicalTransactionChain',
    },
  ]

  const contracts = {}
  for (const input of inputs) {
    contracts[input.name] = await loadProxyFromManager(
      input.interface,
      input.name,
      Lib_AddressManager,
      l1RpcProvider
    )

    if (signer) {
      contracts[input.name] = contracts[input.name].connect(signer)
    }
  }

  contracts['Lib_AddressManager'] = Lib_AddressManager

  // TODO: sorry
  return contracts as OptimismContracts
}
