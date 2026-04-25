/// <reference types="vitest" />
import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json-summary'],
      include: ['src/lib/**/*.ts'],
      exclude: ['src/lib/types.ts', 'src/**/*.test.ts'],
      thresholds: {
        statements: 100,
        branches: 90,
        functions: 100,
        lines: 100
      }
    }
  }
});

