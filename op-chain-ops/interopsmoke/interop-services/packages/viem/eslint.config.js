import { defineConfig } from 'eslint/config';

import rootConfig from '../../eslint.config.js';

// Create a new configuration that extends the root configuration
export default defineConfig([{
  ...rootConfig,
  ignores: [...rootConfig.ignores, '**/dist/**'],
  rules: {
    ...rootConfig.rules,
    '@typescript-eslint/no-unused-vars': [
      'error',
      { varsIgnorePattern: '_', argsIgnorePattern: '_' },
    ],
  },
}]);
