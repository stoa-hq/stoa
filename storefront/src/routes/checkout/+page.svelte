<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { cartStore } from '$lib/stores/cart';
	import { productsApi, getTranslation, type Product } from '$lib/api/products';
	import { ordersApi } from '$lib/api/orders';
	import { shippingApi, getShippingName, type ShippingMethod } from '$lib/api/shipping';
	import { paymentApi, getPaymentName, type PaymentMethod } from '$lib/api/payment';
	import { t, locale } from 'svelte-i18n';
	import { fmt } from '$lib/i18n/formatters';
	import PluginSlot from '$lib/components/PluginSlot.svelte';

	import { pluginStore } from '$lib/stores/plugins';

	let step = $state<'address' | 'shipping' | 'confirm'>('address');
	let loading = $state(true);
	let submitting = $state(false);
	let error = $state('');

	// Stripe / plugin payment flow state
	let awaitingPayment = $state(false);
	let awaitingProviderPayment = $state(false);
	let orderId = $state('');
	let orderNumber = $state('');
	let guestToken = $state('');
	let paymentReference = $state('');

	const hasPaymentPlugin = $derived(
		($pluginStore.extensions ?? []).some((e) => e.slot === 'storefront:checkout:payment')
	);

	const selectedPaymentMethod = $derived(
		paymentMethods.find((m) => m.id === selectedPayment)
	);

	const hasProvider = $derived(
		selectedPaymentMethod?.provider ? selectedPaymentMethod.provider !== '' : false
	);

	let shippingMethods = $state<ShippingMethod[]>([]);
	let paymentMethods = $state<PaymentMethod[]>([]);

	let selectedShipping = $state<string>('');
	let selectedPayment = $state<string>('');

	// Raw cart item data with product references for reactive translation
	interface LineItemData {
		id: string;
		product_id: string;
		variant_id?: string;
		quantity: number;
		product?: Product;
		sku: string;
		price_net: number;
		price_gross: number;
	}
	let lineItemsData = $state<LineItemData[]>([]);

	// Derive display names reactively based on locale
	const lineItems = $derived(lineItemsData.map((item) => {
		const loc = $locale ?? 'de-DE';
		const tr = item.product ? getTranslation(item.product, loc) : null;
		return {
			...item,
			name: tr?.name ?? $t('checkout.fallbackProductName')
		};
	}));

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

	let sameAsShipping = $state(true);

	let billingForm = $state({
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
			lineItemsData = items.map((item) => {
				const product = productMap.get(item.product_id);
				const variant = product?.variants?.find((v) => v.id === item.variant_id);
				return {
					id: item.id,
					product_id: item.product_id,
					variant_id: item.variant_id,
					quantity: item.quantity,
					product,
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
			error = $t('checkout.loadError');
		} finally {
			loading = false;
		}
	});

	function buildAddresses() {
		const shippingAddress = {
			first_name: form.first_name,
			last_name: form.last_name,
			street: form.street,
			city: form.city,
			zip: form.zip,
			country_code: form.country_code,
			email: form.email
		};

		const billingAddress = sameAsShipping ? shippingAddress : {
			first_name: billingForm.first_name,
			last_name: billingForm.last_name,
			street: billingForm.street,
			city: billingForm.city,
			zip: billingForm.zip,
			country_code: billingForm.country_code,
			email: billingForm.email
		};

		return { shippingAddress, billingAddress };
	}

	async function submitCheckout(ref?: string) {
		const { shippingAddress, billingAddress } = buildAddresses();

		const payload: Record<string, unknown> = {
			currency: 'EUR',
			billing_address: billingAddress,
			shipping_address: shippingAddress,
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
		};

		if (ref) {
			payload.payment_reference = ref;
		}

		const res = await ordersApi.checkout(payload);
		return res;
	}

	async function placeOrder() {
		submitting = true;
		error = '';
		try {
			// Provider-based payment (e.g. Stripe): show payment UI before creating order.
			if (hasPaymentPlugin && hasProvider) {
				awaitingProviderPayment = true;
				submitting = false;
				return;
			}

			// Manual payment method or no plugin: create order directly.
			const res = await submitCheckout();

			if (hasPaymentPlugin && !hasProvider && res.data?.id) {
				// Legacy flow: plugin payment after order creation (non-provider plugins).
				orderId = res.data.id;
				orderNumber = res.data.order_number ?? '';
				guestToken = res.data.guest_token ?? '';
				awaitingPayment = true;
			} else {
				cartStore.clear();
				goto(`/checkout/success?order=${res.data?.order_number ?? ''}`);
			}
		} catch (e: unknown) {
			error = (e as Error).message ?? $t('checkout.orderError');
		} finally {
			submitting = false;
		}
	}

	async function handlePluginEvent(e: CustomEvent) {
		const detail = e.detail;
		if (detail?.type === 'payment-success') {
			if (awaitingProviderPayment && detail.paymentIntentId) {
				// Pay-first flow: payment succeeded → now create the order with reference.
				paymentReference = detail.paymentIntentId;
				awaitingProviderPayment = false;
				submitting = true;
				error = '';
				try {
					const res = await submitCheckout(paymentReference);
					cartStore.clear();
					goto(`/checkout/success?order=${res.data?.order_number ?? ''}`);
				} catch (e: unknown) {
					error = (e as Error).message ?? $t('checkout.orderError');
				} finally {
					submitting = false;
				}
			} else {
				// Legacy flow: order already created, payment confirmed via webhook.
				cartStore.clear();
				goto(`/checkout/success?order=${orderNumber}`);
			}
		} else if (detail?.type === 'payment-error') {
			error = detail.message ?? $t('checkout.orderError');
		}
	}
</script>

<svelte:head>
	<title>{$t('checkout.pageTitle')}</title>
</svelte:head>

<div class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<h1 class="text-2xl font-bold text-gray-900 mb-6">{$t('checkout.title')}</h1>

	{#if loading}
		<div class="animate-pulse h-40 bg-gray-100 rounded-xl"></div>
	{:else}
		<div class="grid grid-cols-1 lg:grid-cols-5 gap-8">
			<!-- Form -->
			<div class="lg:col-span-3 space-y-6">
				<!-- Shipping Address -->
				<div class="card p-6">
					<h2 class="text-lg font-semibold text-gray-900 mb-4">{$t('checkout.shippingAddress')}</h2>
					<div class="grid grid-cols-2 gap-4">
						<div>
							<label class="label" for="first_name">{$t('checkout.firstName')}</label>
							<input class="input" id="first_name" bind:value={form.first_name} required />
						</div>
						<div>
							<label class="label" for="last_name">{$t('checkout.lastName')}</label>
							<input class="input" id="last_name" bind:value={form.last_name} required />
						</div>
						<div class="col-span-2">
							<label class="label" for="email">{$t('checkout.email')}</label>
							<input class="input" id="email" type="email" bind:value={form.email} required />
						</div>
						<div class="col-span-2">
							<label class="label" for="street">{$t('checkout.street')}</label>
							<input class="input" id="street" bind:value={form.street} required />
						</div>
						<div>
							<label class="label" for="zip">{$t('checkout.zip')}</label>
							<input class="input" id="zip" bind:value={form.zip} required />
						</div>
						<div>
							<label class="label" for="city">{$t('checkout.city')}</label>
							<input class="input" id="city" bind:value={form.city} required />
						</div>
					</div>
				</div>

				<!-- Billing Address -->
				<div class="card p-6">
					<label class="flex items-center gap-3 cursor-pointer select-none">
						<input
							type="checkbox"
							bind:checked={sameAsShipping}
							class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
						/>
						<span class="text-sm font-medium text-gray-700">{$t('checkout.sameAsBilling')}</span>
					</label>

					{#if !sameAsShipping}
						<div class="mt-5">
							<h2 class="text-lg font-semibold text-gray-900 mb-4">{$t('checkout.billingAddress')}</h2>
							<div class="grid grid-cols-2 gap-4">
								<div>
									<label class="label" for="billing_first_name">{$t('checkout.firstName')}</label>
									<input class="input" id="billing_first_name" bind:value={billingForm.first_name} required />
								</div>
								<div>
									<label class="label" for="billing_last_name">{$t('checkout.lastName')}</label>
									<input class="input" id="billing_last_name" bind:value={billingForm.last_name} required />
								</div>
								<div class="col-span-2">
									<label class="label" for="billing_email">{$t('checkout.email')}</label>
									<input class="input" id="billing_email" type="email" bind:value={billingForm.email} required />
								</div>
								<div class="col-span-2">
									<label class="label" for="billing_street">{$t('checkout.street')}</label>
									<input class="input" id="billing_street" bind:value={billingForm.street} required />
								</div>
								<div>
									<label class="label" for="billing_zip">{$t('checkout.zip')}</label>
									<input class="input" id="billing_zip" bind:value={billingForm.zip} required />
								</div>
								<div>
									<label class="label" for="billing_city">{$t('checkout.city')}</label>
									<input class="input" id="billing_city" bind:value={billingForm.city} required />
								</div>
							</div>
						</div>
					{/if}
				</div>

				<!-- Shipping -->
				{#if shippingMethods.length > 0}
					<div class="card p-6">
						<h2 class="text-lg font-semibold text-gray-900 mb-4">{$t('checkout.shippingMethod')}</h2>
						<div class="space-y-2">
							{#each shippingMethods as m}
								<label class="flex items-center gap-3 p-3 rounded-lg border-2 cursor-pointer transition-colors
									{selectedShipping === m.id ? 'border-primary-600 bg-primary-50' : 'border-gray-200 hover:border-gray-300'}">
									<input type="radio" bind:group={selectedShipping} value={m.id} class="text-primary-600" />
									<span class="flex-1 font-medium text-gray-900">{getShippingName(m, $locale ?? 'de-DE')}</span>
									<span class="font-bold text-gray-900">{$fmt.price(m.price_gross)}</span>
								</label>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Payment -->
				{#if paymentMethods.length > 0}
					<div class="card p-6">
						<h2 class="text-lg font-semibold text-gray-900 mb-4">{$t('checkout.paymentMethod')}</h2>
						<div class="space-y-2">
							{#each paymentMethods as m}
								<label class="flex items-center gap-3 p-3 rounded-lg border-2 cursor-pointer transition-colors
									{selectedPayment === m.id ? 'border-primary-600 bg-primary-50' : 'border-gray-200 hover:border-gray-300'}">
									<input type="radio" bind:group={selectedPayment} value={m.id} class="text-primary-600" />
									<span class="font-medium text-gray-900">{getPaymentName(m, $locale ?? 'de-DE')}</span>
								</label>
							{/each}
						</div>
					</div>
				{/if}

				<!-- Plugin payment: pre-order (pay first, then create order) -->
				{#if awaitingProviderPayment}
					<PluginSlot
						slot="storefront:checkout:payment"
						context={{ amount: total, currency: 'EUR', paymentMethodId: selectedPayment, email: (sameAsShipping ? form.email : billingForm.email) || form.email, billingDetails: buildAddresses().billingAddress }}
						onEvent={handlePluginEvent}
					/>
				{/if}

				<!-- Plugin payment: legacy post-order flow -->
				{#if awaitingPayment}
					<PluginSlot
						slot="storefront:checkout:payment"
						context={{ orderId, orderNumber, paymentMethodId: selectedPayment, amount: total, currency: 'EUR', guestToken, email: (sameAsShipping ? form.email : billingForm.email) || form.email, billingDetails: buildAddresses().billingAddress }}
						onEvent={handlePluginEvent}
					/>
				{/if}
			</div>

			<!-- Order summary -->
			<div class="lg:col-span-2">
				<div class="card p-6 sticky top-24">
					<h2 class="text-lg font-semibold text-gray-900 mb-4">{$t('checkout.orderSummary')}</h2>
					<ul class="space-y-3 text-sm">
						{#each lineItems as item}
							<li class="flex justify-between">
								<span class="text-gray-700">{item.name} <span class="text-gray-400">× {item.quantity}</span></span>
								<span class="font-medium">{$fmt.price(item.price_gross * item.quantity)}</span>
							</li>
						{/each}
					</ul>

					<div class="border-t border-gray-200 mt-4 pt-4 space-y-2 text-sm">
						<div class="flex justify-between text-gray-600">
							<span>{$t('checkout.shipping')}</span>
							<span>{shippingCost() > 0 ? $fmt.price(shippingCost()) : $t('checkout.shippingFree')}</span>
						</div>
						<div class="flex justify-between font-bold text-gray-900 text-base">
							<span>{$t('checkout.total')}</span>
							<span>{$fmt.price(total)}</span>
						</div>
					</div>

					{#if error}
						<p class="text-red-600 text-sm mt-3">{error}</p>
					{/if}

					<button
						onclick={placeOrder}
						disabled={submitting || awaitingPayment || awaitingProviderPayment || !form.first_name || !form.last_name || !form.street || !form.city || !form.zip
							|| (!sameAsShipping && (!billingForm.first_name || !billingForm.last_name || !billingForm.street || !billingForm.city || !billingForm.zip))}
						class="btn btn-primary btn-lg w-full mt-4"
					>
						{#if submitting}
							<svg class="animate-spin h-5 w-5" viewBox="0 0 24 24" fill="none">
								<circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"/>
								<path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z"/>
							</svg>
						{:else}
							{$t('checkout.placeOrder')}
						{/if}
					</button>
				</div>
			</div>
		</div>
	{/if}
</div>
