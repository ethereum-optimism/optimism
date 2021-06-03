import { sleep } from 'k6'

import { K6RpcProvider } from './utils/provider.js'

export default function() {
  const provider = new K6RpcProvider('http://localhost:8545')

  // Gas estimation for a simple call with no value and no data.
  provider.send('eth_estimateGas', [{
    from: '0x1111111111111111111111111111111111111111',
    to: '0x2222222222222222222222222222222222222222',
    gasPrice: '0x0',
    value: '0x0',
    data: '0x'
  }])

  sleep(1)
}
