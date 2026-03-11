<script lang="ts">
	import { cartStore } from '$lib/stores/cart';
	import { productsApi, getTranslation, type Product } from '$lib/api/products';
	import { formatPrice } from '$lib/utils';
	import { onMount } from 'svelte';

	const LOCALE = 'de-DE';

	interface EnrichedItem {
		id: string;
		product_id: string;
		variant_id?: string;
		quantity: number;
		name: string;
		variant_label: string;
		sku: string;
		price_gross: number;
		image_url?: string;
	}

	let enriched = $state<EnrichedItem[]>([]);
	let loading = $state(true);

	const total = $derived(enriched.reduce((s, i) => s + i.price_gross * i.quantity, 0));

	function variantLabel(product: Product, variantId: string): string {
		const v = product.variants?.find((v) => v.id === variantId);
		if (!v?.options?.length) return v?.sku ?? '';
		return v.options
			.map((o) => o.translations?.find((t) => t.locale === LOCALE)?.name ?? o.translations?.[0]?.name ?? o.id)
			.join(', ');
	}

	onMount(async () => {
		await cartStore.load();
		const items = $cartStore.items;
		if (items.length === 0) {
			loading = false;
			return;
		}

		try {
			// Produkte per ID laden (mit Variants + Options)
			const productIds = [...new Set(items.map((i) => i.product_id))];
			const products = await Promise.all(productIds.map((id) => productsApi.getById(id)));
			const productMap = new Map<string, Product>(
				products.flatMap((res) => res.data ? [[res.data.id, res.data]] : [])
			);

			enriched = items.map((item) => {
				const product = productMap.get(item.product_id);
				const t = product ? getTranslation(product) : null;
				const variant = product?.variants?.find((v) => v.id === item.variant_id);
				return {
					id: item.id,
					product_id: item.product_id,
					variant_id: item.variant_id,
					quantity: item.quantity,
					name: t?.name ?? item.product_id,
					variant_label: item.variant_id && product ? variantLabel(product, item.variant_id) : '',
					sku: variant?.sku ?? product?.sku ?? '',
					price_gross: variant?.price_gross ?? product?.price_gross ?? 0,
					image_url: product?.media?.find((m) => m.url)?.url
				};
			});
		} catch {
			enriched = items.map((item) => ({
				id: item.id,
				product_id: item.product_id,
				variant_id: item.variant_id,
				quantity: item.quantity,
				name: 'Produkt',
				variant_label: '',
				sku: '',
				price_gross: 0
			}));
		} finally {
			loading = false;
		}
	});

	async function updateQty(itemId: string, qty: number) {
		await cartStore.updateQuantity(itemId, qty);
		enriched = enriched
			.map((i) => (i.id === itemId ? { ...i, quantity: qty } : i))
			.filter((i) => i.quantity > 0);
	}
</script>

<svelte:head>
	<title>Warenkorb – stoa</title>
</svelte:head>

<div class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<h1 class="text-2xl font-bold text-gray-900 mb-6">Warenkorb</h1>

	{#if loading}
		<div class="animate-pulse space-y-4">
			{#each Array(3) as _}
				<div class="h-20 bg-gray-100 rounded-xl"></div>
			{/each}
		</div>
	{:else if enriched.length === 0}
		<div class="text-center py-20 text-gray-400">
			<svg xmlns="http://www.w3.org/2000/svg" class="h-16 w-16 mx-auto mb-4 text-gray-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M3 3h2l.4 2M7 13h10l4-8H5.4M7 13L5.4 5M7 13l-2.293 2.293c-.63.63-.184 1.707.707 1.707H17m0 0a2 2 0 100 4 2 2 0 000-4zm-8 2a2 2 0 11-4 0 2 2 0 014 0z" />
			</svg>
			<p class="text-lg">Dein Warenkorb ist leer.</p>
			<a href="/" class="btn btn-primary mt-4">Weiter einkaufen</a>
		</div>
	{:else}
		<div class="space-y-4">
			{#each enriched as item (item.id)}
				<div class="card p-4 flex items-center gap-4">
					<div class="w-16 h-16 bg-gray-100 rounded-lg flex-shrink-0 overflow-hidden">
						{#if item.image_url}
							<img src={item.image_url} alt={item.name} class="w-full h-full object-cover" />
						{/if}
					</div>
					<div class="flex-1 min-w-0">
						<p class="font-medium text-gray-900 truncate">{item.name}</p>
						{#if item.variant_label}
							<p class="text-sm text-gray-500">{item.variant_label}</p>
						{/if}
						{#if item.sku}
							<p class="text-xs text-gray-400">{item.sku}</p>
						{/if}
					</div>
					<div class="flex items-center gap-2">
						<button onclick={() => updateQty(item.id, item.quantity - 1)} class="w-8 h-8 rounded border border-gray-300 flex items-center justify-center text-gray-500 hover:bg-gray-50">−</button>
						<span class="w-6 text-center text-sm">{item.quantity}</span>
						<button onclick={() => updateQty(item.id, item.quantity + 1)} class="w-8 h-8 rounded border border-gray-300 flex items-center justify-center text-gray-500 hover:bg-gray-50">+</button>
					</div>
					<p class="w-24 text-right font-semibold text-gray-900">{formatPrice(item.price_gross * item.quantity)}</p>
					<button onclick={() => updateQty(item.id, 0)} class="text-gray-400 hover:text-red-500 transition-colors" aria-label="Artikel entfernen">
						<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
						</svg>
					</button>
				</div>
			{/each}
		</div>

		<!-- Summary -->
		<div class="mt-8 card p-6 max-w-sm ml-auto">
			<div class="flex justify-between text-gray-600 text-sm mb-2">
				<span>Zwischensumme</span>
				<span>{formatPrice(total)}</span>
			</div>
			<div class="flex justify-between font-bold text-gray-900 text-lg mt-3 pt-3 border-t border-gray-200">
				<span>Gesamt</span>
				<span>{formatPrice(total)}</span>
			</div>
			<a href="/checkout" class="btn btn-primary btn-lg w-full mt-4">Zur Kasse</a>
			<a href="/" class="btn btn-secondary w-full mt-2">Weiter einkaufen</a>
		</div>
	{/if}
</div>
