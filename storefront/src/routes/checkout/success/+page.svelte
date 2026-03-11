<script lang="ts">
	import { page } from '$app/stores';
	import { t } from 'svelte-i18n';

	const orderNumber = $derived($page.url.searchParams.get('order') ?? '');
</script>

<svelte:head>
	<title>{$t('checkoutSuccess.pageTitle')}</title>
</svelte:head>

<div class="max-w-lg mx-auto px-4 py-20 text-center">
	<div class="w-20 h-20 rounded-full bg-green-100 flex items-center justify-center mx-auto mb-6">
		<svg xmlns="http://www.w3.org/2000/svg" class="h-10 w-10 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
		</svg>
	</div>
	<h1 class="text-2xl font-bold text-gray-900">{$t('checkoutSuccess.title')}</h1>
	{#if orderNumber}
		<p class="text-gray-500 mt-2">{$t('checkoutSuccess.orderNumber', { values: { number: orderNumber } })}</p>
	{/if}
	<p class="text-gray-500 mt-4">{$t('checkoutSuccess.confirmation')}</p>
	<div class="mt-8 flex flex-col sm:flex-row gap-3 justify-center">
		<a href="/account/orders" class="btn btn-primary">{$t('checkoutSuccess.myOrders')}</a>
		<a href="/" class="btn btn-secondary">{$t('checkoutSuccess.continueShopping')}</a>
	</div>
</div>
