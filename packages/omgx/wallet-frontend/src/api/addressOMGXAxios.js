import axios from 'axios'
import { getAllNetworks } from 'util/masterConfig'
const nw = getAllNetworks()

export default function addressOMGXAxiosInstance(masterSystemConfig){
  
  if(masterSystemConfig === 'local') {
    return axios.create({baseURL: nw.local.addressOMGXUrl})
  } 
  else if (masterSystemConfig === 'rinkeby') {
    return axios.create({baseURL: nw.rinkeby.addressOMGXUrl})
  }
  else if (masterSystemConfig === 'mainnet') {
    return axios.create({baseURL: nw.mainnet.addressOMGXUrl})
  }

}