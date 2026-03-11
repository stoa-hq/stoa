<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { productsApi, type Product } from '$lib/api/products';
	import ProductCard from '$lib/components/ProductCard.svelte';
	import { t } from 'svelte-i18n';

	let query = $state($page.url.searchParams.get('q') ?? '');
	let products = $state<Product[]>([]);
	let loading = $state(false);
	let searched = $state(false);

	async function search() {
		if (!query.trim()) return;
		loading = true;
		searched = true;
		try {
			const res = await productsApi.list({ search: query, limit: 24 });
			products = res.data?.items ?? [];
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		if (query) search();
	});

	function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		goto(`/search?q=${encodeURIComponent(query)}`);
		search();
	}
</script>

<svelte:head>
	<title>{query ? $t('search.pageTitleWithQuery', { values: { query } }) : $t('search.pageTitle')}</title>
</svelte:head>

<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<h1 class="text-2xl font-bold text-gray-900 mb-6">{$t('search.title')}</h1>

	<form onsubmit={handleSubmit} class="flex gap-3 max-w-xl mb-8">
		<input
			class="input flex-1"
			type="search"
			placeholder={$t('search.placeholder')}
			bind:value={query}
		/>
		<button type="submit" class="btn btn-primary">{$t('search.button')}</button>
	</form>

	{#if loading}
		<div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
			{#each Array(8) as _}
				<div class="card animate-pulse aspect-square bg-gray-100 rounded-xl"></div>
			{/each}
		</div>
	{:else if searched && products.length === 0}
		<p class="text-gray-500">{@html $t('search.noResults', { values: { query } })}</p>
	{:else if products.length > 0}
		<p class="text-sm text-gray-500 mb-4">{$t(products.length !== 1 ? 'search.resultCountPlural' : 'search.resultCount', { values: { count: products.length, query } })}</p>
		<div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
			{#each products as product (product.id)}
				<ProductCard {product} />
			{/each}
		</div>
	{/if}
</div>
