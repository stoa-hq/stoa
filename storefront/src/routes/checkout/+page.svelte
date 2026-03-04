<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { cartStore } from '$lib/stores/cart';
	import { productsApi, getTranslation } from '$lib/api/products';
	import { ordersApi } from '$lib/api/orders';
	import { shippingApi, getShippingName, type ShippingMethod } from '$lib/api/shipping';
	import { paymentApi, getPaymentName, type PaymentMethod } from '$lib/api/payment';
	import { formatPrice } from '$lib/utils';

	let step = $state<'address' | 'shipping' | 'confirm'>('address');
	let loading = $state(true);
	let submitting = $state(false);
	let error = $state('');

	let shippingMethods = $state<ShippingMethod[]>([]);
	let paymentMethods = $state<PaymentMethod[]>([]);

	let selectedShipping = $state<string>('');
	let selectedPayment = $state<string>('');

	// Enriched cart items for display + checkout submission
	interface LineItem {
		id: string;
		product_id: string;
		variant_id?: string;
		quantity: number;
		name: string;
		sku: string;
		price_net: number;
		price_gross: number;
	}
	let lineItems = $state<LineItem[]>([]);

	const subtotal = $derived(lineItems.reduce((s, i) => s + i.price_gross * i.quantity, 0));
	const shippingCost = $derived(() => {
		const m = shippingMethods.find((m) => m.id === selectedShipping);
		return m?.price_gross ?? 0;
	});
	const total = $derived(subtotal + shippingCost());

	let form = $state({
		first_name: '',
		last_name: '',
		street: '',
		city: '',
		zip: '',
		country_code: 'DE',
		email: ''
	});

	onMount(async () => {
		await cartStore.load();
		const items = $cartStore.items;
		if (items.length === 0) {
			goto('/cart');
			return;
		}

		try {
			const productIds = [...new Set(items.map((i) => i.product_id))];
			const [products, shippingRes, paymentRes] = await Promise.all([
				Promise.all(productIds.map((id) => productsApi.getById(id))),
				shippingApi.list(),
				paymentApi.list()
			]);

			const productMap = new Map(
				products.flatMap((res) => (res.data ? [[res.data.id, res.data]] : []))
			);
			lineItems = items.map((item) => {
				const product = productMap.get(item.product_id);
				const t = product ? getTranslation(product) : null;
				const variant = product?.variants?.find((v) => v.id === item.variant_id);
				return {
					id: item.id,
					product_id: item.product_id,
					variant_id: item.variant_id,
					quantity: item.quantity,
					name: t?.name ?? 'Produkt',
					sku: variant?.sku ?? product?.sku ?? '',
					price_net: variant?.price_net ?? product?.price_net ?? 0,
					price_gross: variant?.price_gross ?? product?.price_gross ?? 0
				};
			});

			shippingMethods = shippingRes.data ?? [];
			paymentMethods = paymentRes.data ?? [];
			if (shippingMethods.length > 0) selectedShipping = shippingMethods[0].id;
			if (paymentMethods.length > 0) selectedPayment = paymentMethods[0].id;
		} catch {
			error = 'Fehler beim Laden der Bestelldaten.';
		} finally {
			loading = false;
		}
	});

	async function placeOrder() {
		submitting = true;
		error = '';
		try {
			const address = {
				first_name: form.first_name,
				last_name: form.last_name,
				street: form.street,
				city: form.city,
				zip: form.zip,
				country_code: form.country_code,
				email: form.email
			};

			const res = await ordersApi.checkout({
				currency: 'EUR',
				billing_address: address,
				shipping_address: address,
				shipping_method_id: selectedShipping || undefined,
				payment_method_id: selectedPayment || undefined,
				items: lineItems.map((i) => ({
					product_id: i.product_id,
					variant_id: i.variant_id,
					sku: i.sku,
					name: i.name,
					quantity: i.quantity,
					unit_price_net: i.price_net,
					unit_price_gross: i.price_gross,
					tax_rate: 0
				}))
			});

			cartStore.clear();
			goto(`/checkout/success?order=${res.data?.order_number ?? ''}`);
		} catch (e: unknown) {
			error = (e as Error).message ?? 'Bestellung fehlgeschlagen.';
		} finally {
			submitting = false;
		}
	}
</script>

<svelte:head>
	<title>Kasse – stoa</title>
</svelte:head>

