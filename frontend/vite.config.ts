/// <reference types="vitest/config" />
import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

// The version shown in the UI is the single source of truth from the repo's
// project.yaml (bumped on every release). Read + inject it at build/dev-server
// start so a loaded SPA always reports the exact build it came from — no
// package.json duplication, no manual drift.
function resolveAppVersion(): string {
  try {
    const yaml = readFileSync(new URL('../project.yaml', import.meta.url), 'utf-8');
    return yaml.match(/^version:\s*(\S+)/m)?.[1] ?? 'dev';
  } catch {
    // project.yaml isn't reachable (e.g. an image build without it in context)
    // — degrade to 'dev' rather than failing the whole build.
    return 'dev';
  }
}
const appVersion = resolveAppVersion();

export default defineConfig({
  plugins: [vue()],
  // Anchor Vite/Vitest's cache to THIS directory (frontend/node_modules/.vite),
  // resolved from the config file rather than the cwd. Without this, a vitest
  // run launched with the repo root as its working directory (e.g. an IDE test
  // integration) writes its cache to <root>/node_modules/.vite, spawning a
  // stray node_modules at the repo root. Pinning it keeps the cache in-tree.
  cacheDir: fileURLToPath(new URL('./node_modules/.vite', import.meta.url)),
  define: {
    __APP_VERSION__: JSON.stringify(appVersion),
  },
  server: {
    host: '0.0.0.0',
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: false,
        ws: true,
      },
    },
  },
  test: {
    globals: true,
    environment: 'happy-dom',
    exclude: ['e2e/**', 'node_modules/**', 'dist/**'],
  },
});
