<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { productsApi, getTranslation, type Product, type ProductVariant } from '$lib/api/products';
	import { cartStore } from '$lib/stores/cart';
	import { t, locale } from 'svelte-i18n';
	import { fmt } from '$lib/i18n/formatters';

	function variantLabel(v: ProductVariant): string {
		const loc = $locale ?? 'de-DE';
		if (!v.options || v.options.length === 0) return v.sku;
		return v.options
			.map((o) => o.translations?.find((t) => t.locale === loc)?.name ?? o.translations?.find((t) => t.locale === 'de-DE')?.name ?? o.translations?.[0]?.name ?? o.id)
			.join(', ');
	}

	let product = $state<Product | null>(null);
	let loading = $state(true);
	let error = $state('');
	let selectedVariantId = $state<string | undefined>(undefined);
	let quantity = $state(1);
	let adding = $state(false);
	let added = $state(false);

	const slug = $derived($page.params.slug);
	const translation = $derived(product ? getTranslation(product, $locale ?? 'de-DE') : null);
	const activeVariants = $derived(product?.variants?.filter((v) => v.active) ?? []);
	const needsVariantSelection = $derived(activeVariants.length > 0 && !selectedVariantId);

	const displayPrice = $derived.by(() => {
		if (!product) return 0;
		if (selectedVariantId) {
			const v = activeVariants.find((v) => v.id === selectedVariantId);
			return v?.price_gross ?? product.price_gross;
		}
		return product.price_gross;
	});

	const productImages = $derived(product?.media?.filter((m) => m.url) ?? []);

	const inStock = $derived.by(() => {
		if (!product) return false;
		if (needsVariantSelection) return false;
		if (selectedVariantId) {
			const v = activeVariants.find((v) => v.id === selectedVariantId);
			return (v?.stock ?? 0) > 0;
		}
		return product.stock > 0;
	});

	onMount(async () => {
		try {
			const res = await productsApi.getBySlug(slug as string);
			product = res.data ?? null;
			if (!product) error = $t('productDetail.notFound');
		} catch {
			error = $t('productDetail.loadError');
		} finally {
			loading = false;
		}
	});

	async function addToCart() {
		if (!product || needsVariantSelection) return;
		adding = true;
		try {
			await cartStore.add(product.id, quantity, selectedVariantId);
			added = true;
			setTimeout(() => (added = false), 2000);
		} finally {
			adding = false;
		}
	}
</script>

<svelte:head>
	<title>{translation?.name ?? $t('productDetail.product')} – stoa</title>
</svelte:head>