<div class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<h1 class="text-2xl font-bold text-gray-900 mb-6">Kasse</h1>

	{#if loading}
		<div class="animate-pulse h-40 bg-gray-100 rounded-xl"></div>
	{:else}
		<div class="grid grid-cols-1 lg:grid-cols-5 gap-8">
			<!-- Form -->
			<div class="lg:col-span-3 space-y-6">
				<!-- Address -->
				<div class="card p-6">
					<h2 class="text-lg font-semibold text-gray-900 mb-4">Lieferadresse</h2>
					<div class="grid grid-cols-2 gap-4">
						<div>
							<label class="label" for="first_name">Vorname</label>
							<input class="input" id="first_name" bind:value={form.first_name} required />
						</div>
						<div>
							<label class="label" for="last_name">Nachname</label>
							<input class="input" id="last_name" bind:value={form.last_name} required />
						</div>
						<div class="col-span-2">
							<label class="label" for="email">E-Mail</label>
							<input class="input" id="email" type="email" bind:value={form.email} required />
						</div>
						<div class="col-span-2">
							<label class="label" for="street">Straße & Hausnummer</label>
							<input class="input" id="street" bind:value={form.street} required />
						</div>
						<div>
							<label class="label" for="zip">PLZ</label>
							<input class="input" id="zip" bind:value={form.zip} required />
						</div>
						<div>
							<label class="label" for="city">Stadt</label>
							<input class="input" id="city" bind:value={form.city} required />
						</div>
					</div>
				</div>

				<!-- Shipping -->
				{#if shippingMethods.length > 0}
					<div class="card p-6">
						<h2 class="text-lg font-semibold text-gray-900 mb-4">Versandart</h2>
						<div class="space-y-2">
							{#each shippingMethods as m}
								<label class="flex items-center gap-3 p-3 rounded-lg border-2 cursor-pointer transition-colors
									{selectedShipping === m.id ? 'border-primary-600 bg-primary-50' : 'border-gray-200 hover:border-gray-300'}">
									<input type="radio" bind:group={selectedShipping} value={m.id} class="text-primary-600" />
									<span class="flex-1 font-medium text-gray-900">{getShippingName(m)}</span>
									<span class="font-bold text-gray-900">{formatPrice(m.price_gross)}</span>
								</label>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Payment -->
				{#if paymentMethods.length > 0}
					<div class="card p-6">
						<h2 class="text-lg font-semibold text-gray-900 mb-4">Zahlungsart</h2>
						<div class="space-y-2">
							{#each paymentMethods as m}
								<label class="flex items-center gap-3 p-3 rounded-lg border-2 cursor-pointer transition-colors
									{selectedPayment === m.id ? 'border-primary-600 bg-primary-50' : 'border-gray-200 hover:border-gray-300'}">
									<input type="radio" bind:group={selectedPayment} value={m.id} class="text-primary-600" />
									<span class="font-medium text-gray-900">{getPaymentName(m)}</span>
								</label>
							{/each}
						</div>
					</div>
				{/if}
			</div>

			<!-- Order summary -->
			<div class="lg:col-span-2">
				<div class="card p-6 sticky top-24">
					<h2 class="text-lg font-semibold text-gray-900 mb-4">Bestellübersicht</h2>
					<ul class="space-y-3 text-sm">
						{#each lineItems as item}
							<li class="flex justify-between">
								<span class="text-gray-700">{item.name} <span class="text-gray-400">× {item.quantity}</span></span>
								<span class="font-medium">{formatPrice(item.price_gross * item.quantity)}</span>
							</li>
						{/each}
					</ul>

					<div class="border-t border-gray-200 mt-4 pt-4 space-y-2 text-sm">
						<div class="flex justify-between text-gray-600">
							<span>Versand</span>
							<span>{shippingCost() > 0 ? formatPrice(shippingCost()) : 'kostenlos'}</span>
						</div>
						<div class="flex justify-between font-bold text-gray-900 text-base">
							<span>Gesamt</span>
							<span>{formatPrice(total)}</span>
						</div>
					</div>

					{#if error}
						<p class="text-red-600 text-sm mt-3">{error}</p>
					{/if}

					<button
						onclick={placeOrder}
						disabled={submitting || !form.first_name || !form.last_name || !form.street || !form.city || !form.zip}
						class="btn btn-primary btn-lg w-full mt-4"
					>
						{#if submitting}
							<svg class="animate-spin h-5 w-5" viewBox="0 0 24 24" fill="none">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"/>
							</svg>
						{:else}
							Jetzt kaufen
						{/if}
					</button>
				</div>
			</div>
		</div>
	{/if}
</div>
