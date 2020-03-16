/* External Imports */
import { ExpressHttpServer, getLogger, Logger } from '@eth-optimism/core-utils'
import { createMockProvider } from 'ethereum-waffle'
import { JsonRpcProvider, Web3Provider } from 'ethers/providers'

/* Internal Imports */
import { FullnodeRpcServer, DefaultWeb3Handler, TestWeb3Handler, DEFAULT_ETHNODE_GAS_LIMIT } from '../app'
const fs = require('fs')
const dns = require('dns')
const http = require("http")
const axios = require('axios').default;

const log: Logger = getLogger('rollup-fullnode')

/**
 * Runs a fullnode.
 * @param testFullnode Whether or not this is a test.
 */
export const runFullnode = async (
  testFullnode: boolean = false
): Promise<ExpressHttpServer> => {
  // TODO Get these from config
  const host = '0.0.0.0'
  const port = 8545

  var contents = fs.readFileSync('/etc/hosts', 'utf8');
  log.info(contents)
  log.info(`Starting fullnode in ${testFullnode ? 'TEST' : 'LIVE'} mode!`)
  const backend: Web3Provider = testFullnode ? createMockProvider({
      gasLimit: DEFAULT_ETHNODE_GAS_LIMIT,
      allowUnlimitedContractSize: true,
    }) : new Web3Provider(new JsonRpcProvider("http://geth:8546"));
  log.info("log.info")
  // dns.lookup('geth', (err, result) => {
  //   log.info(result)
  // })
//   const url = "http://geth:8546/";
//   http.get(url, res => {
//   res.setEncoding("utf8");
//   let body = "";
//   res.on("data", data => {
//     body += data;
//   });
//   res.on("end", () => {
//     body = JSON.parse(body);
//     log.info(body);
//   });
// });
  log.info("log.info 1")
await (new Promise(r => setTimeout(r, 5000)));
  log.info("log.info 2")
await (new Promise(r => {
axios({
  method: 'post',
  url: 'http://geth:8546/',
  data: {
    "jsonrpc": "2.0",
    "method": "net_version",
    "params":[],
    "id":67
  }
}).then((response) => {
  log.info("success")
  log.info(response);
  r()
}, (error) => {
  log.info("error")
  log.info(error);
  r()
});
}))
  log.info("log.info 3")
await (new Promise(r => setTimeout(r, 5000)));

  // log.info(await backend.send('net_version', []))
  const fullnodeHandler = testFullnode
    ? await TestWeb3Handler.create(backend)
    : await DefaultWeb3Handler.create(backend)
  const fullnodeRpcServer = new FullnodeRpcServer(fullnodeHandler, host, port)

  fullnodeRpcServer.listen()

  const baseUrl = `http://${host}:${port}`
  log.info(`Listening at ${baseUrl}`)

  return fullnodeRpcServer
}
