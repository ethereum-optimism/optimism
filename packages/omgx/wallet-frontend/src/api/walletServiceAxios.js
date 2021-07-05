import axios from 'axios'
import { getBaseServices } from 'util/masterConfig'

export default function walletServiceAxiosInstance(masterSystemConfig){
  
  let axiosInstance = null;
  
  /*
   hack for now until we have a master endpoint for the wallet version
  */
  // if(masterSystemConfig === 'local') {
  //   return null //does not make sense on local
  // } 
  // else if (masterSystemConfig === 'rinkeby') {
  //   axiosInstance = axios.create({
  //     baseURL: nw.rinkeby.WALLET_SERVICE,
  //   })
  // }
  // else if (masterSystemConfig === 'mainnet') {
  //   axiosInstance = axios.create({
  //     baseURL: nw.mainnet.WALLET_SERVICE,
  //   })
  // }

  axiosInstance = axios.create({
    baseURL: getBaseServices().WALLET_SERVICE,
  })

  axiosInstance.interceptors.request.use((config) => {
    config.headers['Accept'] = 'application/json'
    config.headers['Content-Type'] = 'application/json'
    return config
  })

  return axiosInstance
}