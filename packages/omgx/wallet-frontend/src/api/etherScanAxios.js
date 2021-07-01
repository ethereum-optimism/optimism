import axios from 'axios'

const _etherScanInstance = axios.create({
  baseURL: `${process.env.REACT_APP_ETHERSCAN_URL}${process.env.REACT_APP_ETHERSCAN_API}`,
})

_etherScanInstance.interceptors.request.use((config) => {
  config.headers['Accept'] = 'application/json'
  config.headers['Content-Type'] = 'application/json'
  return config
})

export default _etherScanInstance
