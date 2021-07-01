import axios from 'axios'

const _buyerAxiosInstance = axios.create({
  baseURL: process.env.REACT_APP_BUYER_OPTIMISM_API_URL,
})

_buyerAxiosInstance.interceptors.request.use((config) => {
  config.headers['Accept'] = 'application/json'
  config.headers['Content-Type'] = 'application/json'
  return config
})

export default _buyerAxiosInstance
