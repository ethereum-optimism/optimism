import axios from 'axios'

const _sellerAxiosInstance = axios.create({
  baseURL: process.env.REACT_APP_SELLER_OPTIMISM_API_URL,
})

_sellerAxiosInstance.interceptors.request.use((config) => {
  config.headers['Accept'] = 'application/json'
  config.headers['Content-Type'] = 'application/json'
  return config
})

export default _sellerAxiosInstance
