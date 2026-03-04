import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),
	kit: {
		adapter: adapter({
			pages: '../internal/storefront/build',
			assets: '../internal/storefront/build',
			fallback: 'index.html'
		})
	}
};

export default config;
