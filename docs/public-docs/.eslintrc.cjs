module.exports = {
  parserOptions: {
    ecmaVersion: 'latest'
  },
  extends: [
    'plugin:mdx/recommended',
  ],
  rules: {
    'semi': ['error', 'never']
  },
  overrides: [
    {
      files: ['pages/**/*.mdx'],
      extends: [
        'plugin:mdx/recommended'
      ],
      settings: {
        'mdx/code-blocks': true
      },
      rules: {
        'no-unused-expressions': 'off',
        'semi': ['error', 'never']
      }
    }
  ]
}
