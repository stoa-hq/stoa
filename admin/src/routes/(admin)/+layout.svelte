<script lang="ts">
	import { authStore } from '$lib/stores/auth';
	import { loadPluginManifest } from '$lib/stores/plugins';
	import { goto } from '$app/navigation';
	import { base } from '$app/paths';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import Sidebar from '$lib/components/Layout/Sidebar.svelte';
	import Header from '$lib/components/Layout/Header.svelte';

	interface Props {
		children: import('svelte').Snippet;
	}
	let { children }: Props = $props();

	let ready = $state(false);

	onMount(() => {
		if (!authStore.isAuthenticated()) {
			goto(`${base}/login`);
		} else {
			ready = true;
			loadPluginManifest();
		}
	});
</script>

{#if ready}
	<div class="flex min-h-screen">
		<Sidebar />
		<div class="flex-1 flex flex-col min-w-0">
			<Header title={$page.data?.title ?? 'Admin'} />
			<main class="flex-1 p-6 overflow-auto">
				{@render children()}
			</main>
		</div>
	</div>
{/if}
