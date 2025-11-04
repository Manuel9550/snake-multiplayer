import { defineConfig } from "vite";
import vitePluginRequire from 'vite-plugin-require-transform';

// https://vite.dev/config/
export default defineConfig({
  server: {
    port: 8070,
    open: true,
  },
  build: {
    commonjsOptions: { 
      transformMixedEsModules: true,
      include: [/node_modules/]
    }
  },
  define: {
    global: 'globalThis',
    exports: '{}',
    module: '{}',
  },
  plugins: [
    vitePluginRequire({
      fileRegex: /(.ts|.tsx|.vue|.js|.jsx)$/,
      importType: 'import'
    }),
  ],
  optimizeDeps: {
    include: ['google-protobuf']
  }
});
