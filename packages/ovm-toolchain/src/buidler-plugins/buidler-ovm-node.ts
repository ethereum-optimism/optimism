// tslint:disable-next-line
const BN = require('bn.js')
import { extendEnvironment } from '@nomiclabs/buidler/config'

/* Internal Imports */
import { makeOVM } from '../utils/ovm'

extendEnvironment(async (bre) => {
  const config: any = bre.config
  config.startOvmNode = async (): Promise<void> => {
    const ovmGasLimit = config.ovmGasLimit || 100_000_000

    // Initialize the provider so it has a VM instance ready to copy.
    await bre.network.provider['_init' as any]()
    const node = bre.network.provider['_node' as any]

    // Copy the options from the old VM instance and create a new one.
    const vm = node['_vm' as any]
    const ovm = makeOVM(
      {
        evmOpts: {
          ...vm.opts,
          stateManager: vm.stateManager
        },
        ovmOpts: {
          emGasLimit: ovmGasLimit
        }
      }
    )

    // Initialize the OVM and replace the old VM.
    await ovm.init()
    node['_vm' as any] = ovm

    // Hijack the gas estimation function.
    node.estimateGas = async (): Promise<{ estimation: any }> => {
      return {
        estimation: new BN(ovmGasLimit),
      }
    }

    // Reset the vm tracer to avoid other buidler errors.
    const vmTracer = node['_vmTracer' as any]
    vmTracer['_vm' as any] = ovm
    vmTracer.enableTracing()
  }
})
