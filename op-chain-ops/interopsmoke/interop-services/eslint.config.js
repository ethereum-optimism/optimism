const { fixupPluginRules } = require('@eslint/compat');
const tseslintPlugin = require('@typescript-eslint/eslint-plugin');
const tseslintParser = require('@typescript-eslint/parser');
const importPlugin = require('eslint-plugin-import');
const jsdocPlugin = require('eslint-plugin-jsdoc');
const reactPlugin = require('eslint-plugin-react');
const reactHooksPlugin = require('eslint-plugin-react-hooks');
const reactRefreshPlugin = require('eslint-plugin-react-refresh');
const simpleImportSortPlugin = require('eslint-plugin-simple-import-sort');
const globals = require('globals');

// Fix up plugins that might not be fully compatible with ESLint 9
const react = fixupPluginRules(reactPlugin);
const reactHooks = fixupPluginRules(reactHooksPlugin);
const reactRefresh = fixupPluginRules(reactRefreshPlugin);
const jsdoc = fixupPluginRules(jsdocPlugin);
const simpleImportSort = fixupPluginRules(simpleImportSortPlugin);
const importRule = fixupPluginRules(importPlugin);
const typescript = fixupPluginRules(tseslintPlugin);

  // JavaScript and TypeScript files
module.exports = {
  files: ['**/*.{js,jsx,ts,tsx}'],
  ignores: ['**/.storybook/**', '**/dist/**'],
  plugins: {
    'react': react,
    'react-hooks': reactHooks,
    'react-refresh': reactRefresh,
    'jsdoc': jsdoc,
    'simple-import-sort': simpleImportSort,
    'import': importRule,
    '@typescript-eslint': typescript
  },
  languageOptions: {
    parser: tseslintParser,
    globals: {
      ...globals.node,
      ...globals.browser,
      ...globals.jest,
      ...globals.mocha,
    },
    parserOptions: {
      ecmaFeatures: {
        jsx: true,
      },
    },
  },
  rules: {
    // TypeScript rules
    '@typescript-eslint/consistent-type-imports': 'error',
    '@typescript-eslint/member-ordering': 'warn',
    '@typescript-eslint/no-unused-vars': 'error',
    '@typescript-eslint/ban-ts-comment': [
      'error',
      {
        'ts-expect-error': 'allow-with-description',
        'ts-ignore': 'allow-with-description',
      },
    ],
    '@typescript-eslint/no-floating-promises': 'off',
    '@typescript-eslint/no-explicit-any': 'warn',
    '@typescript-eslint/array-type': ['error', { default: 'array-simple' }],

    // React rules
    'react-hooks/rules-of-hooks': 'error',
    'react-hooks/exhaustive-deps': 'error',

    // Import rules
    'simple-import-sort/imports': 'error',
    'simple-import-sort/exports': 'error',
    'import/first': 'error',
    'import/newline-after-import': 'error',
    'import/no-duplicates': 'error',

    // JSDoc rules
    'jsdoc/check-alignment': 'error',
    'jsdoc/check-indentation': 'error',
    'jsdoc/tag-lines': 'error',

    // General rules
    'no-console': 'warn',
    'prefer-const': 'error',
    'eol-last': 'error',
  },
  // Ignore patterns
  ignores: [
    'src/assets/**/*',
    '**/build/**/*',
    '**/dist/**/*',
    'dist',
    '**/typechain/**/*',
    'node_modules/**',
    'lib/**',
  ]
}
