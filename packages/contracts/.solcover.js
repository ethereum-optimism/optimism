module.exports = {
  skipFiles: [
    './test-helpers',
    './test-libraries',
    './optimistic-ethereum/mockOVM'
  ],
  mocha: {
    grep: "@skip-on-coverage",
    invert: true
  }
};
