/* External Imports */
import { Contract, ContractFactory } from 'ethers'

export interface Smodify {
  put: (storage: any) => void
  set: (storage: any) => void
  check: (storage: any) => Promise<boolean>
  reset: () => void
}

export interface Smodded {
  [hash: string]: string
}

export interface ModifiableContract extends Contract {
  smodify: Smodify
  _smodded: Smodded
}

export interface ModifiableContractFactory extends ContractFactory {
  deploy: (...args: any[]) => Promise<ModifiableContract>
}
