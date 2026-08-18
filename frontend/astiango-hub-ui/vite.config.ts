import { resolve } from 'path';
import { defineConfig, UserConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import vueJsx from '@vitejs/plugin-vue-jsx';
import dynamicImport from 'vite-plugin-dynamic-import';
import { visualizer } from 'rollup-plugin-visualizer';

export default defineConfig(({ mode }) => {
  const config: UserConfig = {
    define: {
      // Compile vue-i18n messages to ASTs instead of generating functions with
      // `new Function()`, which is incompatible with the production CSP.
      __INTLIFY_JIT_COMPILATION__: true,
      __INTLIFY_DROP_MESSAGE_COMPILER__: false,
      __INTLIFY_PROD_DEVTOOLS__: false,
    },
    build: {
      rollupOptions: {
        output: {
          manualChunks(id, meta) {
            if (id.includes('node_modules')) {
              const arr = id.toString().split('node_modules/');
              const modulePath = arr[arr.length - 1];
              const packageName = modulePath?.split('/')?.[0];
              // Keep upstream dependency names out of public asset URLs.
              if (packageName === 'crawlab-vue3-dropzone') {
                return 'file-dropzone';
              }
              return packageName;
            }
            if (id.includes('three.min.js')) {
              return 'three';
            }
          },
        },
      },
    },
    optimizeDeps: {
      include: ['element-plus', 'axios', 'monaco-editor'],
    },
    resolve: {
      alias: {
        '@': resolve(__dirname, 'src'),
      },
      extensions: ['.js', '.ts', '.jsx', '.tsx', '.json', '.vue'],
    },
    // @ts-ignore
    plugins: [vue(), vueJsx(), dynamicImport()],
    server: {
      cors: true,
    },
  };

  if (mode === 'analyze') {
    // @ts-ignore
    config.plugins.push(visualizer({ open: true, gzipSize: true }));
  }

  return config;
});
