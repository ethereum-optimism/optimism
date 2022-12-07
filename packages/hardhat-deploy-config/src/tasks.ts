import { task } from 'hardhat/config'

task(
  'cfg-as-json',
  'prints the config for the given network as JSON'
).setAction(async (args, hre) => {
  console.log(JSON.stringify(hre.deployConfig))
})
