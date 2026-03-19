<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { searchApi, type SearchResult } from '$lib/api/search';
	import { t, locale } from 'svelte-i18n';

	type FilterType = 'all' | 'product' | 'category';

	let query = $state($page.url.searchParams.get('q') ?? '');
	let results = $state<SearchResult[]>([]);
	let loading = $state(false);
	let searched = $state(false);
	let activeFilter = $state<FilterType>('all');
	let totalResults = $state(0);
	let currentPage = $state(1);
	let totalPages = $state(0);

	const limit = 24;

	async function search(pg = 1) {
		if (!query.trim()) return;
		loading = true;
		searched = true;
		currentPage = pg;
		try {
			const res = await searchApi.search({
				q: query,
				locale: $locale ?? 'de-DE',
				page: pg,
				limit,
				type: activeFilter !== 'all' ? activeFilter : undefined
			});
			results = res.data ?? [];
			totalResults = res.meta?.total ?? results.length;
			totalPages = res.meta?.pages ?? 1;
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

	function setFilter(filter: FilterType) {
		activeFilter = filter;
		search();
	}

	function goToPage(pg: number) {
		search(pg);
		window.scrollTo({ top: 0, behavior: 'smooth' });
	}

	function resultHref(result: SearchResult): string {
		if (result.type === 'product' && result.slug) {
			return `/products/${result.slug}`;
		}
		if (result.type === 'category') {
			return `/?category=${result.id}`;
		}
		return '#';
	}
</script>

<svelte:head>
	<title>{query ? $t('search.pageTitleWithQuery', { values: { query } }) : $t('search.pageTitle')}</title>
</svelte:head>

<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<h1 class="text-2xl font-bold text-gray-900 mb-6">{$t('search.title')}</h1>

	<form onsubmit={handleSubmit} class="flex gap-3 max-w-xl mb-6">
		<input
			class="input flex-1"
			type="search"
			placeholder={$t('search.placeholder')}
			bind:value={query}
		/>
		<button type="submit" class="btn btn-primary">{$t('search.button')}</button>
	</form>

	<!-- Type filter tabs -->
	{#if searched}
		<div class="flex gap-2 mb-8">
			{#each [
				{ key: 'all', label: $t('search.allTypes') },
				{ key: 'product', label: $t('search.products') },
				{ key: 'category', label: $t('search.categories') }
			] as tab (tab.key)}
				<button
					onclick={() => setFilter(tab.key as FilterType)}
					class="rounded-full px-4 py-1.5 text-sm font-medium transition-colors
						{activeFilter === tab.key
							? 'bg-primary-600 text-white'
							: 'bg-gray-100 text-gray-600 hover:bg-gray-200'}"
				>
					{tab.label}
				</button>
			{/each}
		</div>
	{/if}

	{#if loading}
		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
			{#each Array(8) as _}
				<div class="card animate-pulse rounded-xl">
					<div class="p-5">
						<div class="h-3 bg-gray-200 rounded w-16 mb-3"></div>
						<div class="h-5 bg-gray-200 rounded w-3/4 mb-2"></div>
						<div class="h-4 bg-gray-100 rounded w-full mb-1"></div>
						<div class="h-4 bg-gray-100 rounded w-2/3"></div>
					</div>
				</div>
			{/each}
		</div>
	{:else if searched && results.length === 0}
		<p class="text-gray-500">{$t('search.noResultsPrefix')} <strong>"{query}"</strong>{$t('search.noResultsSuffix')}</p>
	{:else if results.length > 0}
		<p class="text-sm text-gray-500 mb-4">
			{$t(totalResults !== 1 ? 'search.resultCountPlural' : 'search.resultCount', { values: { count: totalResults, query } })}
		</p>

		<div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
			{#each results as result (result.id + result.type)}
				<a href={resultHref(result)} class="group block">
					<div class="card overflow-hidden hover:shadow-md transition-shadow h-full">
						<div class="p-5">
							<span class="inline-block text-xs font-medium uppercase tracking-wide mb-2
								{result.type === 'product' ? 'text-primary-600' : 'text-amber-600'}">
								{result.type === 'product' ? $t('search.products') : $t('search.categories')}
							</span>
							<h3 class="font-semibold text-gray-900 group-hover:text-primary-700 transition-colors line-clamp-2 mb-1">
								{result.title}
							</h3>
							{#if result.description}
								<p class="text-sm text-gray-500 line-clamp-2">{result.description}</p>
							{/if}
						</div>
					</div>
				</a>
			{/each}
		</div>

		<!-- Pagination -->
		{#if totalPages > 1}
			<nav class="mt-8 flex justify-center gap-2">
				<button
					onclick={() => goToPage(currentPage - 1)}
					disabled={currentPage <= 1}
					class="btn btn-secondary"
				>
					&larr;
				</button>
				{#each Array(totalPages) as _, i}
					{@const pg = i + 1}
					{#if pg === 1 || pg === totalPages || (pg >= currentPage - 1 && pg <= currentPage + 1)}
						<button
							onclick={() => goToPage(pg)}
							class="rounded-lg px-3 py-2 text-sm font-medium transition-colors
								{pg === currentPage
									? 'bg-primary-600 text-white'
									: 'bg-white text-gray-700 border border-gray-300 hover:bg-gray-50'}"
						>
							{pg}
						</button>
					{:else if pg === currentPage - 2 || pg === currentPage + 2}
						<span class="px-1 py-2 text-gray-400">&hellip;</span>
					{/if}
				{/each}
				<button
					onclick={() => goToPage(currentPage + 1)}
					disabled={currentPage >= totalPages}
					class="btn btn-secondary"
				>
					&rarr;
				</button>
			</nav>
		{/if}
	{/if}
</div>
