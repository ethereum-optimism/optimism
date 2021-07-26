const fs = require('fs');
const path = require('path');

let [_, _f, buildFile, contract] = process.argv;

if (!buildFile || !contract) {
  throw new Error(`builder.js <build_file> <contract>`);
}
if (!fs.existsSync(buildFile)) {
  throw new Error(`build_file: file not found`);
}
let buildRaw = fs.readFileSync(buildFile, 'utf8');
let build;

try {
  build = JSON.parse(buildRaw);
} catch (e) {
  throw new Error(`Error parsing build file: ${e.toString()}`);
}
if (!build.contracts) {
  throw new Error(`Invalid build file, missing contracts`);
}
let contractInfo = Object.entries(build.contracts).find(([k,v]) => k.split(':')[1] === contract);
if (!contractInfo) {
  throw new Error(`Build file does not contain info for ${contract}`);
}
let contractABI = JSON.parse(contractInfo[1].abi);

console.log(`export interface ${contract}Methods {`);
contractABI.forEach(abi => {
  if (abi.type === 'function') {
    function mapped(io) {
      let typeMap = {
        'address': 'string',
        'address[]': 'string[]',
        'uint256': 'number',
        'bool': 'boolean'
      };
      return typeMap[io.type] || io.type;
    };
    let name = abi.name;
    let args = abi.inputs.map((input) => {
      return `${input.name}: ${mapped(input)}`;
    }).join(', ');
    let returnType = abi.outputs.map((output) => {
      if (output.type == 'tuple' || output.type == 'tuple[]') {
        let res = output.components.map((c) => {
          return mapped(c);
        }).join(',');
        if (output.type == 'tuple[]') {
          return `[${res}][]`;
        } else {
          return `[${res}]`;
        }
      } else {
        return mapped(output);
      }
    }).join(',');
    let able = abi.constant ? 'Callable' : 'Sendable';
    console.log(`  ${name}(${args}): ${able}<${returnType}>;`);
  }
});
console.log("}");
