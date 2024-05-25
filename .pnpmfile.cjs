// This function does not modify the lockfile. It asserts that packages do not use SSH
// when specifying git repository
function afterAllResolved(lockfile, context) {
  const pkgs = lockfile['packages'];
  for (const [pkg, entry] of Object.entries(pkgs)) {
    const repo = entry.resolution['repo'];
    if (repo !== undefined) {
      if (repo.startsWith('git@github.com')) {
        throw new Error(`Invalid git ssh specification found for package ${pkg}. Ensure sure that the dependencies do not reference SSH-based git repos before running installing them`);
      }
    }
  }
  return lockfile
}

module.exports = {
  hooks: {
    afterAllResolved
  }
}
