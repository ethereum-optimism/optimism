// tslint:disable-next-line
const VM = require('@eth-optimism/ethereumjs-vm').default
import { StateDump, getLatestStateDump } from '@eth-optimism/rollup-contracts'

interface OVMOpts {
  initialized: boolean
  emGasLimit: number
  dump: StateDump
  contracts: {
    ovmExecutionManager: any
    ovmStateManager: any
  }
}

export const makeOVM = (args: {
  evmOpts?: any
  ovmOpts?: Partial<OVMOpts>
}): any => {
  return new VM({
    ...args.evmOpts,
    ovmOpts: {
      dump: getLatestStateDump(),
      emGasLimit: 100_000_000,
      ...args.ovmOpts,
    },
  })
}
