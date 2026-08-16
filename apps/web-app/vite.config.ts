import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// No vite.config existed before this — the app relied entirely on Vite's
// zero-config defaults. Added now specifically for the `test` block, which
// Vitest needs to run component tests in a browser-like DOM (jsdom)
// instead of Node's default environment.
export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./src/test-setup.ts"],
  },
});
