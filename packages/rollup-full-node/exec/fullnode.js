const axios = require('axios').default;
// const fullnode = require("../build/src/exec/fullnode")

// fullnode.runFullnode()
setTimeout(() => {
  console.log("running now")
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
  console.log("success")
  console.log(response);
}, (error) => {
  console.log("error")
   if (error.response) {
        console.log(error.response.data);
   }
});
}, 5000);
