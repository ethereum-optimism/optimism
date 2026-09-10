import { defineConfig } from 'eslint/config';

import rootConfig from '../../eslint.config.js';

// Create a new configuration that extends the root configuration
export default defineConfig([{
  ...rootConfig,
  ignores: [...rootConfig.ignores, 'ponder-env.d.ts'],
}]);