import { fileURLToPath, URL } from 'node:url';
import { gzipSync } from 'node:zlib';
import { existsSync, readFileSync } from 'node:fs';
import { readFile, writeFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { defineConfig, loadEnv } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '');
  const backend = process.env.VITE_BACKEND_URL || env.VITE_BACKEND_URL || 'http://localhost:8100';

  const versionFile = resolve(__dirname, '../VERSION');
  let appVersion = 'dev';
  if (existsSync(versionFile)) {
    try {
      appVersion = readFileSync(versionFile, 'utf-8').trim();
    } catch {}
  }

  return {
    define: {
      __APP_VERSION__: JSON.stringify(appVersion),
    },
    plugins: [
      react(),
      tailwindcss(),
      {
        name: 'precompress-gzip',
        apply: 'build',
        enforce: 'post',
        async writeBundle(options, bundle) {
          for (const output of Object.values(bundle)) {
            if (!/\.(?:js|css|html|svg|json|txt)$/.test(output.fileName)) continue;
            const filePath = resolve(options.dir || 'dist', output.fileName);
            const original = await readFile(filePath);
            const compressed = gzipSync(original, { level: 9 });
            if (compressed.length >= original.length) continue;
            await writeFile(`${filePath}.gz`, compressed);
          }
        },
      },
    ],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    server: {
      host: '0.0.0.0',
      port: 6100,
      strictPort: true,
      proxy: {
        '/health': backend,
        '/api': backend,
        '/media': backend,
        '/swagger': backend,
      },
    },
    preview: {
      host: '0.0.0.0',
      port: 6100,
      strictPort: true,
    },
  };
});
