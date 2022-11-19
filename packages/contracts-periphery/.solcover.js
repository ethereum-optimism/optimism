module.exports = {
  skipFiles: [
    './test-libraries',
    './foundry-tests',
    './testing'
  ],
  mocha: {
    grep: "@skip-on-coverage",
    invert: true
  }
};
