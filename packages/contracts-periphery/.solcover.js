module.exports = {
  skipFiles: [
    './test-libraries',
  ],
  mocha: {
    grep: "@skip-on-coverage",
    invert: true
  }
};
