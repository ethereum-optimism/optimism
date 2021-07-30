import axios from 'axios'

import { getBaseServices } from 'util/masterConfig'

const _ethGasStationAxiosInstance = axios.create({
  baseURL: getBaseServices().ETH_GAS_STATION_URL,
})

export default _ethGasStationAxiosInstance
