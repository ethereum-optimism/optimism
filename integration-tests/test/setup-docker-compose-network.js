const { DockerComposeNetwork } = require("./shared/docker-compose")


before(async () => {
  await new DockerComposeNetwork().up()
})