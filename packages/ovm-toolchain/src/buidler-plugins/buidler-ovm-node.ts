import { extendEnvironment } from '@nomiclabs/buidler/config'
// tslint:disable-next-line
const VM = require('ethereumjs-vm').default

extendEnvironment(async (bre) => {
  // Initialize the provider so it has a VM instance ready to copy.
  await bre.network.provider['_init' as any]()
  const node = bre.network.provider['_node' as any]

  // Copy the options from the old VM instance and insert our new one.
  const vm = node['_vm' as any]
  const ovm = new VM({
    ...vm.opts,
    stateManager: vm.stateManager,
    emGasLimit: 100_000_000,
  })
  node['_vm' as any] = ovm

  // Reset the vm tracer to avoid other buidler errors.
  const vmTracer = node['_vmTracer' as any]
  vmTracer['_vm' as any] = ovm
  vmTracer.enableTracing()
})
