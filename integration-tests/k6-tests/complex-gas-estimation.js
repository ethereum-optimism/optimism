import { sleep } from 'k6'

import { K6RpcProvider } from './utils/provider.js'

export default function() {
  const provider = new K6RpcProvider('http://localhost:8545')

  // Gas estimation for an ETH transfer.
  provider.send('eth_estimateGas', [{
    from: '0x1111111111111111111111111111111111111111',
    to: '0x4200000000000000000000000000000000000006',
    gasPrice: '0x0',
    value: '0x0',
    // transfer(0x2222222222222222222222222222222222222222, 0x1234)
    data: '0xa9059cbb00000000000000000000000022222222222222222222222222222222222222220000000000000000000000000000000000000000000000000000000000001234'
  }])

  sleep(1)
}
