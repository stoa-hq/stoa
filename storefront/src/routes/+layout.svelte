<script lang="ts">
	import '../app.css';
	import '$lib/i18n';
	import Header from '$lib/components/Header.svelte';
	import Footer from '$lib/components/Footer.svelte';
	import { cartStore } from '$lib/stores/cart';
	import { authStore } from '$lib/stores/auth';
	import { loadPluginManifest } from '$lib/stores/plugins';
	import { loadStoreSettings, storeSettings } from '$lib/stores/settings';
	import { onMount } from 'svelte';
	import { isLoading } from 'svelte-i18n';

	interface Props {
		children: import('svelte').Snippet;
	}
	let { children }: Props = $props();

	onMount(() => {
		authStore.hydrate();
		cartStore.load();
		loadPluginManifest();
		loadStoreSettings();
	});

	$effect(() => {
		if ($storeSettings.favicon_url) {
			let link: HTMLLinkElement | null = document.querySelector("link[rel~='icon']");
			if (!link) {
				link = document.createElement('link');
				link.rel = 'icon';
				document.head.appendChild(link);
			}
			link.href = $storeSettings.favicon_url;
		}
	});
</script>

{#if $isLoading}
	<div class="min-h-screen flex items-center justify-center">
		<div class="animate-spin h-8 w-8 border-4 border-primary-600 border-t-transparent rounded-full"></div>
	</div>
{:else}
	<div class="min-h-screen flex flex-col">
		<Header />
		<main class="flex-1">
			{@render children()}
		</main>
		<Footer />
	</div>
{/if}
