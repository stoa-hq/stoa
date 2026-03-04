import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		adapter: adapter({
			pages: '../internal/admin/build',
			assets: '../internal/admin/build',
			fallback: 'index.html' // SPA mode
		}),
		paths: {
			base: '/admin'
		}
	}
};

export default config;
