import axios from 'axios'

const _omgxWatcherAxiosInstance = axios.create({
  baseURL: process.env.REACT_APP_OMGX_WATCHER_URL,
})

_omgxWatcherAxiosInstance.interceptors.request.use((config) => {
  config.headers['Accept'] = 'application/json'
  config.headers['Content-Type'] = 'application/json'
  return config
})

export default _omgxWatcherAxiosInstance
