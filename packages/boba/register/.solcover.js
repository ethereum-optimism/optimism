module.exports = {
  skipFiles: [
    './test-helpers',
    './test-libraries'
  ],
  mocha: {
    grep: "@skip-on-coverage",
    invert: true
  }
};
