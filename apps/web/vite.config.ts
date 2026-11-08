import tailwindcss from "@tailwindcss/vite";
import { tanstackRouter } from "@tanstack/router-plugin/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vitest/config";

export default defineConfig({
	plugins: [
		tanstackRouter({ target: "react", autoCodeSplitting: true }),
		react(),
		tailwindcss(),
	],
	envDir: "../..",
	resolve: { tsconfigPaths: true },
	server: {
		proxy: {
			"/api": "http://localhost:4100",
			"/events": "http://localhost:4200",
		},
	},
	test: {
		environment: "jsdom",
	},
});