<div class="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	{#if loading}
		<div class="animate-pulse space-y-4">
			<div class="h-8 bg-gray-200 rounded w-1/3"></div>
			<div class="h-64 bg-gray-200 rounded"></div>
		</div>
	{:else if error}
		<div class="text-center py-20">
			<p class="text-gray-500">{error}</p>
			<a href="/" class="btn btn-primary mt-4">{$t('productDetail.backToOverview')}</a>
		</div>
	{:else if product && translation}
		<!-- Breadcrumb -->
		<nav class="text-sm text-gray-500 mb-6">
			<a href="/" class="hover:text-gray-900">{$t('productDetail.breadcrumbProducts')}</a>
			<span class="mx-2">/</span>
			<span class="text-gray-900">{translation.name}</span>
		</nav>

		<div class="grid grid-cols-1 lg:grid-cols-2 gap-10">
			<!-- Image -->
			<div class="space-y-3">
				<div class="aspect-square bg-gradient-to-br from-gray-100 to-gray-200 rounded-2xl flex items-center justify-center overflow-hidden">
					{#if productImages[0]?.url}
						<img src={productImages[0].url} alt={translation.name} class="w-full h-full object-cover rounded-2xl" />
					{:else}
						<svg xmlns="http://www.w3.org/2000/svg" class="h-32 w-32 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
						</svg>
					{/if}
				</div>
				{#if productImages.length > 1}
					<div class="grid grid-cols-4 gap-2">
						{#each productImages as img, i}
							<div class="aspect-square rounded-lg overflow-hidden bg-gray-100">
								<img src={img.url} alt="{translation.name} {i + 1}" class="w-full h-full object-cover" />
							</div>
						{/each}
					</div>
				{/if}
			</div>

			<!-- Info -->
			<div class="flex flex-col">
				<p class="text-sm text-gray-400 mb-1">{product.sku}</p>
				<h1 class="text-3xl font-bold text-gray-900">{translation.name}</h1>

				<p class="text-3xl font-bold text-primary-700 mt-4">{$fmt.price(displayPrice)}</p>

				{#if translation.description}
					<div class="mt-4 text-gray-600 leading-relaxed prose prose-sm max-w-none">
						{translation.description}
					</div>
				{/if}

				<!-- Variants -->
				{#if activeVariants.length > 0}
					<div class="mt-6">
						<p class="text-sm font-medium text-gray-700 mb-2">{$t('productDetail.variant')}</p>
						<div class="flex flex-wrap gap-2">
							{#each activeVariants as v}
								{@const loc = $locale ?? 'de-DE'}
								<button
									onclick={() => selectedVariantId = v.id}
									class="px-3 py-1.5 rounded-lg text-sm border-2 transition-colors flex items-center gap-1.5
										{selectedVariantId === v.id ? 'border-primary-600 text-primary-700 bg-primary-50' : 'border-gray-200 text-gray-600 hover:border-gray-400'}"
								>
									{#if v.options && v.options.length > 0}
										{#each v.options as o, i}
											{#if i > 0}<span class="text-gray-300 text-xs">·</span>{/if}
											{#if o.color_hex}
												<span class="w-3 h-3 rounded-full border border-gray-300 shrink-0" style="background:{o.color_hex}"></span>
											{/if}
											<span>{o.translations?.find(t => t.locale === loc)?.name ?? o.translations?.find(t => t.locale === 'de-DE')?.name ?? o.translations?.[0]?.name ?? o.id}</span>
										{/each}
									{:else}
										{v.sku}
									{/if}
								</button>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Quantity -->
				<div class="mt-6">
					<p class="text-sm font-medium text-gray-700 mb-2">{$t('productDetail.quantity')}</p>
					<div class="flex items-center gap-3">
						<button
							onclick={() => quantity = Math.max(1, quantity - 1)}
							class="w-9 h-9 rounded-lg border border-gray-300 flex items-center justify-center text-gray-600 hover:bg-gray-50"
						>−</button>
						<span class="w-8 text-center font-medium">{quantity}</span>
						<button
							onclick={() => quantity = quantity + 1}
							class="w-9 h-9 rounded-lg border border-gray-300 flex items-center justify-center text-gray-600 hover:bg-gray-50"
						>+</button>
					</div>
				</div>

				<!-- Add to cart -->
				<button
					onclick={addToCart}
					disabled={adding || needsVariantSelection || !inStock}
					class="btn btn-primary btn-lg mt-6 w-full"
				>
					{#if added}
						{$t('productDetail.addedToCart')}
					{:else if adding}
						<svg class="animate-spin h-5 w-5" viewBox="0 0 24 24" fill="none">
							<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
							<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"/>
						</svg>
					{:else if needsVariantSelection}
						{$t('productDetail.selectVariant')}
					{:else if !inStock}
						{$t('productDetail.soldOut')}
					{:else}
						{$t('productDetail.addToCart')}
					{/if}
				</button>

				<!-- Stock info -->
				{#if selectedVariantId}
					{@const sv = activeVariants.find(v => v.id === selectedVariantId)}
					{#if sv && sv.stock > 0 && sv.stock <= 5}
						<p class="text-sm text-amber-600 mt-2 text-center">{$t('productDetail.lowStock', { values: { count: sv.stock } })}</p>
					{/if}
				{:else if !needsVariantSelection && inStock && product.stock <= 5}
					<p class="text-sm text-amber-600 mt-2 text-center">{$t('productDetail.lowStock', { values: { count: product.stock } })}</p>
				{/if}
			</div>
		</div>
	{/if}
</div>
