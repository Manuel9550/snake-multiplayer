import { defineConfig } from "vite";
import vitePluginCommonjs from "vite-plugin-commonjs";

// https://vite.dev/config/
export default defineConfig({
  server: {
    port: 8070,
    open: true,
  },
  build: {
    commonjsOptions: {
      transformMixedEsModules: true,
      include: [/node_modules/],
    },
    rollupOptions: {
      external: ["@xmldom/xmldom"],
    },
    target: "esnext",
  },
  define: {
    global: "globalThis",
  },
  plugins: [vitePluginCommonjs()],
  optimizeDeps: {
    include: ["google-protobuf", "protobufjs", "parse-svg-path"],
  },
});
