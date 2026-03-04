<script lang="ts">
	import '../app.css';
	import Header from '$lib/components/Header.svelte';
	import Footer from '$lib/components/Footer.svelte';
	import { cartStore } from '$lib/stores/cart';
	import { onMount } from 'svelte';

	interface Props {
		children: import('svelte').Snippet;
	}
	let { children }: Props = $props();

	onMount(() => {
		cartStore.load();
	});
</script>

<div class="min-h-screen flex flex-col">
	<Header />
	<main class="flex-1">
		{@render children()}
	</main>
	<Footer />
</div>
