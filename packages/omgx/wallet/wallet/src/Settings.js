require('dotenv').config()

export const INFURA_ID = process.env.REACT_APP_INFURA_ID;

export const ETHERSCAN_URL = `https://api-rinkeby.etherscan.io/api?module=account&action=txlist&startblock=0&endblock=99999999&sort=asc&apikey=${process.env.REACT_APP_ETHERSCAN_API}`;
export const OMGX_WATCHER_URL = `https://api-watcher.rinkeby.omgx.network/`;
export const SELLER_OPTIMISM_API_URL = "https://pm7f0dp9ud.execute-api.us-west-1.amazonaws.com/prod/";
export const BUYER_OPTIMISM_API_URL = "https://n245h0ka3i.execute-api.us-west-1.amazonaws.com/prod/";
export const SERVICE_OPTIMISM_API_URL = "https://zlba6djrv6.execute-api.us-west-1.amazonaws.com/prod/";
export const WEBSOCKET_API_URL = "wss://d1cj5xnal2.execute-api.us-west-1.amazonaws.com/prod";