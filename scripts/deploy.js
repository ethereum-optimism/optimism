const { deploy } = require("../scripts/lib")

async function main() {
  var [c, m, mm] = await deploy()
  console.log("deployed at", c.address, m.address, mm.address)
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
