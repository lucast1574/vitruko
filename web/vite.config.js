import { defineConfig } from "vite";
export default defineConfig({
  server: {
    proxy: {
      "/console": "http://127.0.0.1:4000",
      "/v1": "http://127.0.0.1:4000",
    },
  },
  build: { outDir: "dist", emptyOutDir: true },
});
