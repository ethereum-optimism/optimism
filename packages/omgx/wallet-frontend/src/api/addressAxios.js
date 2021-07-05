import axios from 'axios'
import { getAllNetworks } from 'util/masterConfig'
const nw = getAllNetworks()

export default function addressAxiosInstance(masterSystemConfig){
  
  if(masterSystemConfig === 'local') {
    return axios.create({baseURL: nw.local.addressUrl})
  } 
  else if (masterSystemConfig === 'rinkeby') {
    return axios.create({baseURL: nw.rinkeby.addressUrl})
  }
  else if (masterSystemConfig === 'mainnet') {
    return axios.create({baseURL: nw.mainnet.addressUrl})
  }

}