const getArtifact = (target) => {
  const buildFolder = (target === 'OVM') ? 'build-ovm' : 'build'
  const ERC20Artifact = require(`../${buildFolder}/ERC20.json`)
  return ERC20Artifact
}

module.exports = { getArtifact }