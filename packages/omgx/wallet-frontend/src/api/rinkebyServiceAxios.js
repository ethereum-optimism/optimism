import axios from 'axios'

const _rinkebyServiceAxiosInstance = axios.create({
  baseURL: process.env.REACT_APP_WALLET_SERVICE,
})

_rinkebyServiceAxiosInstance.interceptors.request.use((config) => {
  config.headers['Accept'] = 'application/json'
  config.headers['Content-Type'] = 'application/json'
  return config
})

export default _rinkebyServiceAxiosInstance
