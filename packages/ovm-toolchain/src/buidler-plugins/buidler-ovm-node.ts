import { extendEnvironment } from '@nomiclabs/buidler/config'
// tslint:disable-next-line
const VM = require('ethereumjs-ovm').default
// tslint:disable-next-line
const BN = require('bn.js')

extendEnvironment(async (bre) => {
  const config: any = bre.config
  if (config.useOvm) {
    const gasLimit = 100_000_000

    // Initialize the provider so it has a VM instance ready to copy.
    await bre.network.provider['_init' as any]()
    const node = bre.network.provider['_node' as any]

    // Copy the options from the old VM instance and insert our new one.
    const vm = node['_vm' as any]
    const ovm = new VM({
      ...vm.opts,
      stateManager: vm.stateManager,
      emGasLimit: gasLimit,
    })
    node['_vm' as any] = ovm

    // Hijack the gas estimation function.
    node.estimateGas = async (txParams: any): Promise<{ estimation: any }> => {
      return {
        estimation: new BN(gasLimit),
      }
    }

    // Reset the vm tracer to avoid other buidler errors.
    const vmTracer = node['_vmTracer' as any]
    vmTracer['_vm' as any] = ovm
    vmTracer.enableTracing()
  }
})
