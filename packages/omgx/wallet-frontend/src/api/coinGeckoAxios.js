import axios from 'axios'

import { getBaseServices } from 'util/masterConfig'

const _coinGeckoAxiosInstance = axios.create({
  baseURL: getBaseServices().COIN_GECKO_URL,
})

_coinGeckoAxiosInstance.interceptors.request.use((config) => {
  config.headers['Accept'] = 'application/json'
  config.headers['Content-Type'] = 'application/json'
  return config
})

export default _coinGeckoAxiosInstance
