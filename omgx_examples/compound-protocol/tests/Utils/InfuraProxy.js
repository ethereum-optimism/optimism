#!/usr/bin/env node --max-old-space-size=32768

// Very Dumb Proxy
//  - will slow you down for uncached responses
//  - will crash when it runs out of memory (although now runs with 32GB heap by default)
//  - does not handle block number in url gracefully yet
//
// Run the proxy server e.g.:
//  $ tests/Utils/InfuraProxy.js
//
// Replace in Web3Fork command, or use from curl e.g.:
//  $ curl -X POST localhost:1337/kovan/367b617143a94994b4f9c20e36c31839 \
//   --data "{\"jsonrpc\": \"2.0\", \"method\": \"eth_blockNumber\", \"params\": [], \"id\": 1}"
//  {"jsonrpc":"2.0","id":1,"result":"0x10675d7"}

const http = require('http');
const https = require('https');

const port = 1337;
const server = http.createServer(handle).listen(port);
const cache = {};

async function handle(req, res) {
  let data = ''
  req.on('data', (d) => data += d);
  req.on('end', async () => {
    const [network, project] = req.url.substr(1).split('/');
    const query = JSON.parse(data), id = query.id; delete query['id'];
    const key = `${req.url}:${JSON.stringify(query)}`;
    const hit = cache[key];
    if (hit) {
      console.log('Cache hit...', network, req.method, project, data);
      const reply = JSON.parse(hit); reply.id = id;
      res.writeHead(200, {'Content-Type': 'application/javascript'});
      res.end(JSON.stringify(reply));
    } else {
      try {
        console.log('Requesting...', network, req.method, project, data);
        const result = await fetch({
          host: `${network}.infura.io`,
          method: req.method,
          path: `/v3/${project}`,
          data: data
        });
        res.writeHead(200, {'Content-Type': 'application/javascript'});
        res.end(cache[key] = result);
      } catch (e) {
        console.error(e)
        res.writeHead(500, {'Content-Type': 'application/javascript'});
        res.end(JSON.stringify({error: 'request failed'}));
      }
    }
  });
}

async function fetch(options) {
  let data = ''
  return new Promise(
    (okay, fail) => {
      const req = https.request(options, (res) => {
        res.on('data', (d) => data += d);
        res.on('end', () => okay(data));
      });
      req.on('error', (e) => fail(e));
      req.end(options.data);
    });
}

