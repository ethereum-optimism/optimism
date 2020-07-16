const bre = require("@nomiclabs/buidler")

async function main() {
  console.log(bre)
}

main()
  .then(() => process.exit(0))
  .catch(error => {
    console.error(error)
    process.exit(1)  
  })