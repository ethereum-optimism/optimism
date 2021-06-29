const getArtifact = (useL2) => {
  const buildFolder = useL2 ? 'build-ovm' : 'build'
  const ERC20Artifact = require(`../${buildFolder}/ERC20.json`)
  return ERC20Artifact
}

module.exports = { getArtifact }
