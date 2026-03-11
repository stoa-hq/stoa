<script lang="ts">
	import { cartStore } from '$lib/stores/cart';
	import { productsApi, getTranslation, type Product } from '$lib/api/products';
	import { onMount } from 'svelte';
	import { t, locale } from 'svelte-i18n';
	import { fmt } from '$lib/i18n/formatters';

	interface CartItemData {
		id: string;
		product_id: string;
		variant_id?: string;
		quantity: number;
		product?: Product;
		sku: string;
		price_gross: number;
		image_url?: string;
	}

	let itemsData = $state<CartItemData[]>([]);
	let loading = $state(true);

	function variantLabel(product: Product, variantId: string, loc: string): string {
		const v = product.variants?.find((v) => v.id === variantId);
		if (!v?.options?.length) return v?.sku ?? '';
		return v.options
			.map((o) => o.translations?.find((t) => t.locale === loc)?.name ?? o.translations?.find((t) => t.locale === 'de-DE')?.name ?? o.translations?.[0]?.name ?? o.id)
			.join(', ');
	}

	// Derive enriched items reactively based on locale
	const enriched = $derived(itemsData.map((item) => {
		const loc = $locale ?? 'de-DE';
		const tr = item.product ? getTranslation(item.product, loc) : null;
		return {
			...item,
			name: tr?.name ?? item.product_id,
			variant_label: item.variant_id && item.product ? variantLabel(item.product, item.variant_id, loc) : ''
		};
	}));

	const total = $derived(enriched.reduce((s, i) => s + i.price_gross * i.quantity, 0));

	onMount(async () => {
		await cartStore.load();
		const items = $cartStore.items;
		if (items.length === 0) {
			loading = false;
			return;
		}

		try {
			const productIds = [...new Set(items.map((i) => i.product_id))];
			const products = await Promise.all(productIds.map((id) => productsApi.getById(id)));
			const productMap = new Map<string, Product>(
				products.flatMap((res) => res.data ? [[res.data.id, res.data]] : [])
			);

			itemsData = items.map((item) => {
				const product = productMap.get(item.product_id);
				const variant = product?.variants?.find((v) => v.id === item.variant_id);
				return {
					id: item.id,
					product_id: item.product_id,
					variant_id: item.variant_id,
					quantity: item.quantity,
					product,
					sku: variant?.sku ?? product?.sku ?? '',
					price_gross: variant?.price_gross ?? product?.price_gross ?? 0,
					image_url: product?.media?.find((m) => m.url)?.url
				};
			});
		} catch {
			itemsData = items.map((item) => ({
				id: item.id,
				product_id: item.product_id,
				variant_id: item.variant_id,
				quantity: item.quantity,
				sku: '',
				price_gross: 0
			}));
		} finally {
			loading = false;
		}
	});

	async function updateQty(itemId: string, qty: number) {
		await cartStore.updateQuantity(itemId, qty);
		itemsData = itemsData
			.map((i) => (i.id === itemId ? { ...i, quantity: qty } : i))
			.filter((i) => i.quantity > 0);
	}
</script>

<svelte:head>
	<title>{$t('cart.pageTitle')}</title>
</svelte:head>

<div class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<h1 class="text-2xl font-bold text-gray-900 mb-6">{$t('cart.title')}</h1>

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
			<p class="text-lg">{$t('cart.empty')}</p>
			<a href="/" class="btn btn-primary mt-4">{$t('cart.continueShopping')}</a>
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
					<p class="w-24 text-right font-semibold text-gray-900">{$fmt.price(item.price_gross * item.quantity)}</p>
					<button onclick={() => updateQty(item.id, 0)} class="text-gray-400 hover:text-red-500 transition-colors" aria-label={$t('cart.removeItem')}>
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
				<span>{$t('cart.subtotal')}</span>
				<span>{$fmt.price(total)}</span>
			</div>
			<div class="flex justify-between font-bold text-gray-900 text-lg mt-3 pt-3 border-t border-gray-200">
				<span>{$t('cart.total')}</span>
				<span>{$fmt.price(total)}</span>
			</div>
			<a href="/checkout" class="btn btn-primary btn-lg w-full mt-4">{$t('cart.checkout')}</a>
			<a href="/" class="btn btn-secondary w-full mt-2">{$t('cart.continueShopping')}</a>
		</div>
	{/if}
</div>
