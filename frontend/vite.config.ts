import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vitest/config';

// In development the browser calls /api and /files on the same origin,
// like in production where Traefik routes them to the backend.
const backend = process.env.BACKEND_URL || 'http://localhost:8080';

export default defineConfig({
	plugins: [sveltekit()],
	server: {
		proxy: {
			'/api': backend,
			'/files': backend
		}
	},
	test: {
		include: ['src/**/*.test.ts'],
		environment: 'node'
	}
});
