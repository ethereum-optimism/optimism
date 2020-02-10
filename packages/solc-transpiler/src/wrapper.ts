/* Internal Imports */
import * as packageJson from '../package.json'
import { compile } from './compiler'

export const wrapper = {
  version: () => packageJson.version,
  semver: () => packageJson.version,
  license: () => packageJson.version,
  compile,
  compileStandard: compile,
  compileStandardWrapper: compile,
}

/* SOLC JS Interface

{
  version: version,
  semver: versionToSemver,
  license: license,
  lowlevel: {
    compileSingle: compileJSON,
    compileMulti: compileJSONMulti,
    compileCallback: compileJSONCallback,
    compileStandard: compileStandard
  },
  features: {
    legacySingleInput: compileJSON !== null,
    multipleInputs: compileJSONMulti !== null || compileStandard !== null,
    importCallback: compileJSONCallback !== null || compileStandard !== null,
    nativeStandardJSON: compileStandard !== null
  },
  compile: compileStandardWrapper,
  // Temporary wrappers to minimise breaking with other projects.
  // NOTE: to be removed in 0.5.2
  compileStandard: compileStandardWrapper,
  compileStandardWrapper: compileStandardWrapper,
  // Loads the compiler of the given version from the github repository
  // instead of from the local filesystem.
  loadRemoteVersion: function (versionString, cb) {
    var mem = new MemoryStream(null, {readable: false});
    var url = 'https://ethereum.github.io/solc-bin/bin/soljson-' + versionString + '.js';
    https.get(url, function (response) {
      if (response.statusCode !== 200) {
        cb(new Error('Error retrieving binary: ' + response.statusMessage));
      } else {
        response.pipe(mem);
        response.on('end', function () {
          cb(null, setupMethods(requireFromString(mem.toString(), 'soljson-' + versionString + '.js')));
        });
      }
    }).on('error', function (error) {
      cb(error);
    });
  },
  // Use this if you want to add wrapper functions around the pure module.
  setupMethods: setupMethods
};

 */
