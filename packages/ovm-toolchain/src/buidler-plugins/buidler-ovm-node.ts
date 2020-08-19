import { extendEnvironment } from '@nomiclabs/buidler/config'
// tslint:disable-next-line
const VM = require('ethereumjs-vm').default

extendEnvironment(async (bre) => {
  await bre.network.provider['_init' as any]()

  const node = bre.network.provider['_node' as any]
  const vm = node['_vm' as any]
  const ovm = new VM({
    ...vm.opts,
    stateManager: vm.stateManager,
    emGasLimit: 100_000_000,
  })
  bre.network.provider['_node' as any]['_vm' as any] = ovm

  const vmTracer = bre.network.provider['_node' as any]['_vmTracer' as any]
  vmTracer['_vm' as any] = ovm
  vmTracer.enableTracing()
})
