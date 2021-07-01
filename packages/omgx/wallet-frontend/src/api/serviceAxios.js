import axios from 'axios'

const _serviceAxiosInstance = axios.create({
  baseURL: process.env.REACT_APP_SERVICE_OPTIMISM_API_URL,
})

_serviceAxiosInstance.interceptors.request.use((config) => {
  config.headers['Accept'] = 'application/json'
  config.headers['Content-Type'] = 'application/json'
  return config
})

export default _serviceAxiosInstance
