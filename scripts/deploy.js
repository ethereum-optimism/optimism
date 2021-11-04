const { deploy } = require("../scripts/lib")
const fs = require("fs")

async function main() {
  let [c, m, mm] = await deploy()
  let json = {
    "Challenge": c.address,
    "MIPS": m.address,
    "MIPSMemory": mm.address,
  }
  console.log("deployed", json)
  fs.writeFileSync("/tmp/cannon/deployed.json", JSON.stringify(json))
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error(error);
    process.exit(1);
  });
