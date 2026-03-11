<script lang="ts">
	import { cartStore } from '$lib/stores/cart';
	import type { Product } from '$lib/api/products';
	import { getTranslation } from '$lib/api/products';
	import { t, locale } from 'svelte-i18n';
	import { fmt } from '$lib/i18n/formatters';

	interface Props {
		product: Product;
	}
	let { product }: Props = $props();

	const translation = $derived(getTranslation(product, $locale ?? 'de-DE'));
	const firstImage = $derived(product.media?.find((m) => m.url));
	let adding = $state(false);

	async function addToCart(e: MouseEvent) {
		e.preventDefault();
		e.stopPropagation();
		adding = true;
		try {
			await cartStore.add(product.id, 1);
		} finally {
			adding = false;
		}
	}
</script>

<a href="/products/{translation.slug}" class="group block">
	<div class="card overflow-hidden hover:shadow-md transition-shadow">
		<!-- Product image -->
		<div class="aspect-square bg-gradient-to-br from-gray-100 to-gray-200 flex items-center justify-center overflow-hidden">
			{#if firstImage?.url}
				<img src={firstImage.url} alt={translation.name} class="w-full h-full object-cover" />
			{:else}
				<svg xmlns="http://www.w3.org/2000/svg" class="h-16 w-16 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
				</svg>
			{/if}
		</div>

		<div class="p-4">
			<p class="text-xs text-gray-400 mb-1">{product.sku}</p>
			<h3 class="font-semibold text-gray-900 group-hover:text-primary-700 transition-colors line-clamp-2">
				{translation.name}
			</h3>
			<div class="mt-3 flex items-center justify-between">
				<span class="text-lg font-bold text-gray-900">{$fmt.price(product.price_gross)}</span>
				{#if product.has_variants}
					<span class="btn btn-secondary btn-sm">{$t('products.select')}</span>
				{:else}
					<button
						onclick={addToCart}
						disabled={adding || product.stock === 0}
						class="btn btn-primary btn-sm"
					>
						{#if adding}
							<svg class="animate-spin h-4 w-4" viewBox="0 0 24 24" fill="none">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"/>
							</svg>
						{:else if product.stock === 0}
							{$t('products.soldOut')}
						{:else}
							{$t('products.addToCart')}
						{/if}
					</button>
				{/if}
			</div>
			{#if product.stock > 0 && product.stock <= 5 && !product.has_variants}
				<p class="text-xs text-amber-600 mt-2">{$t('products.lowStock', { values: { count: product.stock } })}</p>
			{/if}
		</div>
	</div>
</a>
