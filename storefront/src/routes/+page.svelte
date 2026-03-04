<script lang="ts">
	import { onMount } from 'svelte';
	import { productsApi, type Product } from '$lib/api/products';
	import { categoriesApi, type Category, getCategoryName } from '$lib/api/categories';
	import ProductCard from '$lib/components/ProductCard.svelte';
	import Pagination from '$lib/components/Pagination.svelte';

	let products = $state<Product[]>([]);
	let categories = $state<Category[]>([]);
	let meta = $state<{ total: number; pages: number } | null>(null);
	let loading = $state(true);
	let page = $state(1);
	let selectedCategory = $state<string | null>(null);
	let search = $state('');

	async function load() {
		loading = true;
		try {
			const [prodRes, catRes] = await Promise.allSettled([
				productsApi.list({
					page,
					limit: 12,
					search: search || undefined,
					category_id: selectedCategory ?? undefined
				}),
				categories.length === 0 ? categoriesApi.tree() : Promise.resolve({ data: categories })
			]);
			if (prodRes.status === 'fulfilled') {
				products = prodRes.value.data?.items ?? [];
				if (prodRes.value.meta) meta = prodRes.value.meta;
			}
			if (catRes.status === 'fulfilled' && Array.isArray(catRes.value.data)) {
				categories = catRes.value.data;
			}
		} finally {
			loading = false;
		}
	}

	onMount(load);

	function selectCategory(id: string | null) {
		selectedCategory = id;
		page = 1;
		load();
	}

	let searchTimeout: ReturnType<typeof setTimeout>;
	function handleSearch() {
		clearTimeout(searchTimeout);
		searchTimeout = setTimeout(() => {
			page = 1;
			load();
		}, 400);
	}
</script>

<svelte:head>
	<title>Produkte – stoa</title>
</svelte:head>

<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<!-- Header -->
	<div class="mb-8">
		<h1 class="text-3xl font-bold text-gray-900">Produkte</h1>
		<p class="text-gray-500 mt-1">{meta?.total ?? 0} Artikel</p>
	</div>

	<div class="flex flex-col lg:flex-row gap-8">
		<!-- Sidebar: Categories + Search -->
		<aside class="lg:w-56 flex-shrink-0">
			<div class="mb-6">
				<input
					class="input"
					type="search"
					placeholder="Suchen..."
					bind:value={search}
					oninput={handleSearch}
				/>
			</div>

			{#if categories.length > 0}
				<div>
					<h2 class="text-xs font-semibold uppercase tracking-wider text-gray-400 mb-3">Kategorien</h2>
					<ul class="space-y-1">
						<li>
							<button
								onclick={() => selectCategory(null)}
								class="w-full text-left px-3 py-2 rounded-lg text-sm transition-colors
									{selectedCategory === null ? 'bg-primary-50 text-primary-700 font-medium' : 'text-gray-600 hover:bg-gray-100'}"
							>
								Alle Produkte
							</button>
						</li>
						{#each categories as cat}
							<li>
								<button
									onclick={() => selectCategory(cat.id)}
									class="w-full text-left px-3 py-2 rounded-lg text-sm transition-colors
										{selectedCategory === cat.id ? 'bg-primary-50 text-primary-700 font-medium' : 'text-gray-600 hover:bg-gray-100'}"
								>
									{getCategoryName(cat)}
								</button>
							</li>
						{/each}
					</ul>
				</div>
			{/if}
		</aside>

		<!-- Product grid -->
		<div class="flex-1">
			{#if loading}
				<div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
					{#each Array(8) as _}
						<div class="card animate-pulse aspect-square bg-gray-100 rounded-xl"></div>
					{/each}
				</div>
			{:else if products.length === 0}
				<div class="text-center py-20 text-gray-400">
					<svg xmlns="http://www.w3.org/2000/svg" class="h-16 w-16 mx-auto mb-4 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
					</svg>
					<p>Keine Produkte gefunden.</p>
				</div>
			{:else}
				<div class="grid grid-cols-2 md:grid-cols-3 xl:grid-cols-4 gap-4">
					{#each products as product (product.id)}
						<ProductCard {product} />
					{/each}
				</div>

				{#if meta && meta.pages > 1}
					<div class="mt-8">
						<Pagination currentPage={page} totalPages={meta.pages} onPageChange={(p) => { page = p; load(); }} />
					</div>
				{/if}
			{/if}
		</div>
	</div>
</div>
