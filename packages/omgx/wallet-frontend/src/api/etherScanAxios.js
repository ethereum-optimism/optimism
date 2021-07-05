import axios from 'axios'
import { getAllNetworks } from 'util/masterConfig'
const nw = getAllNetworks()

export default function etherScanInstance(masterSystemConfig, layer){
  
  let axiosInstance = null;
  
  if(masterSystemConfig === 'local') {
    return null //does not make sense on local
  } 
  else if (masterSystemConfig === 'rinkeby' && layer === 'L1') {
    axiosInstance = axios.create({
      baseURL: nw.rinkeby.L1.blockExplorer,
    })
  }
  else if (masterSystemConfig === 'rinkeby' && layer === 'L2') {
    axiosInstance = axios.create({
      baseURL: nw.rinkeby.L2.blockExplorer,
    })
  }
  else if (masterSystemConfig === 'mainnet' && layer === 'L1') {
    axiosInstance = axios.create({
      baseURL: nw.mainnet.L1.blockExplorer,
    })
  }
  else if (masterSystemConfig === 'mainnet' && layer === 'L2') {
    axiosInstance = axios.create({
      baseURL: nw.mainnet.L2.blockExplorer,
    })
  }

  axiosInstance.interceptors.request.use((config) => {
    config.headers['Accept'] = 'application/json'
    config.headers['Content-Type'] = 'application/json'
    return config
  })

  return axiosInstance
}
