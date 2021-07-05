import axios from 'axios'

import { getBaseServices } from 'util/masterConfig'

/*
might need to be updated - not sure this makes sense for local?
*/

const _buyerAxiosInstance = axios.create({
  baseURL: getBaseServices().BUYER_OPTIMISM_API_URL,
})

_buyerAxiosInstance.interceptors.request.use((config) => {
  config.headers['Accept'] = 'application/json'
  config.headers['Content-Type'] = 'application/json'
  return config
})

export default _buyerAxiosInstance
