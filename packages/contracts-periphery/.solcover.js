module.exports = {
  skipFiles: [
    './test-libraries',
    './foundry-tests'
  ],
  mocha: {
    grep: "@skip-on-coverage",
    invert: true
  }
};
